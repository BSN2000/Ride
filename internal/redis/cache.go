package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheStore handles entity caching in Redis.
type CacheStore struct {
	client *redis.Client
}

// NewCacheStore creates a new CacheStore.
func NewCacheStore(client *redis.Client) *CacheStore {
	return &CacheStore{client: client}
}

// Cache TTL constants
const (
	DriverCacheTTL = 30 * time.Second  // Driver status can change frequently
	RideCacheTTL   = 10 * time.Second  // Ride status changes during assignment
	TripCacheTTL   = 60 * time.Second  // Trip changes less frequently
)

// Key prefixes
const (
	driverCachePrefix = "cache:driver:"
	rideCachePrefix   = "cache:ride:"
	tripCachePrefix   = "cache:trip:"
)

// CachedDriver represents a cached driver entity.
type CachedDriver struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
	Tier   string `json:"tier"`
}

// CachedRide represents a cached ride entity.
type CachedRide struct {
	ID               string  `json:"id"`
	RiderID          string  `json:"rider_id"`
	Status           string  `json:"status"`
	AssignedDriverID string  `json:"assigned_driver_id"`
	SurgeMultiplier  float64 `json:"surge_multiplier"`
}

// GetDriver retrieves a driver from cache.
func (s *CacheStore) GetDriver(ctx context.Context, driverID string) (*CachedDriver, error) {
	key := driverCachePrefix + driverID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var driver CachedDriver
	if err := json.Unmarshal(data, &driver); err != nil {
		return nil, err
	}
	return &driver, nil
}

// SetDriver stores a driver in cache.
func (s *CacheStore) SetDriver(ctx context.Context, driver *CachedDriver) error {
	key := driverCachePrefix + driver.ID
	data, err := json.Marshal(driver)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, DriverCacheTTL).Err()
}

// InvalidateDriver removes a driver from cache.
func (s *CacheStore) InvalidateDriver(ctx context.Context, driverID string) error {
	key := driverCachePrefix + driverID
	return s.client.Del(ctx, key).Err()
}

// GetRide retrieves a ride from cache.
func (s *CacheStore) GetRide(ctx context.Context, rideID string) (*CachedRide, error) {
	key := rideCachePrefix + rideID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var ride CachedRide
	if err := json.Unmarshal(data, &ride); err != nil {
		return nil, err
	}
	return &ride, nil
}

// SetRide stores a ride in cache.
func (s *CacheStore) SetRide(ctx context.Context, ride *CachedRide) error {
	key := rideCachePrefix + ride.ID
	data, err := json.Marshal(ride)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, RideCacheTTL).Err()
}

// InvalidateRide removes a ride from cache.
func (s *CacheStore) InvalidateRide(ctx context.Context, rideID string) error {
	key := rideCachePrefix + rideID
	return s.client.Del(ctx, key).Err()
}

// GetDriversBatch retrieves multiple drivers from cache using pipeline.
// Returns a map of driverID -> CachedDriver, and a slice of missing IDs.
func (s *CacheStore) GetDriversBatch(ctx context.Context, driverIDs []string) (map[string]*CachedDriver, []string, error) {
	if len(driverIDs) == 0 {
		return make(map[string]*CachedDriver), nil, nil
	}

	// Use pipeline for batch get
	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd, len(driverIDs))

	for _, id := range driverIDs {
		key := driverCachePrefix + id
		cmds[id] = pipe.Get(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		// Check if it's a partial error (some keys missing)
		// Pipeline returns nil error even if some keys are missing
	}

	result := make(map[string]*CachedDriver)
	var missing []string

	for id, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			if err == redis.Nil {
				missing = append(missing, id)
				continue
			}
			// Log error but continue
			missing = append(missing, id)
			continue
		}

		var driver CachedDriver
		if err := json.Unmarshal(data, &driver); err != nil {
			missing = append(missing, id)
			continue
		}
		result[id] = &driver
	}

	return result, missing, nil
}

// SetDriversBatch stores multiple drivers in cache using pipeline.
func (s *CacheStore) SetDriversBatch(ctx context.Context, drivers []*CachedDriver) error {
	if len(drivers) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()

	for _, driver := range drivers {
		key := driverCachePrefix + driver.ID
		data, err := json.Marshal(driver)
		if err != nil {
			continue // Skip invalid entries
		}
		pipe.Set(ctx, key, data, DriverCacheTTL)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// AcquireRideLock attempts to acquire a lock for ride assignment.
// This prevents multiple matching attempts on the same ride.
func (s *CacheStore) AcquireRideLock(ctx context.Context, rideID string, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("lock:ride:%s", rideID)
	ok, err := s.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// ReleaseRideLock releases the lock for a ride.
func (s *CacheStore) ReleaseRideLock(ctx context.Context, rideID string) error {
	key := fmt.Sprintf("lock:ride:%s", rideID)
	return s.client.Del(ctx, key).Err()
}

// TrackDriverStatus stores driver availability status for fast lookup.
// This is separate from the main cache - it's a set of available driver IDs.
func (s *CacheStore) AddAvailableDriver(ctx context.Context, driverID string) error {
	return s.client.SAdd(ctx, "available_drivers", driverID).Err()
}

// RemoveAvailableDriver removes a driver from the available set.
func (s *CacheStore) RemoveAvailableDriver(ctx context.Context, driverID string) error {
	return s.client.SRem(ctx, "available_drivers", driverID).Err()
}

// IsDriverAvailable checks if a driver is in the available set.
func (s *CacheStore) IsDriverAvailable(ctx context.Context, driverID string) (bool, error) {
	return s.client.SIsMember(ctx, "available_drivers", driverID).Result()
}

// GetAvailableDrivers returns all available driver IDs.
func (s *CacheStore) GetAvailableDrivers(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, "available_drivers").Result()
}
