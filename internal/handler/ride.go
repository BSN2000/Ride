package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ride/internal/domain"
	"ride/internal/repository"
	"ride/internal/service"
)

// RideHandler handles HTTP requests for rides.
type RideHandler struct {
	rideService *service.RideService
	rideRepo    repository.RideRepository
}

// NewRideHandler creates a new RideHandler.
func NewRideHandler(rideService *service.RideService, rideRepo repository.RideRepository) *RideHandler {
	return &RideHandler{
		rideService: rideService,
		rideRepo:    rideRepo,
	}
}

// CreateRideRequest is the HTTP request body for creating a ride.
type CreateRideRequest struct {
	RiderID        string  `json:"rider_id"`
	PickupLat      float64 `json:"pickup_lat"`
	PickupLng      float64 `json:"pickup_lng"`
	DestinationLat float64 `json:"destination_lat"`
	DestinationLng float64 `json:"destination_lng"`
	Tier           string  `json:"tier,omitempty"`
	PaymentMethod  string  `json:"payment_method,omitempty"` // CASH, CARD, WALLET, UPI
}

// CancelRideRequest is the HTTP request body for cancelling a ride.
type CancelRideRequest struct {
	CancelledBy string `json:"cancelled_by"`
	Reason      string `json:"reason,omitempty"`
}

// CreateRideResponse is the HTTP response for creating a ride.
type CreateRideResponse struct {
	ID               string  `json:"id"`
	RiderID          string  `json:"rider_id"`
	PickupLat        float64 `json:"pickup_lat"`
	PickupLng        float64 `json:"pickup_lng"`
	DestinationLat   float64 `json:"destination_lat"`
	DestinationLng   float64 `json:"destination_lng"`
	Status           string  `json:"status"`
	AssignedDriverID string  `json:"assigned_driver_id,omitempty"`
	DriverAssigned   bool    `json:"driver_assigned"`
	SurgeMultiplier  float64 `json:"surge_multiplier"`
	SurgeActive      bool    `json:"surge_active"`
	PaymentMethod    string  `json:"payment_method"`
}

// GetRideResponse is the HTTP response for getting a ride.
type GetRideResponse struct {
	ID               string  `json:"id"`
	RiderID          string  `json:"rider_id"`
	PickupLat        float64 `json:"pickup_lat"`
	PickupLng        float64 `json:"pickup_lng"`
	DestinationLat   float64 `json:"destination_lat"`
	DestinationLng   float64 `json:"destination_lng"`
	Status           string  `json:"status"`
	AssignedDriverID string  `json:"assigned_driver_id,omitempty"`
	SurgeMultiplier  float64 `json:"surge_multiplier"`
	SurgeActive      bool    `json:"surge_active"`
	PaymentMethod    string  `json:"payment_method"`
	CancelledAt      string  `json:"cancelled_at,omitempty"`
	CancelReason     string  `json:"cancel_reason,omitempty"`
}

// CreateRide handles POST /v1/rides
func (h *RideHandler) CreateRide(c *gin.Context) {
	var req CreateRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Validate payment method
	paymentMethod, err := service.ValidatePaymentMethod(req.PaymentMethod)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := h.rideService.CreateRide(c.Request.Context(), service.CreateRideRequest{
		RiderID:        req.RiderID,
		PickupLat:      req.PickupLat,
		PickupLng:      req.PickupLng,
		DestinationLat: req.DestinationLat,
		DestinationLng: req.DestinationLng,
		Tier:           domain.DriverTier(req.Tier),
		PaymentMethod:  paymentMethod,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	respondJSON(c, http.StatusCreated, CreateRideResponse{
		ID:               result.Ride.ID,
		RiderID:          result.Ride.RiderID,
		PickupLat:        result.Ride.PickupLat,
		PickupLng:        result.Ride.PickupLng,
		DestinationLat:   result.Ride.DestinationLat,
		DestinationLng:   result.Ride.DestinationLng,
		Status:           string(result.Ride.Status),
		AssignedDriverID: result.DriverID,
		DriverAssigned:   result.DriverAssigned,
		SurgeMultiplier:  result.SurgeMultiplier,
		SurgeActive:      result.SurgeMultiplier > 1.0,
		PaymentMethod:    string(result.Ride.PaymentMethod),
	})
}

// GetRide handles GET /v1/rides/:id
func (h *RideHandler) GetRide(c *gin.Context) {
	rideID := c.Param("id")

	ride, err := h.rideService.GetRideStatus(c.Request.Context(), rideID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := GetRideResponse{
		ID:               ride.ID,
		RiderID:          ride.RiderID,
		PickupLat:        ride.PickupLat,
		PickupLng:        ride.PickupLng,
		DestinationLat:   ride.DestinationLat,
		DestinationLng:   ride.DestinationLng,
		Status:           string(ride.Status),
		AssignedDriverID: ride.AssignedDriverID,
		SurgeMultiplier:  ride.SurgeMultiplier,
		SurgeActive:      ride.SurgeMultiplier > 1.0,
		PaymentMethod:    string(ride.PaymentMethod),
	}

	if !ride.CancelledAt.IsZero() {
		response.CancelledAt = ride.CancelledAt.Format("2006-01-02T15:04:05Z07:00")
		response.CancelReason = ride.CancelReason
	}

	respondJSON(c, http.StatusOK, response)
}

// CancelRide handles POST /v1/rides/:id/cancel
func (h *RideHandler) CancelRide(c *gin.Context) {
	rideID := c.Param("id")

	var req CancelRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	ride, err := h.rideService.CancelRide(c.Request.Context(), service.CancelRideRequest{
		RideID:      rideID,
		CancelledBy: req.CancelledBy,
		Reason:      req.Reason,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response := GetRideResponse{
		ID:               ride.ID,
		RiderID:          ride.RiderID,
		PickupLat:        ride.PickupLat,
		PickupLng:        ride.PickupLng,
		DestinationLat:   ride.DestinationLat,
		DestinationLng:   ride.DestinationLng,
		Status:           string(ride.Status),
		AssignedDriverID: ride.AssignedDriverID,
		SurgeMultiplier:  ride.SurgeMultiplier,
		SurgeActive:      ride.SurgeMultiplier > 1.0,
		PaymentMethod:    string(ride.PaymentMethod),
		CancelledAt:      ride.CancelledAt.Format("2006-01-02T15:04:05Z07:00"),
		CancelReason:     ride.CancelReason,
	}

	respondJSON(c, http.StatusOK, response)
}

// GetAll handles GET /v1/rides
func (h *RideHandler) GetAll(c *gin.Context) {
	rides, err := h.rideRepo.GetAll(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	var response []GetRideResponse
	for _, r := range rides {
		response = append(response, GetRideResponse{
			ID:               r.ID,
			RiderID:          r.RiderID,
			PickupLat:        r.PickupLat,
			PickupLng:        r.PickupLng,
			DestinationLat:   r.DestinationLat,
			DestinationLng:   r.DestinationLng,
			Status:           string(r.Status),
			AssignedDriverID: r.AssignedDriverID,
			SurgeMultiplier:  r.SurgeMultiplier,
			SurgeActive:      r.SurgeMultiplier > 1.0,
		})
	}

	c.JSON(http.StatusOK, response)
}
