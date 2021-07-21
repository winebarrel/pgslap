package pgslap

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type NullDB struct{}

func (db *NullDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	fmt.Fprintf(os.Stderr, "%s %v\n", sql, args)
	return nil, nil
}

func (db *NullDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	fmt.Fprintf(os.Stderr, "%s %v\n", sql, args)
	return nil, nil
}

func (db *NullDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	fmt.Fprintf(os.Stderr, "%s %v\n", sql, args)
	return nil
}

func (db *NullDB) Close(ctx context.Context) error {
	return nil
}
