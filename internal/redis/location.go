package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const driverLocationKey = "drivers:locations"

// DriverLocation represents a driver's position.
type DriverLocation struct {
	DriverID string
	Lat      float64
	Lng      float64
}

// LocationStore handles driver location operations in Redis.
type LocationStore struct {
	client *redis.Client
}

// NewLocationStore creates a new LocationStore.
func NewLocationStore(client *redis.Client) *LocationStore {
	return &LocationStore{client: client}
}

// UpdateLocation stores a driver's location using GEOADD.
func (s *LocationStore) UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error {
	return s.client.GeoAdd(ctx, driverLocationKey, &redis.GeoLocation{
		Name:      driverID,
		Longitude: lng,
		Latitude:  lat,
	}).Err()
}

// FindNearbyDrivers returns driver IDs within the given radius (in kilometers).
func (s *LocationStore) FindNearbyDrivers(ctx context.Context, lat, lng, radiusKm float64) ([]DriverLocation, error) {
	results, err := s.client.GeoRadius(ctx, driverLocationKey, lng, lat, &redis.GeoRadiusQuery{
		Radius:    radiusKm,
		Unit:      "km",
		WithCoord: true,
		Sort:      "ASC",
	}).Result()
	if err != nil {
		return nil, err
	}

	locations := make([]DriverLocation, 0, len(results))
	for _, r := range results {
		locations = append(locations, DriverLocation{
			DriverID: r.Name,
			Lat:      r.Latitude,
			Lng:      r.Longitude,
		})
	}

	return locations, nil
}

// RemoveLocation removes a driver's location from the geo index.
func (s *LocationStore) RemoveLocation(ctx context.Context, driverID string) error {
	return s.client.ZRem(ctx, driverLocationKey, driverID).Err()
}
