package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"

	"github.com/labstack/echo/v4"
)

func notifyToDoByMail(c echo.Context) error {

	var todos []Todo
	ctx := context.Background()
	err := db.NewSelect().Model(&todos).Order("created_at").Where("until is not null").Where("done is false").Scan(ctx)
	if err != nil {
		e.Logger.Error(err)
		return c.Render(http.StatusInternalServerError, "index",
			Data{
				Errors: []error{err},
			})
	}
	if len(todos) == 0 {
		e.Logger.Info("no todos to notify")
		return c.Redirect(http.StatusSeeOther, "/")
	}

	from := mail.Address{Name: "TODO Reminder", Address: os.Getenv("MAIL_FROM")}
	var buf bytes.Buffer
	buf.WriteString("From: " + from.String() + "\r\n")
	buf.WriteString("To: " + os.Getenv("MAIL_TO") + "\r\n")
	buf.WriteString("Subject: TODO Reminder\r\n")
	buf.WriteString("\r\n")
	buf.WriteString("This is your todo list\n\n")
	for _, todo := range todos {
		fmt.Fprintf(&buf, "%s %s\n", todo.Until, todo.Content)
	}

	smtpAuth := smtp.PlainAuth(
		os.Getenv("MAIL_DOMAIN"),
		os.Getenv("MAIL_USER"),
		os.Getenv("MAIL_PASSWORD"),
		os.Getenv("MAIL_AUTHSERVER"),
	)
	var messages []string
	err = smtp.SendMail(
		os.Getenv("MAIL_SERVER"),
		smtpAuth,
		from.Address,
		[]string{os.Getenv("MAIL_TO")},
		buf.Bytes())
	if err != nil {
		e.Logger.Error(err)
		messages = append(messages, "Failed to notify todos by mail")
	} else {
		e.Logger.Info("notified todos by mail")
		messages = append(messages, "Notified todos by mail!")
	}

	// Fetch the todos list
	var all []Todo
	err = db.NewSelect().Model(&all).Order("created_at").Scan(ctx)
	if err != nil {
		e.Logger.Error(err)
		return c.Render(http.StatusInternalServerError, "index", Data{Errors: []error{err}})
	}

	// Pass both the messages and the todos list to the template
	return c.Render(http.StatusOK, "index",
		Data{
			Todos:    all,
			Messages: messages,
		})

}
