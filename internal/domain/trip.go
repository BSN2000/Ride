package domain

import "time"

// TripStatus represents the current status of a trip.
type TripStatus string

const (
	TripStatusStarted TripStatus = "STARTED"
	TripStatusPaused  TripStatus = "PAUSED"
	TripStatusEnded   TripStatus = "ENDED"
)

// Trip represents an active or completed trip in the system.
type Trip struct {
	ID          string
	RideID      string
	DriverID    string
	Status      TripStatus
	Fare        float64
	StartedAt   time.Time
	EndedAt     time.Time
	PausedAt    time.Time     // When trip was paused
	TotalPaused time.Duration // Total time paused (for fare calculation)
}

// Receipt represents a trip receipt.
type Receipt struct {
	ID            string
	TripID        string
	RideID        string
	DriverID      string
	RiderID       string
	PickupLat     float64
	PickupLng     float64
	DestinationLat float64
	DestinationLng float64
	BaseFare      float64
	SurgeMultiplier float64
	SurgeAmount   float64
	TotalFare     float64
	PaymentMethod PaymentMethod
	PaymentStatus PaymentStatus
	Duration      time.Duration
	Distance      float64 // In kilometers (estimated)
	StartedAt     time.Time
	EndedAt       time.Time
	CreatedAt     time.Time
}
