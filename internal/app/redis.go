package app

import (
	"context"
	"fmt"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/go-redis/v9"

	"ride/internal/config"
)

// NewRedisClient creates a new Redis client with optional New Relic instrumentation.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig, nrApp *newrelic.Application) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Add New Relic hook for Redis instrumentation if enabled
	if nrApp != nil {
		client.AddHook(&nrRedisHook{app: nrApp})
	}

	// Verify connection.
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

// nrRedisHook implements redis.Hook for New Relic instrumentation.
type nrRedisHook struct {
	app *newrelic.Application
}

func (h *nrRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *nrRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		txn := newrelic.FromContext(ctx)
		if txn != nil {
			segment := newrelic.DatastoreSegment{
				StartTime:  txn.StartSegmentNow(),
				Product:    newrelic.DatastoreRedis,
				Operation:  cmd.Name(),
				Collection: "redis",
			}
			defer segment.End()
		}
		return next(ctx, cmd)
	}
}

func (h *nrRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		txn := newrelic.FromContext(ctx)
		if txn != nil {
			segment := newrelic.DatastoreSegment{
				StartTime:  txn.StartSegmentNow(),
				Product:    newrelic.DatastoreRedis,
				Operation:  "pipeline",
				Collection: "redis",
			}
			defer segment.End()
		}
		return next(ctx, cmds)
	}
}
