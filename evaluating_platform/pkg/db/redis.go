package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(addr, password string, dbIndex int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           dbIndex,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return rdb, nil
}
