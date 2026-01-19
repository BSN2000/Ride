package postgres

import (
	"context"
	"database/sql"
	"errors"

	"ride/internal/domain"
	"ride/internal/repository"
)

// DriverRepository is a PostgreSQL implementation of repository.DriverRepository.
type DriverRepository struct {
	q Querier
}

// NewDriverRepository creates a new PostgreSQL driver repository.
func NewDriverRepository(db *sql.DB) *DriverRepository {
	return &DriverRepository{q: db}
}

// NewDriverRepositoryWithTx creates a driver repository using a transaction.
func NewDriverRepositoryWithTx(tx *sql.Tx) *DriverRepository {
	return &DriverRepository{q: tx}
}

// Create adds a new driver.
func (r *DriverRepository) Create(ctx context.Context, driver *domain.Driver) error {
	query := `INSERT INTO drivers (id, name, phone, status, tier) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.q.ExecContext(ctx, query, driver.ID, driver.Name, driver.Phone, driver.Status, driver.Tier)
	return err
}

// GetByID retrieves a driver by ID.
func (r *DriverRepository) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	query := `SELECT id, COALESCE(name, ''), COALESCE(phone, ''), status, tier FROM drivers WHERE id = $1`

	var driver domain.Driver
	err := r.q.QueryRowContext(ctx, query, id).Scan(
		&driver.ID,
		&driver.Name,
		&driver.Phone,
		&driver.Status,
		&driver.Tier,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &driver, nil
}

// GetByPhone retrieves a driver by phone number.
func (r *DriverRepository) GetByPhone(ctx context.Context, phone string) (*domain.Driver, error) {
	query := `SELECT id, name, phone, status, tier FROM drivers WHERE phone = $1`

	var driver domain.Driver
	err := r.q.QueryRowContext(ctx, query, phone).Scan(
		&driver.ID,
		&driver.Name,
		&driver.Phone,
		&driver.Status,
		&driver.Tier,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &driver, nil
}

// GetAll retrieves all drivers.
func (r *DriverRepository) GetAll(ctx context.Context) ([]*domain.Driver, error) {
	query := `SELECT id, COALESCE(name, ''), COALESCE(phone, ''), status, tier FROM drivers ORDER BY id`
	rows, err := r.q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []*domain.Driver
	for rows.Next() {
		var driver domain.Driver
		if err := rows.Scan(&driver.ID, &driver.Name, &driver.Phone, &driver.Status, &driver.Tier); err != nil {
			return nil, err
		}
		drivers = append(drivers, &driver)
	}
	return drivers, rows.Err()
}

// UpdateStatus updates the status of a driver.
func (r *DriverRepository) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
	query := `UPDATE drivers SET status = $1 WHERE id = $2`

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
