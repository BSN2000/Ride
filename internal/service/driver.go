package service

import (
	"context"

	"ride/internal/domain"
	"ride/internal/redis"
	"ride/internal/repository"
)

// DriverService handles driver operations.
type DriverService struct {
	locationStore redis.LocationStoreInterface
	cacheStore    *redis.CacheStore
	driverRepo    repository.DriverRepository
}

// NewDriverService creates a new DriverService.
func NewDriverService(
	locationStore redis.LocationStoreInterface,
	cacheStore *redis.CacheStore,
	driverRepo repository.DriverRepository,
) *DriverService {
	return &DriverService{
		locationStore: locationStore,
		cacheStore:    cacheStore,
		driverRepo:    driverRepo,
	}
}

// UpdateLocationRequest contains the parameters for updating driver location.
type UpdateLocationRequest struct {
	DriverID string
	Lat      float64
	Lng      float64
}

// UpdateLocation updates a driver's location in Redis and sets them ONLINE.
// Optimized with cache invalidation and available driver tracking.
func (s *DriverService) UpdateLocation(ctx context.Context, req UpdateLocationRequest) error {
	if req.DriverID == "" {
		return ErrInvalidDriverID
	}

	if !isValidLatitude(req.Lat) || !isValidLongitude(req.Lng) {
		return ErrInvalidLocation
	}

	// Update location in Redis (primary real-time data store)
	if err := s.locationStore.UpdateLocation(ctx, req.DriverID, req.Lat, req.Lng); err != nil {
		return err
	}

	// Set driver status to ONLINE when they update location
	err := s.driverRepo.UpdateStatus(ctx, req.DriverID, domain.DriverStatusOnline)
	if err != nil && err != repository.ErrNotFound {
		return err
	}

	if s.cacheStore != nil {
		// Add to available drivers set for fast lookup
		_ = s.cacheStore.AddAvailableDriver(ctx, req.DriverID)

		// Update driver cache with new status
		driver, err := s.driverRepo.GetByID(ctx, req.DriverID)
		if err == nil {
			cached := &redis.CachedDriver{
				ID:     driver.ID,
				Name:   driver.Name,
				Phone:  driver.Phone,
				Status: string(driver.Status),
				Tier:   string(driver.Tier),
			}
			_ = s.cacheStore.SetDriver(ctx, cached)
		}
	}

	return nil
}

// SetDriverOffline sets a driver as offline and updates cache.
func (s *DriverService) SetDriverOffline(ctx context.Context, driverID string) error {
	if driverID == "" {
		return ErrInvalidDriverID
	}

	// Update DB
	if err := s.driverRepo.UpdateStatus(ctx, driverID, domain.DriverStatusOffline); err != nil {
		return err
	}

	// Remove from Redis GEO index
	if err := s.locationStore.RemoveLocation(ctx, driverID); err != nil {
		return err
	}

	if s.cacheStore != nil {
		_ = s.cacheStore.InvalidateDriver(ctx, driverID)
		_ = s.cacheStore.RemoveAvailableDriver(ctx, driverID)
	}

	return nil
}
