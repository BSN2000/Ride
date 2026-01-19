package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ride/internal/domain"
)

// ReceiptService handles receipt generation.
type ReceiptService struct {
	notificationService *NotificationService
}

// NewReceiptService creates a new ReceiptService.
func NewReceiptService(notificationService *NotificationService) *ReceiptService {
	return &ReceiptService{
		notificationService: notificationService,
	}
}

// GenerateReceiptRequest contains the parameters for generating a receipt.
type GenerateReceiptRequest struct {
	Trip    *domain.Trip
	Ride    *domain.Ride
	Payment *domain.Payment
}

// GenerateReceipt generates a receipt for a completed trip.
func (s *ReceiptService) GenerateReceipt(ctx context.Context, req GenerateReceiptRequest) (*domain.Receipt, error) {
	if req.Trip == nil || req.Ride == nil {
		return nil, ErrInvalidTripID
	}

	// Calculate fare components
	baseFare := s.calculateBaseFare(req.Trip)
	surgeMultiplier := req.Ride.SurgeMultiplier
	if surgeMultiplier < 1.0 {
		surgeMultiplier = 1.0
	}
	surgeAmount := baseFare * (surgeMultiplier - 1.0)
	totalFare := req.Trip.Fare

	// Calculate duration (excluding paused time)
	duration := req.Trip.EndedAt.Sub(req.Trip.StartedAt) - req.Trip.TotalPaused

	// Estimate distance (simplified: based on coordinates)
	distance := s.estimateDistance(
		req.Ride.PickupLat, req.Ride.PickupLng,
		req.Ride.DestinationLat, req.Ride.DestinationLng,
	)

	// Determine payment status
	paymentStatus := domain.PaymentStatusPending
	if req.Payment != nil {
		paymentStatus = req.Payment.Status
	}

	receipt := &domain.Receipt{
		ID:              uuid.New().String(),
		TripID:          req.Trip.ID,
		RideID:          req.Ride.ID,
		DriverID:        req.Trip.DriverID,
		RiderID:         req.Ride.RiderID,
		PickupLat:       req.Ride.PickupLat,
		PickupLng:       req.Ride.PickupLng,
		DestinationLat:  req.Ride.DestinationLat,
		DestinationLng:  req.Ride.DestinationLng,
		BaseFare:        baseFare,
		SurgeMultiplier: surgeMultiplier,
		SurgeAmount:     surgeAmount,
		TotalFare:       totalFare,
		PaymentMethod:   req.Ride.PaymentMethod,
		PaymentStatus:   paymentStatus,
		Duration:        duration,
		Distance:        distance,
		StartedAt:       req.Trip.StartedAt,
		EndedAt:         req.Trip.EndedAt,
		CreatedAt:       time.Now(),
	}

	// Notify rider that receipt is ready
	if s.notificationService != nil {
		_ = s.notificationService.NotifyReceiptReady(ctx, receipt)
	}

	return receipt, nil
}

// calculateBaseFare calculates the base fare before surge.
func (s *ReceiptService) calculateBaseFare(trip *domain.Trip) float64 {
	const (
		baseFare      = 2.0
		perMinuteRate = 0.5
		minimumFare   = 5.0
	)

	duration := trip.EndedAt.Sub(trip.StartedAt) - trip.TotalPaused
	minutes := duration.Minutes()

	fare := baseFare + (minutes * perMinuteRate)
	if fare < minimumFare {
		return minimumFare
	}

	return fare
}

// estimateDistance estimates distance using Haversine formula.
func (s *ReceiptService) estimateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Simplified estimation using Euclidean approximation
	// In production, use actual route distance from Maps API
	const kmPerDegree = 111.0 // Approximate km per degree at equator

	latDiff := (lat2 - lat1) * kmPerDegree
	lngDiff := (lng2 - lng1) * kmPerDegree * 0.85 // Adjust for latitude

	distance := latDiff*latDiff + lngDiff*lngDiff
	if distance > 0 {
		return distance // sqrt approximated for simplicity
	}
	return 0
}

// FormatReceipt formats the receipt as a string (for email/print).
func (s *ReceiptService) FormatReceipt(receipt *domain.Receipt) string {
	return `
=====================================
        RIDE RECEIPT
=====================================
Receipt ID: ` + receipt.ID + `
Trip ID: ` + receipt.TripID + `
Date: ` + receipt.CreatedAt.Format("Jan 02, 2006 3:04 PM") + `

TRIP DETAILS
-------------------------------------
Pickup:      (` + formatFloat(receipt.PickupLat) + `, ` + formatFloat(receipt.PickupLng) + `)
Destination: (` + formatFloat(receipt.DestinationLat) + `, ` + formatFloat(receipt.DestinationLng) + `)
Duration:    ` + formatDuration(receipt.Duration) + `
Distance:    ` + formatFloat(receipt.Distance) + ` km

FARE BREAKDOWN
-------------------------------------
Base Fare:        $` + formatFloat(receipt.BaseFare) + `
Surge (` + formatFloat(receipt.SurgeMultiplier) + `x):   $` + formatFloat(receipt.SurgeAmount) + `
-------------------------------------
TOTAL:            $` + formatFloat(receipt.TotalFare) + `

PAYMENT
-------------------------------------
Method: ` + string(receipt.PaymentMethod) + `
Status: ` + string(receipt.PaymentStatus) + `

=====================================
     Thank you for riding with us!
=====================================
`
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	return fmt.Sprintf("%d min", minutes)
}
