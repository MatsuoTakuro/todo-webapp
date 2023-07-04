package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"

	_ "github.com/lib/pq"
)

var db *bun.DB

func setupDB() (*bun.DB, func(), error) {
	sqldb, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, nil, err
	}
	db = bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	ctx := context.Background()
	_, err = db.NewCreateTable().Model((*Todo)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return nil, func() {
			defer sqldb.Close()
			defer db.Close()
		}, err
	}

	return db, func() {
		defer sqldb.Close()
		defer db.Close()
	}, nil
}
