package pgslap

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type PgConfig struct {
	*pgx.ConnConfig
	OnlyPrint bool
}

type DB interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Close(ctx context.Context) error
}

func (pgCfg *PgConfig) openAndPing() (DB, error) {
	if pgCfg.OnlyPrint {
		return &NullDB{}, nil
	}

	conn, err := pgx.ConnectConfig(context.Background(), pgCfg.ConnConfig)

	if err != nil {
		return nil, err
	}

	err = conn.Ping(context.Background())

	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (pgCfg *PgConfig) Copy() *PgConfig {
	return &PgConfig{
		ConnConfig: pgCfg.ConnConfig.Copy(),
		OnlyPrint:  pgCfg.OnlyPrint,
	}
}
