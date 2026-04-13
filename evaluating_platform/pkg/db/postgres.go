package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool 创建 PostgreSQL 连接池
func NewPostgresPool(ctx context.Context, dsn string, maxConns, minConns int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(minConns)

	// 每个新连接建立后强制设置 client_encoding = UTF8
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET client_encoding = 'UTF8'")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// MigrateUp 执行数据库迁移
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, migrationFile string) error {
	sql, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("read migration file %s: %w", migrationFile, err)
	}

	_, err = pool.Exec(ctx, string(sql))
	if err != nil {
		return fmt.Errorf("execute migration %s: %w", migrationFile, err)
	}

	return nil
}
