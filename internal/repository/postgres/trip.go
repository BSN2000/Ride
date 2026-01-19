package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ride/internal/domain"
	"ride/internal/repository"
)

// TripRepository is a PostgreSQL implementation of repository.TripRepository.
type TripRepository struct {
	q Querier
}

// NewTripRepository creates a new PostgreSQL trip repository.
func NewTripRepository(db *sql.DB) *TripRepository {
	return &TripRepository{q: db}
}

// NewTripRepositoryWithTx creates a trip repository using a transaction.
func NewTripRepositoryWithTx(tx *sql.Tx) *TripRepository {
	return &TripRepository{q: tx}
}

// Create persists a new trip.
func (r *TripRepository) Create(ctx context.Context, trip *domain.Trip) error {
	query := `
		INSERT INTO trips (id, ride_id, driver_id, status, fare, started_at, ended_at, paused_at, total_paused_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	var endedAt sql.NullTime
	if !trip.EndedAt.IsZero() {
		endedAt = sql.NullTime{Time: trip.EndedAt, Valid: true}
	}

	var pausedAt sql.NullTime
	if !trip.PausedAt.IsZero() {
		pausedAt = sql.NullTime{Time: trip.PausedAt, Valid: true}
	}

	totalPausedSeconds := int64(trip.TotalPaused.Seconds())

	_, err := r.q.ExecContext(ctx, query,
		trip.ID,
		trip.RideID,
		trip.DriverID,
		trip.Status,
		trip.Fare,
		trip.StartedAt,
		endedAt,
		pausedAt,
		totalPausedSeconds,
	)

	return err
}

// GetByID retrieves a trip by ID.
func (r *TripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	query := `
		SELECT id, ride_id, driver_id, status, fare, started_at, ended_at, paused_at, total_paused_seconds
		FROM trips WHERE id = $1
	`

	var trip domain.Trip
	var endedAt sql.NullTime
	var pausedAt sql.NullTime
	var totalPausedSeconds int64

	err := r.q.QueryRowContext(ctx, query, id).Scan(
		&trip.ID,
		&trip.RideID,
		&trip.DriverID,
		&trip.Status,
		&trip.Fare,
		&trip.StartedAt,
		&endedAt,
		&pausedAt,
		&totalPausedSeconds,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	if endedAt.Valid {
		trip.EndedAt = endedAt.Time
	}
	if pausedAt.Valid {
		trip.PausedAt = pausedAt.Time
	}
	trip.TotalPaused = time.Duration(totalPausedSeconds) * time.Second

	return &trip, nil
}

// GetAll retrieves all trips.
func (r *TripRepository) GetAll(ctx context.Context) ([]*domain.Trip, error) {
	query := `
		SELECT id, ride_id, driver_id, status, fare, started_at, ended_at, paused_at, total_paused_seconds
		FROM trips ORDER BY started_at DESC LIMIT 100
	`

	rows, err := r.q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []*domain.Trip
	for rows.Next() {
		var trip domain.Trip
		var endedAt sql.NullTime
		var pausedAt sql.NullTime
		var totalPausedSeconds int64

		if err := rows.Scan(
			&trip.ID,
			&trip.RideID,
			&trip.DriverID,
			&trip.Status,
			&trip.Fare,
			&trip.StartedAt,
			&endedAt,
			&pausedAt,
			&totalPausedSeconds,
		); err != nil {
			return nil, err
		}

		if endedAt.Valid {
			trip.EndedAt = endedAt.Time
		}
		if pausedAt.Valid {
			trip.PausedAt = pausedAt.Time
		}
		trip.TotalPaused = time.Duration(totalPausedSeconds) * time.Second

		trips = append(trips, &trip)
	}

	return trips, rows.Err()
}

// Update updates an existing trip.
func (r *TripRepository) Update(ctx context.Context, trip *domain.Trip) error {
	query := `
		UPDATE trips
		SET ride_id = $1, driver_id = $2, status = $3, fare = $4, started_at = $5, ended_at = $6, paused_at = $7, total_paused_seconds = $8
		WHERE id = $9
	`

	var endedAt sql.NullTime
	if !trip.EndedAt.IsZero() {
		endedAt = sql.NullTime{Time: trip.EndedAt, Valid: true}
	}

	var pausedAt sql.NullTime
	if !trip.PausedAt.IsZero() {
		pausedAt = sql.NullTime{Time: trip.PausedAt, Valid: true}
	}

	totalPausedSeconds := int64(trip.TotalPaused.Seconds())

	result, err := r.q.ExecContext(ctx, query,
		trip.RideID,
		trip.DriverID,
		trip.Status,
		trip.Fare,
		trip.StartedAt,
		endedAt,
		pausedAt,
		totalPausedSeconds,
		trip.ID,
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

// GetActiveByDriverID retrieves the active trip for a driver.
// Returns nil if no active trip exists.
func (r *TripRepository) GetActiveByDriverID(ctx context.Context, driverID string) (*domain.Trip, error) {
	query := `
		SELECT id, ride_id, driver_id, status, fare, started_at, ended_at, paused_at, total_paused_seconds
		FROM trips
		WHERE driver_id = $1 AND status != $2
		LIMIT 1
	`

	var trip domain.Trip
	var endedAt sql.NullTime
	var pausedAt sql.NullTime
	var totalPausedSeconds int64

	err := r.q.QueryRowContext(ctx, query, driverID, domain.TripStatusEnded).Scan(
		&trip.ID,
		&trip.RideID,
		&trip.DriverID,
		&trip.Status,
		&trip.Fare,
		&trip.StartedAt,
		&endedAt,
		&pausedAt,
		&totalPausedSeconds,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if endedAt.Valid {
		trip.EndedAt = endedAt.Time
	}
	if pausedAt.Valid {
		trip.PausedAt = pausedAt.Time
	}
	trip.TotalPaused = time.Duration(totalPausedSeconds) * time.Second

	return &trip, nil
}

// Ensure TripRepository implements repository.TripRepository.
var _ repository.TripRepository = (*TripRepository)(nil)
