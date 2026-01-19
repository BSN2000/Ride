package domain

import "time"

// User represents a rider in the system.
type User struct {
	ID        string
	Name      string
	Phone     string
	CreatedAt time.Time
}
