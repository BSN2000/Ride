package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ride/internal/domain"
	"ride/internal/repository"
	"ride/internal/service"
)

// DriverHandler handles HTTP requests for drivers.
type DriverHandler struct {
	driverService *service.DriverService
	tripService   *service.TripService
	driverRepo    repository.DriverRepository
}

// NewDriverHandler creates a new DriverHandler.
func NewDriverHandler(driverService *service.DriverService, tripService *service.TripService, driverRepo repository.DriverRepository) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		tripService:   tripService,
		driverRepo:    driverRepo,
	}
}

// UpdateLocationRequest is the HTTP request body for updating driver location.
type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// AcceptRideRequest is the HTTP request body for accepting a ride.
type AcceptRideRequest struct {
	RideID string `json:"ride_id"`
}

// AcceptRideResponse is the HTTP response for accepting a ride.
type AcceptRideResponse struct {
	TripID    string `json:"trip_id"`
	RideID    string `json:"ride_id"`
	DriverID  string `json:"driver_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

// RegisterDriverRequest is the HTTP request body for driver registration.
type RegisterDriverRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Tier  string `json:"tier"`
}

// DriverResponse is the HTTP response for driver data.
type DriverResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
	Tier   string `json:"tier"`
}

// Register handles POST /v1/drivers/register
func (h *DriverHandler) Register(c *gin.Context) {
	var req RegisterDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "name and phone are required"})
		return
	}

	tier := domain.DriverTierBasic
	if req.Tier == "PREMIUM" {
		tier = domain.DriverTierPremium
	}

	// Check if driver already exists
	existing, err := h.driverRepo.GetByPhone(c.Request.Context(), req.Phone)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		respondError(c, err)
		return
	}

	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message": "Driver already registered",
			"driver":  DriverResponse{ID: existing.ID, Name: existing.Name, Phone: existing.Phone, Status: string(existing.Status), Tier: string(existing.Tier)},
		})
		return
	}

	// Create new driver
	driver := &domain.Driver{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Phone:  req.Phone,
		Status: domain.DriverStatusOffline,
		Tier:   tier,
	}

	if err := h.driverRepo.Create(c.Request.Context(), driver); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, DriverResponse{
		ID:     driver.ID,
		Name:   driver.Name,
		Phone:  driver.Phone,
		Status: string(driver.Status),
		Tier:   string(driver.Tier),
	})
}

// GetAll handles GET /v1/drivers
func (h *DriverHandler) GetAll(c *gin.Context) {
	drivers, err := h.driverRepo.GetAll(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	var response []DriverResponse
	for _, d := range drivers {
		response = append(response, DriverResponse{
			ID:     d.ID,
			Name:   d.Name,
			Phone:  d.Phone,
			Status: string(d.Status),
			Tier:   string(d.Tier),
		})
	}

	c.JSON(http.StatusOK, response)
}

// UpdateLocation handles POST /v1/drivers/:id/location
func (h *DriverHandler) UpdateLocation(c *gin.Context) {
	driverID := c.Param("id")

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	err := h.driverService.UpdateLocation(c.Request.Context(), service.UpdateLocationRequest{
		DriverID: driverID,
		Lat:      req.Lat,
		Lng:      req.Lng,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// AcceptRide handles POST /v1/drivers/:id/accept
func (h *DriverHandler) AcceptRide(c *gin.Context) {
	driverID := c.Param("id")

	var req AcceptRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	trip, err := h.tripService.StartTrip(c.Request.Context(), service.StartTripRequest{
		RideID:   req.RideID,
		DriverID: driverID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	respondJSON(c, http.StatusCreated, AcceptRideResponse{
		TripID:    trip.ID,
		RideID:    trip.RideID,
		DriverID:  trip.DriverID,
		Status:    string(trip.Status),
		StartedAt: trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
