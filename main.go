package main

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/uptrace/bun"

	_ "github.com/lib/pq"
)

//go:embed static
var static embed.FS

//go:embed templates
var templates embed.FS

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func formatDateTime(d time.Time) string {
	if d.IsZero() {
		return ""
	}
	return d.Format("2006-01-02 15:04")
}

type Todo struct {
	bun.BaseModel `bun:"table:todos,alias:t"`

	ID        int64     `bun:"id,pk,autoincrement"`
	Content   string    `bun:"content,notnull"`
	Done      bool      `bun:"done"`
	Until     time.Time `bun:"until,nullzero"`
	CreatedAt time.Time
	UpdatedAt time.Time `bun:",nullzero"`
	DeletedAt time.Time `bun:",soft_delete,nullzero"`
}

type Data struct {
	Todos    []Todo
	Errors   []error
	Messages []string
}

func bindUntil(todo *Todo) func([]string) []error {
	return func(values []string) []error {
		if len(values) == 0 || values[0] == "" {
			return nil
		}
		dt, err := time.Parse("2006-01-02T15:04 MST", values[0]+" JST")
		if err != nil {
			return []error{echo.NewBindingError("until", values[0:1], "failed to decode time", err)}
		}
		todo.Until = dt
		return nil
	}
}

var e *echo.Echo

func main() {

	db, closeDB, err := setupDB()
	if err != nil {
		log.Fatal(err)
	}
	defer closeDB()

	e = echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Logger.SetOutput(os.Stdout)

	e.Renderer = &Template{
		templates: template.Must(template.New("").
			Funcs(template.FuncMap{
				"FormatDateTime": formatDateTime,
			}).ParseFS(templates, "templates/*")),
	}

	e.GET("/", func(c echo.Context) error {
		// get a message from cookie
		message := GetCookie(c, MESSAGE)
		if message != "" {
			// clear the cookie cuz it's a one-time message
			ClearCookie(c, MESSAGE)
		}

		var todos []Todo
		ctx := context.Background()
		err = db.NewSelect().Model(&todos).Order("created_at").Scan(ctx)
		if err != nil {
			e.Logger.Error(err)
			return c.Render(http.StatusBadRequest, "index", Data{
				Errors: []error{errors.New("Cannot get todos")},
			})
		}
		return c.Render(http.StatusOK, "index", Data{Todos: todos, Messages: []string{message}})
	})

	e.POST("/", func(c echo.Context) error {
		var todo Todo
		// bind form params to todo's fields
		errs := echo.FormFieldBinder(c).
			Int64("id", &todo.ID).
			String("content", &todo.Content).
			Bool("done", &todo.Done).
			CustomFunc("until", bindUntil(&todo)).
			BindErrors()
		if errs != nil {
			e.Logger.Error(errs)
			return c.Render(http.StatusBadRequest, "index", Data{Errors: errs})
		} else if todo.ID == 0 {
			// register a ToDo task
			ctx := context.Background()
			if todo.Content == "" {
				err = errors.New("Todo not found")
			} else {
				todo.CreatedAt = time.Now()
				_, err = db.NewInsert().Model(&todo).Exec(ctx)
				if err != nil {
					e.Logger.Error(err)
					err = errors.New("Cannot update")
				}
			}
		} else {
			ctx := context.Background()
			if c.FormValue("delete") != "" {
				// delete a ToDo task
				_, err = db.NewDelete().Model(&todo).Where("id = ?", todo.ID).Exec(ctx)
			} else {
				// update a ToDo task
				var original Todo
				err = db.NewSelect().Model(&original).Where("id = ?", todo.ID).Scan(ctx)
				if err == nil {
					original.Done = todo.Done
					original.UpdatedAt = time.Now()
					_, err = db.NewUpdate().Model(&original).Where("id = ?", todo.ID).Exec(ctx)
				}
			}
			if err != nil {
				e.Logger.Error(err)
				err = errors.New("Cannot update")
			}
		}
		if err != nil {
			return c.Render(http.StatusBadRequest, "index", Data{Errors: []error{err}})
		}
		return c.Redirect(http.StatusFound, "/")
	})

	e.POST("/notify", notifyToDoByMail)

	staticFs, err := fs.Sub(static, "static")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FileSystem(http.FS(staticFs)))
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", fileServer)))

	e.Logger.Fatal(e.Start(":8989"))
}
