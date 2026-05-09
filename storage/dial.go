package storage

import (
	"context"
	"database/sql"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/goflower-io/xsql"
	"github.com/netlify/gotrue/conf"
	"github.com/pkg/errors"
)

// Connection holds the database connection. db is the current active querier
// (either *sql.DB or *sql.Tx), and rawDB is always the root *sql.DB.
type Connection struct {
	db    xsql.ExecQuerier
	rawDB *sql.DB
}

// DB returns the underlying ExecQuerier (db or tx).
func (c *Connection) DB() xsql.ExecQuerier {
	return c.db
}

// Dial opens a MySQL database connection and returns a Connection.
func Dial(config *conf.GlobalConfiguration) (*Connection, error) {
	if config.DB.Driver == "" && config.DB.URL != "" {
		// derive driver from URL scheme - assume "mysql"
		config.DB.Driver = "mysql"
	}

	sqlDB, err := sql.Open(config.DB.Driver, config.DB.URL)
	if err != nil {
		return nil, errors.Wrap(err, "opening database connection")
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		sqlDB.Close()
		return nil, errors.Wrap(err, "checking database connection")
	}

	return &Connection{db: sqlDB, rawDB: sqlDB}, nil
}

// Transaction runs fn inside a database transaction. If db is already a *sql.Tx,
// fn is called directly without starting a new transaction.
func (c *Connection) Transaction(fn func(*Connection) error) error {
	if _, ok := c.db.(*sql.Tx); ok {
		// Already inside a transaction – just call fn.
		return fn(c)
	}

	tx, err := c.rawDB.BeginTx(context.Background(), nil)
	if err != nil {
		return errors.Wrap(err, "beginning transaction")
	}

	txConn := &Connection{db: tx, rawDB: c.rawDB}
	if err := fn(txConn); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Exec executes a raw SQL statement.
func (c *Connection) Exec(query string, args ...interface{}) error {
	_, err := c.db.ExecContext(context.Background(), query, args...)
	return err
}

// Close closes the underlying database connection.
func (c *Connection) Close() error {
	return c.rawDB.Close()
}
