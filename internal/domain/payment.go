package domain

// PaymentStatus represents the current status of a payment.
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// Payment represents a payment for a trip.
type Payment struct {
	ID             string
	TripID         string
	Amount         float64
	Status         PaymentStatus
	IdempotencyKey string
}
