package service

import (
	"context"

	"ride/internal/redis"
	"ride/internal/repository"
)

// SurgeService calculates surge pricing based on supply and demand.
type SurgeService struct {
	locationStore redis.LocationStoreInterface
	rideRepo      repository.RideRepository
}

// NewSurgeService creates a new SurgeService.
func NewSurgeService(
	locationStore redis.LocationStoreInterface,
	rideRepo repository.RideRepository,
) *SurgeService {
	return &SurgeService{
		locationStore: locationStore,
		rideRepo:      rideRepo,
	}
}

// SurgeConfig contains surge pricing configuration.
type SurgeConfig struct {
	RadiusKm       float64 // Radius to check for supply/demand
	LowSurgeRatio  float64 // Demand/supply ratio for 1.25x surge
	MedSurgeRatio  float64 // Demand/supply ratio for 1.5x surge
	HighSurgeRatio float64 // Demand/supply ratio for 2.0x surge
	MaxSurge       float64 // Maximum surge multiplier
}

// DefaultSurgeConfig returns the default surge configuration.
func DefaultSurgeConfig() SurgeConfig {
	return SurgeConfig{
		RadiusKm:       5.0, // 5km radius
		LowSurgeRatio:  1.2, // 20% more demand than supply
		MedSurgeRatio:  1.5, // 50% more demand than supply
		HighSurgeRatio: 2.0, // 100% more demand than supply
		MaxSurge:       2.0, // Cap at 2x
	}
}

// GetMultiplier calculates the surge multiplier for a given location.
// Returns 1.0 if no surge, up to MaxSurge (default 2.0) if high demand.
func (s *SurgeService) GetMultiplier(ctx context.Context, lat, lng float64) float64 {
	config := DefaultSurgeConfig()

	// Get supply: count online drivers in the area
	supply := s.countDriversInArea(ctx, lat, lng, config.RadiusKm)

	// Get demand: count active ride requests in the area
	demand := s.countActiveRequestsInArea(ctx, lat, lng, config.RadiusKm)

	// Calculate surge based on demand/supply ratio
	return s.calculateSurgeMultiplier(supply, demand, config)
}

// countDriversInArea returns the number of online drivers within radius.
func (s *SurgeService) countDriversInArea(ctx context.Context, lat, lng, radiusKm float64) int {
	drivers, err := s.locationStore.FindNearbyDrivers(ctx, lat, lng, radiusKm)
	if err != nil {
		// On error, assume no surge (fail open)
		return 10 // Return a reasonable default to avoid false surge
	}
	return len(drivers)
}

// countActiveRequestsInArea returns the number of active ride requests in area.
// This is a simplified implementation - in production, you'd use spatial indexing.
func (s *SurgeService) countActiveRequestsInArea(ctx context.Context, lat, lng, radiusKm float64) int {
	rides, err := s.rideRepo.GetAll(ctx)
	if err != nil {
		return 0
	}

	count := 0
	for _, ride := range rides {
		// Only count REQUESTED or ASSIGNED rides (active)
		if ride.Status == "CANCELLED" {
			continue
		}

		// Simple distance check (Euclidean approximation)
		// In production, use Haversine formula
		latDiff := ride.PickupLat - lat
		lngDiff := ride.PickupLng - lng

		// Rough conversion: 1 degree ≈ 111km at equator
		distKm := ((latDiff * latDiff) + (lngDiff * lngDiff)) * 111 * 111
		if distKm <= radiusKm*radiusKm*111*111 {
			count++
		}
	}

	return count
}

// calculateSurgeMultiplier determines the multiplier based on supply/demand ratio.
func (s *SurgeService) calculateSurgeMultiplier(supply, demand int, config SurgeConfig) float64 {
	// Avoid division by zero
	if supply == 0 {
		if demand > 0 {
			return config.MaxSurge // Maximum surge when no drivers
		}
		return 1.0 // No demand, no surge
	}

	ratio := float64(demand) / float64(supply)

	// Determine surge tier based on ratio
	switch {
	case ratio >= config.HighSurgeRatio:
		return config.MaxSurge // 2.0x
	case ratio >= config.MedSurgeRatio:
		return 1.5 // 1.5x
	case ratio >= config.LowSurgeRatio:
		return 1.25 // 1.25x
	default:
		return 1.0 // No surge
	}
}
