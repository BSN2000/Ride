package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ride/internal/service"
)

// TripHandler handles HTTP requests for trips.
type TripHandler struct {
	tripService *service.TripService
}

// NewTripHandler creates a new TripHandler.
func NewTripHandler(tripService *service.TripService) *TripHandler {
	return &TripHandler{tripService: tripService}
}

// TripResponse is the HTTP response for trip operations.
type TripResponse struct {
	TripID      string       `json:"trip_id"`
	RideID      string       `json:"ride_id"`
	DriverID    string       `json:"driver_id"`
	Status      string       `json:"status"`
	Fare        float64      `json:"fare"`
	StartedAt   string       `json:"started_at"`
	EndedAt     string       `json:"ended_at,omitempty"`
	PausedAt    string       `json:"paused_at,omitempty"`
	TotalPaused int64        `json:"total_paused_seconds,omitempty"`
	Payment     *PaymentInfo `json:"payment,omitempty"`
	Receipt     *ReceiptInfo `json:"receipt,omitempty"`
}

// PaymentInfo contains payment details in the response.
type PaymentInfo struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

// ReceiptInfo contains receipt details in the response.
type ReceiptInfo struct {
	ID              string  `json:"id"`
	BaseFare        float64 `json:"base_fare"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	SurgeAmount     float64 `json:"surge_amount"`
	TotalFare       float64 `json:"total_fare"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentStatus   string  `json:"payment_status"`
	DurationMinutes float64 `json:"duration_minutes"`
	DistanceKm      float64 `json:"distance_km"`
}

// EndTrip handles POST /v1/trips/:id/end
func (h *TripHandler) EndTrip(c *gin.Context) {
	tripID := c.Param("id")

	result, err := h.tripService.EndTrip(c.Request.Context(), service.EndTripRequest{
		TripID: tripID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response := TripResponse{
		TripID:      result.Trip.ID,
		RideID:      result.Trip.RideID,
		DriverID:    result.Trip.DriverID,
		Status:      string(result.Trip.Status),
		Fare:        result.Trip.Fare,
		StartedAt:   result.Trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		EndedAt:     result.Trip.EndedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalPaused: int64(result.Trip.TotalPaused.Seconds()),
	}

	if result.Payment != nil {
		response.Payment = &PaymentInfo{
			ID:     result.Payment.ID,
			Amount: result.Payment.Amount,
			Status: string(result.Payment.Status),
		}
	}

	if result.Receipt != nil {
		response.Receipt = &ReceiptInfo{
			ID:              result.Receipt.ID,
			BaseFare:        result.Receipt.BaseFare,
			SurgeMultiplier: result.Receipt.SurgeMultiplier,
			SurgeAmount:     result.Receipt.SurgeAmount,
			TotalFare:       result.Receipt.TotalFare,
			PaymentMethod:   string(result.Receipt.PaymentMethod),
			PaymentStatus:   string(result.Receipt.PaymentStatus),
			DurationMinutes: result.Receipt.Duration.Minutes(),
			DistanceKm:      result.Receipt.Distance,
		}
	}

	respondJSON(c, http.StatusOK, response)
}

// PauseTrip handles POST /v1/trips/:id/pause
func (h *TripHandler) PauseTrip(c *gin.Context) {
	tripID := c.Param("id")

	trip, err := h.tripService.PauseTrip(c.Request.Context(), service.PauseTripRequest{
		TripID: tripID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response := TripResponse{
		TripID:    trip.ID,
		RideID:    trip.RideID,
		DriverID:  trip.DriverID,
		Status:    string(trip.Status),
		Fare:      trip.Fare,
		StartedAt: trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		PausedAt:  trip.PausedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	respondJSON(c, http.StatusOK, response)
}

// ResumeTrip handles POST /v1/trips/:id/resume
func (h *TripHandler) ResumeTrip(c *gin.Context) {
	tripID := c.Param("id")

	trip, err := h.tripService.ResumeTrip(c.Request.Context(), service.ResumeTripRequest{
		TripID: tripID,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response := TripResponse{
		TripID:      trip.ID,
		RideID:      trip.RideID,
		DriverID:    trip.DriverID,
		Status:      string(trip.Status),
		Fare:        trip.Fare,
		StartedAt:   trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalPaused: int64(trip.TotalPaused.Seconds()),
	}

	respondJSON(c, http.StatusOK, response)
}

// GetTrip handles GET /v1/trips/:id
func (h *TripHandler) GetTrip(c *gin.Context) {
	tripID := c.Param("id")

	trip, err := h.tripService.GetTrip(c.Request.Context(), tripID)
	if err != nil {
		respondError(c, err)
		return
	}

	response := TripResponse{
		TripID:      trip.ID,
		RideID:      trip.RideID,
		DriverID:    trip.DriverID,
		Status:      string(trip.Status),
		Fare:        trip.Fare,
		StartedAt:   trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalPaused: int64(trip.TotalPaused.Seconds()),
	}

	if !trip.EndedAt.IsZero() {
		response.EndedAt = trip.EndedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	if !trip.PausedAt.IsZero() {
		response.PausedAt = trip.PausedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	respondJSON(c, http.StatusOK, response)
}

// GetAll handles GET /v1/trips
func (h *TripHandler) GetAll(c *gin.Context) {
	trips, err := h.tripService.GetAllTrips(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	var response []TripResponse
	for _, trip := range trips {
		tr := TripResponse{
			TripID:      trip.ID,
			RideID:      trip.RideID,
			DriverID:    trip.DriverID,
			Status:      string(trip.Status),
			Fare:        trip.Fare,
			StartedAt:   trip.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			TotalPaused: int64(trip.TotalPaused.Seconds()),
		}
		if !trip.EndedAt.IsZero() {
			tr.EndedAt = trip.EndedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		response = append(response, tr)
	}

	c.JSON(http.StatusOK, response)
}
