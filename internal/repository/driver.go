package repository

import (
	"context"

	"ride/internal/domain"
)

// DriverRepository defines the persistence operations for drivers.
type DriverRepository interface {
	// Create adds a new driver.
	Create(ctx context.Context, driver *domain.Driver) error

	// GetByID retrieves a driver by ID.
	GetByID(ctx context.Context, id string) (*domain.Driver, error)

	// GetByPhone retrieves a driver by phone number.
	GetByPhone(ctx context.Context, phone string) (*domain.Driver, error)

	// GetAll retrieves all drivers.
	GetAll(ctx context.Context) ([]*domain.Driver, error)

	// UpdateStatus updates the status of a driver.
	UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error
}
