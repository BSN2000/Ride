package repository

import (
	"context"

	"ride/internal/domain"
)

// RideRepository defines the persistence operations for rides.
type RideRepository interface {
	// Create persists a new ride.
	Create(ctx context.Context, ride *domain.Ride) error

	// GetByID retrieves a ride by ID.
	GetByID(ctx context.Context, id string) (*domain.Ride, error)

	// GetAll retrieves all rides.
	GetAll(ctx context.Context) ([]*domain.Ride, error)

	// Update updates an existing ride.
	Update(ctx context.Context, ride *domain.Ride) error
}
