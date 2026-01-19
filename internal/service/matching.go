package service

import (
	"context"
	"database/sql"
	"time"

	"ride/internal/domain"
	"ride/internal/redis"
	"ride/internal/repository"
	"ride/internal/repository/postgres"
)

const (
	defaultSearchRadiusKm = 5.0
	driverLockTTL         = 10 * time.Second
	rideLockTTL           = 30 * time.Second // Lock ride during matching
)

// MatchingService handles driver-rider matching.
type MatchingService struct {
	db            *sql.DB
	locationStore redis.LocationStoreInterface
	lockStore     redis.LockStoreInterface
	cacheStore    *redis.CacheStore
	driverRepo    repository.DriverRepository
	rideRepo      repository.RideRepository
}

// NewMatchingService creates a new MatchingService.
func NewMatchingService(
	db *sql.DB,
	locationStore redis.LocationStoreInterface,
	lockStore redis.LockStoreInterface,
	cacheStore *redis.CacheStore,
	driverRepo repository.DriverRepository,
	rideRepo repository.RideRepository,
) *MatchingService {
	return &MatchingService{
		db:            db,
		locationStore: locationStore,
		lockStore:     lockStore,
		cacheStore:    cacheStore,
		driverRepo:    driverRepo,
		rideRepo:      rideRepo,
	}
}

// MatchRequest contains the parameters for matching a ride.
type MatchRequest struct {
	RideID   string
	Lat      float64
	Lng      float64
	Tier     domain.DriverTier // Optional: empty means any tier
	RadiusKm float64           // Optional: 0 uses default
}

// MatchResult contains the result of a successful match.
type MatchResult struct {
	DriverID string
	Ride     *domain.Ride
}

// Match finds and assigns an available driver to a ride.
// Optimized with:
// - Ride locking to prevent double assignment
// - Batch driver lookup from cache
// - Cache invalidation on assignment
func (s *MatchingService) Match(ctx context.Context, req MatchRequest) (*MatchResult, error) {
	// Set default radius if not specified.
	radiusKm := req.RadiusKm
	if radiusKm <= 0 {
		radiusKm = defaultSearchRadiusKm
	}

	// OPTIMIZATION 1: Acquire ride lock to prevent concurrent matching
	if s.cacheStore != nil {
		locked, err := s.cacheStore.AcquireRideLock(ctx, req.RideID, rideLockTTL)
		if err != nil {
			return nil, err
		}
		if !locked {
			// Another matching process is handling this ride
			return nil, ErrRideNotInRequestedState
		}
		defer s.cacheStore.ReleaseRideLock(ctx, req.RideID)
	}

	// Get ride and verify it's in REQUESTED state.
	ride, err := s.rideRepo.GetByID(ctx, req.RideID)
	if err != nil {
		return nil, err
	}

	if ride.Status != domain.RideStatusRequested {
		return nil, ErrRideNotInRequestedState
	}

	// Find nearby drivers from Redis (sorted by distance).
	nearbyDrivers, err := s.locationStore.FindNearbyDrivers(ctx, req.Lat, req.Lng, radiusKm)
	if err != nil {
		return nil, err
	}

	if len(nearbyDrivers) == 0 {
		return nil, ErrNoDriverAvailable
	}

	// OPTIMIZATION 2: Batch fetch driver data from cache
	driverIDs := make([]string, len(nearbyDrivers))
	for i, loc := range nearbyDrivers {
		driverIDs[i] = loc.DriverID
	}

	// Try to get drivers from cache first
	cachedDrivers, missingIDs, _ := s.getDriversBatchOptimized(ctx, driverIDs)

	// Fetch missing drivers from DB in a single query (if supported)
	// For now, fall back to individual queries for missing drivers
	dbDrivers := make(map[string]*domain.Driver)
	for _, id := range missingIDs {
		driver, err := s.driverRepo.GetByID(ctx, id)
		if err != nil {
			if err == repository.ErrNotFound {
				continue
			}
			return nil, err
		}
		dbDrivers[id] = driver
		// Cache the driver for future requests
		s.cacheDriverAsync(ctx, driver)
	}

	// Try each driver in order of proximity.
	for _, loc := range nearbyDrivers {
		driverID := loc.DriverID

		// OPTIMIZATION 3: Check cache first, then DB
		var driver *domain.Driver
		if cached, ok := cachedDrivers[driverID]; ok {
			// Use cached data for quick filtering
			if cached.Status != string(domain.DriverStatusOnline) {
				continue
			}
			if req.Tier != "" && cached.Tier != string(req.Tier) {
				continue
			}
			// Cache hit - still need full driver for assignment
			driver = s.cachedToDriver(cached)
		} else if dbDriver, ok := dbDrivers[driverID]; ok {
			driver = dbDriver
		} else {
			// Driver not found in cache or DB
			continue
		}

		// Filter by status (double-check for DB drivers).
		if driver.Status != domain.DriverStatusOnline {
			continue
		}

		// Filter by tier if specified.
		if req.Tier != "" && driver.Tier != req.Tier {
			continue
		}

		// Try to acquire driver lock.
		locked, err := s.lockStore.AcquireDriverLock(ctx, driverID, driverLockTTL)
		if err != nil {
			return nil, err
		}

		if !locked {
			// Driver is being assigned to another ride.
			continue
		}

		// OPTIMIZATION 4: Re-verify driver status from DB before assignment
		// This handles the case where cached status is stale
		freshDriver, err := s.driverRepo.GetByID(ctx, driverID)
		if err != nil {
			_ = s.lockStore.ReleaseDriverLock(ctx, driverID)
			if err == repository.ErrNotFound {
				continue
			}
			return nil, err
		}

		if freshDriver.Status != domain.DriverStatusOnline {
			_ = s.lockStore.ReleaseDriverLock(ctx, driverID)
			// Invalidate stale cache
			s.invalidateDriverCache(ctx, driverID)
			continue
		}

		// Attempt atomic assignment.
		result, err := s.assignDriver(ctx, ride, freshDriver)
		if err != nil {
			// Release lock on failure.
			_ = s.lockStore.ReleaseDriverLock(ctx, driverID)
			return nil, err
		}

		// OPTIMIZATION 5: Invalidate caches after assignment
		s.invalidateDriverCache(ctx, driverID)
		s.invalidateRideCache(ctx, ride.ID)

		// Success - driver lock will expire via TTL.
		return result, nil
	}

	return nil, ErrNoDriverAvailable
}

