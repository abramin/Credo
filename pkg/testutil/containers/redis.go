//go:build integration

package containers

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

// RedisContainer wraps a testcontainers Redis instance.
type RedisContainer struct {
	Container testcontainers.Container
	Addr      string
	Client    *redis.Client
}

// NewRedisContainer starts a new Redis container.
func NewRedisContainer(t *testing.T) *RedisContainer {
	t.Helper()

	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}

	addr, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to get redis connection string: %v", err)
	}

	// Parse the connection string (redis://host:port)
	opts, err := redis.ParseURL(addr)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("failed to parse redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		_ = container.Terminate(ctx)
		t.Fatalf("failed to ping redis: %v", err)
	}

	rc := &RedisContainer{
		Container: container,
		Addr:      addr,
		Client:    client,
	}

	// Note: We don't register t.Cleanup here because the container is managed
	// by the singleton Manager and shared across test suites. Ryuk handles cleanup.

	return rc
}

// FlushAll removes all keys from the Redis database.
// Use between tests to ensure isolation.
func (r *RedisContainer) FlushAll(ctx context.Context) error {
	return r.Client.FlushAll(ctx).Err()
}
