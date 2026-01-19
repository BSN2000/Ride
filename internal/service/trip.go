package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"ride/internal/domain"
	"ride/internal/repository"
	"ride/internal/repository/postgres"
)

// TripService handles trip operations.
type TripService struct {
	db                  *sql.DB
	tripRepo            repository.TripRepository
	rideRepo            repository.RideRepository
	driverRepo          repository.DriverRepository
	paymentService      *PaymentService
	notificationService *NotificationService
	receiptService      *ReceiptService
}

// NewTripService creates a new TripService.
func NewTripService(
	db *sql.DB,
	tripRepo repository.TripRepository,
	rideRepo repository.RideRepository,
	driverRepo repository.DriverRepository,
	paymentService *PaymentService,
	notificationService *NotificationService,
	receiptService *ReceiptService,
) *TripService {
	return &TripService{
		db:                  db,
		tripRepo:            tripRepo,
		rideRepo:            rideRepo,
		driverRepo:          driverRepo,
		paymentService:      paymentService,
		notificationService: notificationService,
		receiptService:      receiptService,
	}
}

// StartTripRequest contains the parameters for starting a trip.
type StartTripRequest struct {
	RideID   string
	DriverID string
}

// StartTrip creates a new trip when a driver accepts a ride.
func (s *TripService) StartTrip(ctx context.Context, req StartTripRequest) (*domain.Trip, error) {
	if req.RideID == "" {
		return nil, ErrInvalidRideID
	}

	if req.DriverID == "" {
		return nil, ErrInvalidDriverID
	}

	// Check if driver already has an active trip.
	existingTrip, err := s.tripRepo.GetActiveByDriverID(ctx, req.DriverID)
	if err != nil {
		return nil, err
	}

	if existingTrip != nil {
		return nil, ErrDriverHasActiveTrip
	}

	// Verify ride is in ASSIGNED state and assigned to this driver.
	ride, err := s.rideRepo.GetByID(ctx, req.RideID)
	if err != nil {
		return nil, err
	}

	if ride.Status != domain.RideStatusAssigned {
		return nil, ErrRideNotAssigned
	}

	if ride.AssignedDriverID != req.DriverID {
		return nil, ErrDriverNotAssignedToRide
	}

	// Use transaction to create trip and update ride status.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create transaction-scoped repositories.
	txTripRepo := postgres.NewTripRepositoryWithTx(tx)
	txRideRepo := postgres.NewRideRepositoryWithTx(tx)
	txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)

	// Create trip in STARTED state.
	trip := &domain.Trip{
		ID:        uuid.New().String(),
		RideID:    req.RideID,
		DriverID:  req.DriverID,
		Status:    domain.TripStatusStarted,
		Fare:      0,
		StartedAt: time.Now(),
	}

	if err = txTripRepo.Create(ctx, trip); err != nil {
		return nil, err
	}

	// Update ride status to IN_TRIP.
	ride.Status = domain.RideStatusInTrip
	if err = txRideRepo.Update(ctx, ride); err != nil {
		return nil, err
	}

	// Update driver status to ON_TRIP.
	if err = txDriverRepo.UpdateStatus(ctx, req.DriverID, domain.DriverStatusOnTrip); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return trip, nil
}

// EndTripRequest contains the parameters for ending a trip.
type EndTripRequest struct {
	TripID string
}

// EndTripResponse contains the result of ending a trip.
type EndTripResponse struct {
	Trip    *domain.Trip
	Payment *domain.Payment
	Receipt *domain.Receipt
}

