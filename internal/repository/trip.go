package repository

import (
	"context"

	"ride/internal/domain"
)

// TripRepository defines the persistence operations for trips.
type TripRepository interface {
	// Create persists a new trip.
	Create(ctx context.Context, trip *domain.Trip) error

	// GetByID retrieves a trip by ID.
	GetByID(ctx context.Context, id string) (*domain.Trip, error)

	// GetAll retrieves all trips.
	GetAll(ctx context.Context) ([]*domain.Trip, error)

	// Update updates an existing trip.
	Update(ctx context.Context, trip *domain.Trip) error

	// GetActiveByDriverID retrieves the active trip for a driver.
	// Returns nil if no active trip exists.
	GetActiveByDriverID(ctx context.Context, driverID string) (*domain.Trip, error)
}
