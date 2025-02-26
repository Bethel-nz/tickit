package pgxadapter

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PGXAdapter wraps a *pgx.Conn and implements the DBTX interface.
type PGXAdapter struct {
	conn *pgx.Conn
}

func NewPGXAdapter(conn *pgx.Conn) *PGXAdapter {
	return &PGXAdapter{conn: conn}
}

// Exec implements the DBTX interface.
func (p *PGXAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return p.conn.Exec(ctx, sql, args...)
}

// Query implements the DBTX interface.
func (p *PGXAdapter) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return p.conn.Query(ctx, sql, args...)
}

// QueryRow implements the DBTX interface.
func (p *PGXAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return p.conn.QueryRow(ctx, sql, args...)
}
