package app

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Database interface {
	Connect(ctx context.Context, url string) error
	Close(ctx context.Context) error

	CreateSchemas(ctx context.Context, rpcs []RPC) error
}

type database struct {
	db *sqlx.DB
}

var (
	errDatabaseClosed = errors.New("database is closed")
)

func NewDatabase() Database {
	return &database{}
}

func (d *database) Connect(ctx context.Context, url string) error {
	db, err := sqlx.Open("postgres", url)
	if err != nil {
		return err
	}
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	d.db = db
	return nil
}

func (d *database) Close(ctx context.Context) error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return err
		}
		d.db = nil
	}
	return nil
}

func (d database) CreateSchemas(ctx context.Context, rpcs []RPC) error {
	if d.db == nil {
		return errDatabaseClosed
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, rpc := range rpcs {
		_, err := tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+rpc.Name())
		if err != nil {
			defer tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
