package database

import (
	"context"
	"database/sql"
)

// Handler 满足 sql.DB 和 sql.Conn 和 sql.Tx
type Handler interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
}
