// Package pg — реализация repository.Repo поверх PostgreSQL/pgx.
package pg

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo — PostgreSQL-реализация.
type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// Ping выполняет SELECT 1 через пул. Используется в /health, чтобы
// readiness-проба не отдавала 200 при недоступной БД.
func (r *Repo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
