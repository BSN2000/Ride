package domain

import "time"

// RideStatus represents the current status of a ride.
type RideStatus string

const (
	RideStatusRequested RideStatus = "REQUESTED"
	RideStatusAssigned  RideStatus = "ASSIGNED"
	RideStatusInTrip    RideStatus = "IN_TRIP"
	RideStatusCompleted RideStatus = "COMPLETED"
	RideStatusCancelled RideStatus = "CANCELLED"
)

// PaymentMethod represents the payment method for a ride.
type PaymentMethod string

const (
	PaymentMethodCash   PaymentMethod = "CASH"
	PaymentMethodCard   PaymentMethod = "CARD"
	PaymentMethodWallet PaymentMethod = "WALLET"
	PaymentMethodUPI    PaymentMethod = "UPI"
)

// Ride represents a ride request in the system.
type Ride struct {
	ID               string
	RiderID          string
	PickupLat        float64
	PickupLng        float64
	DestinationLat   float64
	DestinationLng   float64
	Status           RideStatus
	AssignedDriverID string
	SurgeMultiplier  float64       // 1.0 = no surge, 1.5 = 50% surge, 2.0 = 100% surge
	PaymentMethod    PaymentMethod // Payment method for this ride
	CreatedAt        time.Time
	CancelledAt      time.Time
	CancelReason     string
}