// EndTrip ends a trip, calculates fare, and triggers payment.
func (s *TripService) EndTrip(ctx context.Context, req EndTripRequest) (*EndTripResponse, error) {
	if req.TripID == "" {
		return nil, ErrInvalidTripID
	}

	// Get trip.
	trip, err := s.tripRepo.GetByID(ctx, req.TripID)
	if err != nil {
		return nil, err
	}

	if trip.Status == domain.TripStatusEnded {
		return nil, ErrTripAlreadyEnded
	}

	// If trip was paused, add remaining paused time
	if trip.Status == domain.TripStatusPaused && !trip.PausedAt.IsZero() {
		trip.TotalPaused += time.Since(trip.PausedAt)
	}

	// Get ride to retrieve surge multiplier.
	ride, err := s.rideRepo.GetByID(ctx, trip.RideID)
	if err != nil {
		return nil, err
	}

	// Calculate fare with surge applied.
	endTime := time.Now()
	baseFare := s.calculateFare(trip.StartedAt, endTime, trip.TotalPaused)
	surgeMultiplier := ride.SurgeMultiplier
	if surgeMultiplier < 1.0 {
		surgeMultiplier = 1.0 // Default to no surge if not set
	}
	fare := baseFare * surgeMultiplier

	// Use transaction to end trip, update ride status, and reset driver status.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Create transaction-scoped repositories.
	txTripRepo := postgres.NewTripRepositoryWithTx(tx)
	txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)
	txRideRepo := postgres.NewRideRepositoryWithTx(tx)

	// Update trip.
	trip.Status = domain.TripStatusEnded
	trip.Fare = fare
	trip.EndedAt = endTime

	if err = txTripRepo.Update(ctx, trip); err != nil {
		return nil, err
	}

	// Update ride status to COMPLETED.
	ride.Status = domain.RideStatusCompleted
	if err = txRideRepo.Update(ctx, ride); err != nil {
		return nil, err
	}

	// Reset driver status to ONLINE.
	if err = txDriverRepo.UpdateStatus(ctx, trip.DriverID, domain.DriverStatusOnline); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// Trigger payment (after transaction commits).
	var payment *domain.Payment
	payment, err = s.paymentService.ProcessPayment(ctx, ProcessPaymentRequest{
		TripID: trip.ID,
		Amount: fare,
	})
	if err != nil {
		// Log error but don't fail - trip is ended.
		// Payment can be retried later.
		payment = nil
	}

	// Send notifications
	if s.notificationService != nil {
		_ = s.notificationService.NotifyTripEnded(ctx, trip, ride.RiderID, fare)
		if payment != nil {
			if payment.Status == domain.PaymentStatusSuccess {
				_ = s.notificationService.NotifyPaymentSuccess(ctx, payment, ride.RiderID)
			} else if payment.Status == domain.PaymentStatusFailed {
				_ = s.notificationService.NotifyPaymentFailed(ctx, payment, ride.RiderID)
			}
		}
	}

	// Generate receipt
	var receipt *domain.Receipt
	if s.receiptService != nil {
		receipt, _ = s.receiptService.GenerateReceipt(ctx, GenerateReceiptRequest{
			Trip:    trip,
			Ride:    ride,
			Payment: payment,
		})
	}

	return &EndTripResponse{
		Trip:    trip,
		Payment: payment,
		Receipt: receipt,
	}, nil
}

// GetTrip retrieves a trip by ID.
func (s *TripService) GetTrip(ctx context.Context, tripID string) (*domain.Trip, error) {
	if tripID == "" {
		return nil, ErrInvalidTripID
	}

	return s.tripRepo.GetByID(ctx, tripID)
}

// GetAllTrips retrieves all trips.
func (s *TripService) GetAllTrips(ctx context.Context) ([]*domain.Trip, error) {
	return s.tripRepo.GetAll(ctx)
}

// PauseTripRequest contains the parameters for pausing a trip.
type PauseTripRequest struct {
	TripID string
}

// PauseTrip pauses an active trip.
func (s *TripService) PauseTrip(ctx context.Context, req PauseTripRequest) (*domain.Trip, error) {
	if req.TripID == "" {
		return nil, ErrInvalidTripID
	}

	trip, err := s.tripRepo.GetByID(ctx, req.TripID)
	if err != nil {
		return nil, err
	}

	if trip.Status != domain.TripStatusStarted {
		return nil, ErrTripNotStarted
	}

	// Update trip status to paused
	trip.Status = domain.TripStatusPaused
	trip.PausedAt = time.Now()

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		return nil, err
	}

	// Send notification
	if s.notificationService != nil {
		ride, _ := s.rideRepo.GetByID(ctx, trip.RideID)
		if ride != nil {
			_ = s.notificationService.NotifyTripPaused(ctx, trip, ride.RiderID)
		}
	}

	return trip, nil
}

// ResumeTripRequest contains the parameters for resuming a trip.
type ResumeTripRequest struct {
	TripID string
}

// ResumeTrip resumes a paused trip.
func (s *TripService) ResumeTrip(ctx context.Context, req ResumeTripRequest) (*domain.Trip, error) {
	if req.TripID == "" {
		return nil, ErrInvalidTripID
	}

	trip, err := s.tripRepo.GetByID(ctx, req.TripID)
	if err != nil {
		return nil, err
	}

	if trip.Status != domain.TripStatusPaused {
		return nil, ErrTripNotPaused
	}

	// Calculate paused duration and add to total
	pausedDuration := time.Since(trip.PausedAt)
	trip.TotalPaused += pausedDuration

	// Update trip status to started
	trip.Status = domain.TripStatusStarted
	trip.PausedAt = time.Time{} // Reset paused time

	if err := s.tripRepo.Update(ctx, trip); err != nil {
		return nil, err
	}

	// Send notification
	if s.notificationService != nil {
		ride, _ := s.rideRepo.GetByID(ctx, trip.RideID)
		if ride != nil {
			_ = s.notificationService.NotifyTripResumed(ctx, trip, ride.RiderID)
		}
	}

	return trip, nil
}

// calculateFare calculates the fare based on trip duration.
// Simple implementation: $2 base + $0.50 per minute.
func (s *TripService) calculateFare(startTime, endTime time.Time, totalPaused time.Duration) float64 {
	const (
		baseFare      = 2.0
		perMinuteRate = 0.5
		minimumFare   = 5.0
	)

	// Subtract paused time from total duration
	duration := endTime.Sub(startTime) - totalPaused
	minutes := duration.Minutes()

	fare := baseFare + (minutes * perMinuteRate)

	if fare < minimumFare {
		return minimumFare
	}

	return fare
}
