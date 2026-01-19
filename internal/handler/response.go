package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ride/internal/repository"
	"ride/internal/service"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// respondError sends an error response with the appropriate HTTP status code.
func respondError(c *gin.Context, err error) {
	code := mapErrorToHTTPStatus(err)
	c.JSON(code, ErrorResponse{Error: err.Error()})
}

// respondJSON sends a JSON response with the given status code.
func respondJSON(c *gin.Context, code int, data any) {
	c.JSON(code, data)
}

// mapErrorToHTTPStatus maps service/repository errors to HTTP status codes.
func mapErrorToHTTPStatus(err error) int {
	switch {
	// Not found errors
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound

	// Validation errors - Bad Request
	case errors.Is(err, service.ErrInvalidRiderID),
		errors.Is(err, service.ErrInvalidRideID),
		errors.Is(err, service.ErrInvalidDriverID),
		errors.Is(err, service.ErrInvalidTripID),
		errors.Is(err, service.ErrInvalidPickupLocation),
		errors.Is(err, service.ErrInvalidDestinationLocation),
		errors.Is(err, service.ErrInvalidLocation),
		errors.Is(err, service.ErrInvalidPaymentAmount),
		errors.Is(err, service.ErrInvalidPaymentID),
		errors.Is(err, service.ErrInvalidPaymentMethod):
		return http.StatusBadRequest

	// Conflict errors
	case errors.Is(err, service.ErrDriverHasActiveTrip),
		errors.Is(err, service.ErrTripAlreadyEnded),
		errors.Is(err, service.ErrTripNotStarted),
		errors.Is(err, service.ErrTripNotPaused),
		errors.Is(err, service.ErrRideNotInRequestedState),
		errors.Is(err, service.ErrRideAlreadyCancelled),
		errors.Is(err, service.ErrRideCannotBeCancelled),
		errors.Is(err, service.ErrTripInProgress):
		return http.StatusConflict

	// Forbidden/Business rule errors
	case errors.Is(err, service.ErrRideNotAssigned),
		errors.Is(err, service.ErrDriverNotAssignedToRide):
		return http.StatusForbidden

	// Service unavailable
	case errors.Is(err, service.ErrNoDriverAvailable):
		return http.StatusServiceUnavailable

	// Default to internal server error
	default:
		return http.StatusInternalServerError
	}
}
