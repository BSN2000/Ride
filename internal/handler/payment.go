package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ride/internal/service"
)

// PaymentHandler handles HTTP requests for payments.
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler.
func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// ProcessPaymentRequest is the HTTP request body for processing a payment.
type ProcessPaymentRequest struct {
	TripID string  `json:"trip_id"`
	Amount float64 `json:"amount"`
}

// PaymentResponse is the HTTP response for payment operations.
type PaymentResponse struct {
	ID             string  `json:"id"`
	TripID         string  `json:"trip_id"`
	Amount         float64 `json:"amount"`
	Status         string  `json:"status"`
	IdempotencyKey string  `json:"idempotency_key"`
}

// ProcessPayment handles POST /v1/payments
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.TripID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "trip_id is required"})
		return
	}

	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "amount must be positive"})
		return
	}

	payment, err := h.paymentService.ProcessPayment(c.Request.Context(), service.ProcessPaymentRequest{
		TripID: req.TripID,
		Amount: req.Amount,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	respondJSON(c, http.StatusCreated, PaymentResponse{
		ID:             payment.ID,
		TripID:         payment.TripID,
		Amount:         payment.Amount,
		Status:         string(payment.Status),
		IdempotencyKey: payment.IdempotencyKey,
	})
}

// GetPayment handles GET /v1/payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	paymentID := c.Param("id")

	payment, err := h.paymentService.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		respondError(c, err)
		return
	}

	respondJSON(c, http.StatusOK, PaymentResponse{
		ID:             payment.ID,
		TripID:         payment.TripID,
		Amount:         payment.Amount,
		Status:         string(payment.Status),
		IdempotencyKey: payment.IdempotencyKey,
	})
}
