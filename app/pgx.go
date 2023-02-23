package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func ConnectPostgres(ctx context.Context, url string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, url)
}

func CreateSchema(ctx context.Context, conn *pgx.Conn, name string) error {
	q := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", name)
	_, err := conn.Exec(ctx, q)
	return err
}

func CreateEventTable(ctx context.Context, conn *pgx.Conn, rpcName string, contract Contract) error {
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS "$1".$2 (id BIGSERIAL PRIMARY KEY);`, rpcName, contract.Name())
	return err
}
