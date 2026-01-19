package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ride/internal/domain"
	"ride/internal/repository"
)

// MatchingServiceInterface defines the matching service contract.
// This interface allows for testing with mock implementations.
type MatchingServiceInterface interface {
	Match(ctx context.Context, req MatchRequest) (*MatchResult, error)
}

// Ensure MatchingService implements MatchingServiceInterface.
var _ MatchingServiceInterface = (*MatchingService)(nil)

// RideService handles ride operations.
type RideService struct {
	rideRepo            repository.RideRepository
	matchingService     MatchingServiceInterface
	surgeService        *SurgeService
	notificationService *NotificationService
}

// NewRideService creates a new RideService.
func NewRideService(
	rideRepo repository.RideRepository,
	matchingService MatchingServiceInterface,
	surgeService *SurgeService,
	notificationService *NotificationService,
) *RideService {
	return &RideService{
		rideRepo:            rideRepo,
		matchingService:     matchingService,
		surgeService:        surgeService,
		notificationService: notificationService,
	}
}

// CreateRideRequest contains the parameters for creating a ride.
type CreateRideRequest struct {
	RiderID        string
	PickupLat      float64
	PickupLng      float64
	DestinationLat float64
	DestinationLng float64
	Tier           domain.DriverTier    // Optional: empty means any tier
	PaymentMethod  domain.PaymentMethod // Optional: defaults to CASH
}

// CreateRideResponse contains the result of creating a ride.
type CreateRideResponse struct {
	Ride            *domain.Ride
	DriverAssigned  bool
	DriverID        string
	SurgeMultiplier float64
}

// CreateRide creates a new ride and triggers matching.
func (s *RideService) CreateRide(ctx context.Context, req CreateRideRequest) (*CreateRideResponse, error) {
	// Validate input.
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Calculate surge multiplier based on supply/demand at pickup location.
	surgeMultiplier := 1.0
	if s.surgeService != nil {
		surgeMultiplier = s.surgeService.GetMultiplier(ctx, req.PickupLat, req.PickupLng)
	}

	// Set default payment method if not specified
	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = domain.PaymentMethodCash
	}

	// Create ride in REQUESTED state with surge.
	ride := &domain.Ride{
		ID:              uuid.New().String(),
		RiderID:         req.RiderID,
		PickupLat:       req.PickupLat,
		PickupLng:       req.PickupLng,
		DestinationLat:  req.DestinationLat,
		DestinationLng:  req.DestinationLng,
		Status:          domain.RideStatusRequested,
		SurgeMultiplier: surgeMultiplier,
		PaymentMethod:   paymentMethod,
		CreatedAt:       time.Now(),
	}

	if err := s.rideRepo.Create(ctx, ride); err != nil {
		return nil, err
	}

	// Trigger matching synchronously.
	matchResult, err := s.matchingService.Match(ctx, MatchRequest{
		RideID: ride.ID,
		Lat:    req.PickupLat,
		Lng:    req.PickupLng,
		Tier:   req.Tier,
	})

	// If matching fails, still return the ride (in REQUESTED state).
	if err != nil {
		if err == ErrNoDriverAvailable {
			return &CreateRideResponse{
				Ride:            ride,
				DriverAssigned:  false,
				SurgeMultiplier: surgeMultiplier,
			}, nil
		}
		return nil, err
	}

	return &CreateRideResponse{
		Ride:            matchResult.Ride,
		DriverAssigned:  true,
		DriverID:        matchResult.DriverID,
		SurgeMultiplier: surgeMultiplier,
	}, nil
}

// GetRideStatus retrieves the current status of a ride.
func (s *RideService) GetRideStatus(ctx context.Context, rideID string) (*domain.Ride, error) {
	if rideID == "" {
		return nil, ErrInvalidRideID
	}

	return s.rideRepo.GetByID(ctx, rideID)
}

// validateCreateRequest validates the create ride request.
func (s *RideService) validateCreateRequest(req CreateRideRequest) error {
	if req.RiderID == "" {
		return ErrInvalidRiderID
	}

	if !isValidLatitude(req.PickupLat) {
		return ErrInvalidPickupLocation
	}

	if !isValidLongitude(req.PickupLng) {
		return ErrInvalidPickupLocation
	}

	if !isValidLatitude(req.DestinationLat) {
		return ErrInvalidDestinationLocation
	}

	if !isValidLongitude(req.DestinationLng) {
		return ErrInvalidDestinationLocation
	}

	return nil
}

func isValidLatitude(lat float64) bool {
	return lat >= -90 && lat <= 90
}

func isValidLongitude(lng float64) bool {
	return lng >= -180 && lng <= 180
}

// CancelRideRequest contains the parameters for cancelling a ride.
type CancelRideRequest struct {
	RideID      string
	CancelledBy string // UserID or DriverID
	Reason      string
}

// CancelRide cancels a ride request.
func (s *RideService) CancelRide(ctx context.Context, req CancelRideRequest) (*domain.Ride, error) {
	if req.RideID == "" {
		return nil, ErrInvalidRideID
	}

	ride, err := s.rideRepo.GetByID(ctx, req.RideID)
	if err != nil {
		return nil, err
	}

	// Check if ride can be cancelled
	if ride.Status == domain.RideStatusCancelled {
		return nil, ErrRideAlreadyCancelled
	}

	// Only REQUESTED and ASSIGNED rides can be cancelled
	// If there's an active trip, it cannot be cancelled
	if ride.Status != domain.RideStatusRequested && ride.Status != domain.RideStatusAssigned {
		return nil, ErrRideCannotBeCancelled
	}

	// Update ride status
	ride.Status = domain.RideStatusCancelled
	ride.CancelledAt = time.Now()
	ride.CancelReason = req.Reason

	if err := s.rideRepo.Update(ctx, ride); err != nil {
		return nil, err
	}

	// Send notification to affected party
	if s.notificationService != nil {
		_ = s.notificationService.NotifyRideCancelled(ctx, ride, req.CancelledBy, req.Reason)
	}

	return ride, nil
}

// ValidatePaymentMethod validates a payment method string.
func ValidatePaymentMethod(method string) (domain.PaymentMethod, error) {
	switch domain.PaymentMethod(method) {
	case domain.PaymentMethodCash, domain.PaymentMethodCard,
		domain.PaymentMethodWallet, domain.PaymentMethodUPI:
		return domain.PaymentMethod(method), nil
	case "":
		return domain.PaymentMethodCash, nil // Default to cash
	default:
		return "", ErrInvalidPaymentMethod
	}
}
