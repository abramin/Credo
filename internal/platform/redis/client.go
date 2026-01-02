package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"credo/internal/platform/config"
)

// Client wraps the go-redis client with health checking capabilities.
type Client struct {
	*redis.Client
}

// New creates a new Redis client from the provided configuration.
// Returns nil if the URL is empty (Redis not configured).
func New(cfg config.RedisConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, nil
	}

	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	// Apply configuration overrides
	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout

	client := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Client{Client: client}, nil
}

// Health checks if the Redis connection is healthy.
func (c *Client) Health(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.Client.Close()
}
