package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LockStore handles distributed locking in Redis.
type LockStore struct {
	client *redis.Client
}

// NewLockStore creates a new LockStore.
func NewLockStore(client *redis.Client) *LockStore {
	return &LockStore{client: client}
}

// AcquireDriverLock attempts to acquire a lock for the given driver.
// Returns true if the lock was acquired, false if already held.
func (s *LockStore) AcquireDriverLock(ctx context.Context, driverID string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:driver:%s", driverID)

	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}

	return ok, nil
}

// ReleaseDriverLock releases the lock for the given driver.
func (s *LockStore) ReleaseDriverLock(ctx context.Context, driverID string) error {
	key := fmt.Sprintf("lock:driver:%s", driverID)

	return s.client.Del(ctx, key).Err()
}
