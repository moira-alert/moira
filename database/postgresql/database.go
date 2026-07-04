package postgresql

import (
	"context"
	"database/sql"

	"github.com/moira-alert/moira"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const DriverName = "pgx"

type RDB interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type RWDB interface {
	RDB
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
}

type Database interface {
	Master() RWDB
	Replica() RDB
}

type DefaultDatabase struct {
	conn *sql.DB
}

func (d *DefaultDatabase) Master() RWDB {
	return d.conn
}

func (d *DefaultDatabase) Replica() RDB {
	return d.conn
}

type DbConnector struct {
	db Database
	ctx context.Context
}

func NewDatabase(
	ctx context.Context,
	logger moira.Logger,
	config DatabaseConfig,
) (*DbConnector, error) {
	p, err := sql.Open(DriverName, config.Master.ConnectionString)

	return &DbConnector{
		db: &DefaultDatabase{
			conn: p,
		},
		ctx: ctx,
	}, err
}