// getDriversBatchOptimized fetches drivers from cache using batch operation.
func (s *MatchingService) getDriversBatchOptimized(ctx context.Context, driverIDs []string) (map[string]*redis.CachedDriver, []string, error) {
	if s.cacheStore == nil {
		return make(map[string]*redis.CachedDriver), driverIDs, nil
	}
	return s.cacheStore.GetDriversBatch(ctx, driverIDs)
}

// cacheDriverAsync caches a driver asynchronously (fire and forget).
func (s *MatchingService) cacheDriverAsync(ctx context.Context, driver *domain.Driver) {
	if s.cacheStore == nil {
		return
	}
	go func() {
		cached := &redis.CachedDriver{
			ID:     driver.ID,
			Name:   driver.Name,
			Phone:  driver.Phone,
			Status: string(driver.Status),
			Tier:   string(driver.Tier),
		}
		_ = s.cacheStore.SetDriver(context.Background(), cached)
	}()
}

// cachedToDriver converts a cached driver to domain driver.
func (s *MatchingService) cachedToDriver(cached *redis.CachedDriver) *domain.Driver {
	return &domain.Driver{
		ID:     cached.ID,
		Name:   cached.Name,
		Phone:  cached.Phone,
		Status: domain.DriverStatus(cached.Status),
		Tier:   domain.DriverTier(cached.Tier),
	}
}

// invalidateDriverCache invalidates a driver's cache entry.
func (s *MatchingService) invalidateDriverCache(ctx context.Context, driverID string) {
	if s.cacheStore == nil {
		return
	}
	_ = s.cacheStore.InvalidateDriver(ctx, driverID)
	// Also remove from available drivers set
	_ = s.cacheStore.RemoveAvailableDriver(ctx, driverID)
}

// invalidateRideCache invalidates a ride's cache entry.
func (s *MatchingService) invalidateRideCache(ctx context.Context, rideID string) {
	if s.cacheStore == nil {
		return
	}
	_ = s.cacheStore.InvalidateRide(ctx, rideID)
}

// assignDriver atomically assigns a driver to a ride using a transaction.
func (s *MatchingService) assignDriver(ctx context.Context, ride *domain.Ride, driver *domain.Driver) (*MatchResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create transaction-scoped repositories.
	txRideRepo := postgres.NewRideRepositoryWithTx(tx)
	txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)

	// Update ride status and assign driver.
	ride.Status = domain.RideStatusAssigned
	ride.AssignedDriverID = driver.ID

	if err = txRideRepo.Update(ctx, ride); err != nil {
		return nil, err
	}

	// Update driver status to ON_TRIP.
	if err = txDriverRepo.UpdateStatus(ctx, driver.ID, domain.DriverStatusOnTrip); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &MatchResult{
		DriverID: driver.ID,
		Ride:     ride,
	}, nil
}
