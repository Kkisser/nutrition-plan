// Package db инкапсулирует подключение к PostgreSQL.
package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DSNFromEnv читает DATABASE_DSN из окружения.
func DSNFromEnv() (string, error) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		return "", fmt.Errorf("DATABASE_DSN not set")
	}
	return dsn, nil
}

// Connect открывает пул соединений.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
