package redis

import (
	"context"
	"time"
)

// LocationStoreInterface defines the interface for driver location operations.
type LocationStoreInterface interface {
	UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error
	FindNearbyDrivers(ctx context.Context, lat, lng, radiusKm float64) ([]DriverLocation, error)
	RemoveLocation(ctx context.Context, driverID string) error
}

// LockStoreInterface defines the interface for distributed locking.
type LockStoreInterface interface {
	AcquireDriverLock(ctx context.Context, driverID string, ttl time.Duration) (bool, error)
	ReleaseDriverLock(ctx context.Context, driverID string) error
}

// Ensure concrete types implement interfaces.
var (
	_ LocationStoreInterface = (*LocationStore)(nil)
	_ LockStoreInterface     = (*LockStore)(nil)
)
