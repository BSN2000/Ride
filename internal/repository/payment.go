package repository

import (
	"context"

	"ride/internal/domain"
)

// PaymentRepository defines the persistence operations for payments.
type PaymentRepository interface {
	// Create persists a new payment.
	Create(ctx context.Context, payment *domain.Payment) error

	// GetByID retrieves a payment by ID.
	GetByID(ctx context.Context, id string) (*domain.Payment, error)

	// GetByIdempotencyKey retrieves a payment by its idempotency key.
	// Returns nil if no payment exists with the given key.
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error)

	// UpdateStatus updates the status of a payment.
	UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error
}
