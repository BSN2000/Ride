package postgres

import (
	"context"
	"database/sql"

	"ride/internal/domain"
	"ride/internal/repository"
)

// UserRepository implements repository.UserRepository using PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create adds a new user.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (id, name, phone) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Phone)
	return err
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, name, phone, created_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var user domain.User
	err := row.Scan(&user.ID, &user.Name, &user.Phone, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone retrieves a user by phone number.
func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	query := `SELECT id, name, phone, created_at FROM users WHERE phone = $1`
	row := r.db.QueryRowContext(ctx, query, phone)

	var user domain.User
	err := row.Scan(&user.ID, &user.Name, &user.Phone, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAll retrieves all users.
func (r *UserRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	query := `SELECT id, name, phone, created_at FROM users ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Phone, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}
