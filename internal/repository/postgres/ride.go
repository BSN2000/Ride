package postgres

import (
	"context"
	"database/sql"
	"errors"

	"ride/internal/domain"
	"ride/internal/repository"
)

// RideRepository is a PostgreSQL implementation of repository.RideRepository.
type RideRepository struct {
	q Querier
}

// NewRideRepository creates a new PostgreSQL ride repository.
func NewRideRepository(db *sql.DB) *RideRepository {
	return &RideRepository{q: db}
}

// NewRideRepositoryWithTx creates a ride repository using a transaction.
func NewRideRepositoryWithTx(tx *sql.Tx) *RideRepository {
	return &RideRepository{q: tx}
}

// Create persists a new ride.
func (r *RideRepository) Create(ctx context.Context, ride *domain.Ride) error {
	query := `
		INSERT INTO rides (id, rider_id, pickup_lat, pickup_lng, destination_lat, destination_lng, status, assigned_driver_id, surge_multiplier, payment_method, cancelled_at, cancel_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	var assignedDriverID sql.NullString
	if ride.AssignedDriverID != "" {
		assignedDriverID = sql.NullString{String: ride.AssignedDriverID, Valid: true}
	}

	// Default surge to 1.0 if not set
	surgeMultiplier := ride.SurgeMultiplier
	if surgeMultiplier < 1.0 {
		surgeMultiplier = 1.0
	}

	// Default payment method to CASH if not set
	paymentMethod := ride.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "CASH"
	}

	var cancelledAt sql.NullTime
	if !ride.CancelledAt.IsZero() {
		cancelledAt = sql.NullTime{Time: ride.CancelledAt, Valid: true}
	}

	var cancelReason sql.NullString
	if ride.CancelReason != "" {
		cancelReason = sql.NullString{String: ride.CancelReason, Valid: true}
	}

	_, err := r.q.ExecContext(ctx, query,
		ride.ID,
		ride.RiderID,
		ride.PickupLat,
		ride.PickupLng,
		ride.DestinationLat,
		ride.DestinationLng,
		ride.Status,
		assignedDriverID,
		surgeMultiplier,
		paymentMethod,
		cancelledAt,
		cancelReason,
		ride.CreatedAt,
	)

	return err
}

// GetByID retrieves a ride by ID.
func (r *RideRepository) GetByID(ctx context.Context, id string) (*domain.Ride, error) {
	query := `
		SELECT id, rider_id, pickup_lat, pickup_lng, destination_lat, destination_lng, status, assigned_driver_id, surge_multiplier, payment_method, cancelled_at, cancel_reason, created_at
		FROM rides WHERE id = $1
	`

	var ride domain.Ride
	var assignedDriverID sql.NullString
	var cancelledAt sql.NullTime
	var cancelReason sql.NullString

	err := r.q.QueryRowContext(ctx, query, id).Scan(
		&ride.ID,
		&ride.RiderID,
		&ride.PickupLat,
		&ride.PickupLng,
		&ride.DestinationLat,
		&ride.DestinationLng,
		&ride.Status,
		&assignedDriverID,
		&ride.SurgeMultiplier,
		&ride.PaymentMethod,
		&cancelledAt,
		&cancelReason,
		&ride.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	if assignedDriverID.Valid {
		ride.AssignedDriverID = assignedDriverID.String
	}
	if cancelledAt.Valid {
		ride.CancelledAt = cancelledAt.Time
	}
	if cancelReason.Valid {
		ride.CancelReason = cancelReason.String
	}

	return &ride, nil
}

// GetAll retrieves all rides.
func (r *RideRepository) GetAll(ctx context.Context) ([]*domain.Ride, error) {
	query := `
		SELECT id, rider_id, pickup_lat, pickup_lng, destination_lat, destination_lng, status, assigned_driver_id, surge_multiplier, payment_method, cancelled_at, cancel_reason, created_at
		FROM rides ORDER BY created_at DESC LIMIT 100
	`

	rows, err := r.q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rides []*domain.Ride
	for rows.Next() {
		var ride domain.Ride
		var assignedDriverID sql.NullString
		var cancelledAt sql.NullTime
		var cancelReason sql.NullString
		if err := rows.Scan(
			&ride.ID,
			&ride.RiderID,
			&ride.PickupLat,
			&ride.PickupLng,
			&ride.DestinationLat,
			&ride.DestinationLng,
			&ride.Status,
			&assignedDriverID,
			&ride.SurgeMultiplier,
			&ride.PaymentMethod,
			&cancelledAt,
			&cancelReason,
			&ride.CreatedAt,
		); err != nil {
			return nil, err
		}
		if assignedDriverID.Valid {
			ride.AssignedDriverID = assignedDriverID.String
		}
		if cancelledAt.Valid {
			ride.CancelledAt = cancelledAt.Time
		}
		if cancelReason.Valid {
			ride.CancelReason = cancelReason.String
		}
		rides = append(rides, &ride)
	}
	return rides, rows.Err()
}

// Update updates an existing ride.
func (r *RideRepository) Update(ctx context.Context, ride *domain.Ride) error {
	query := `
		UPDATE rides
		SET rider_id = $1, pickup_lat = $2, pickup_lng = $3, destination_lat = $4, destination_lng = $5, status = $6, assigned_driver_id = $7, surge_multiplier = $8, payment_method = $9, cancelled_at = $10, cancel_reason = $11
		WHERE id = $12
	`

	var assignedDriverID sql.NullString
	if ride.AssignedDriverID != "" {
		assignedDriverID = sql.NullString{String: ride.AssignedDriverID, Valid: true}
	}

	// Default surge to 1.0 if not set
	surgeMultiplier := ride.SurgeMultiplier
	if surgeMultiplier < 1.0 {
		surgeMultiplier = 1.0
	}

	// Default payment method to CASH if not set
	paymentMethod := ride.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "CASH"
	}

	var cancelledAt sql.NullTime
	if !ride.CancelledAt.IsZero() {
		cancelledAt = sql.NullTime{Time: ride.CancelledAt, Valid: true}
	}

	var cancelReason sql.NullString
	if ride.CancelReason != "" {
		cancelReason = sql.NullString{String: ride.CancelReason, Valid: true}
	}

	result, err := r.q.ExecContext(ctx, query,
		ride.RiderID,
		ride.PickupLat,
		ride.PickupLng,
		ride.DestinationLat,
		ride.DestinationLng,
		ride.Status,
		assignedDriverID,
		surgeMultiplier,
		paymentMethod,
		cancelledAt,
		cancelReason,
		ride.ID,
	)
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
