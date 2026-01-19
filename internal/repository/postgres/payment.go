package postgres

import (
	"context"
	"database/sql"
	"errors"

	"ride/internal/domain"
	"ride/internal/repository"
)

// PaymentRepository is a PostgreSQL implementation of repository.PaymentRepository.
type PaymentRepository struct {
	q Querier
}

// NewPaymentRepository creates a new PostgreSQL payment repository.
func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{q: db}
}

// NewPaymentRepositoryWithTx creates a payment repository using a transaction.
func NewPaymentRepositoryWithTx(tx *sql.Tx) *PaymentRepository {
	return &PaymentRepository{q: tx}
}

// Create persists a new payment.
func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, trip_id, amount, status, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.q.ExecContext(ctx, query,
		payment.ID,
		payment.TripID,
		payment.Amount,
		payment.Status,
		payment.IdempotencyKey,
	)

	return err
}

// GetByID retrieves a payment by ID.
func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, trip_id, amount, status, idempotency_key
		FROM payments WHERE id = $1
	`

	var payment domain.Payment
	err := r.q.QueryRowContext(ctx, query, id).Scan(
		&payment.ID,
		&payment.TripID,
		&payment.Amount,
		&payment.Status,
		&payment.IdempotencyKey,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &payment, nil
}

// GetByIdempotencyKey retrieves a payment by its idempotency key.
// Returns nil if no payment exists with the given key.
func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	query := `
		SELECT id, trip_id, amount, status, idempotency_key
		FROM payments WHERE idempotency_key = $1
	`

	var payment domain.Payment
	err := r.q.QueryRowContext(ctx, query, key).Scan(
		&payment.ID,
		&payment.TripID,
		&payment.Amount,
		&payment.Status,
		&payment.IdempotencyKey,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &payment, nil
}

// UpdateStatus updates the status of a payment.
func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	query := `UPDATE payments SET status = $1 WHERE id = $2`

	result, err := r.q.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}
