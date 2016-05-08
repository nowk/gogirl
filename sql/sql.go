package sql

import (
	"database/sql"
)

type closer interface {
	Close() error
}

// DB is a simplified interface that maps database/sql package and is required
// to create factories within gogirl
type DB interface {
	closer

	Prepare(string) (*sql.Stmt, error)
	Exec(string, ...interface{}) (sql.Result, error)
}
