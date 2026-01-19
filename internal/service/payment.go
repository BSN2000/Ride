package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ride/internal/domain"
	"ride/internal/repository"
)

// PSP is the interface for a Payment Service Provider.
type PSP interface {
	Charge(ctx context.Context, amount float64) (bool, error)
}

// MockPSP is a mock implementation of PSP for testing.
type MockPSP struct{}

// NewMockPSP creates a new mock PSP.
func NewMockPSP() *MockPSP {
	return &MockPSP{}
}

// Charge simulates a payment charge. Always succeeds.
func (p *MockPSP) Charge(ctx context.Context, amount float64) (bool, error) {
	// Mock implementation: always succeeds.
	return true, nil
}

// PaymentService handles payment operations.
type PaymentService struct {
	paymentRepo repository.PaymentRepository
	psp         PSP
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(paymentRepo repository.PaymentRepository, psp PSP) *PaymentService {
	return &PaymentService{
		paymentRepo: paymentRepo,
		psp:         psp,
	}
}

// ProcessPaymentRequest contains the parameters for processing a payment.
type ProcessPaymentRequest struct {
	TripID string
	Amount float64
}

// ProcessPayment processes a payment for a trip with idempotency support.
func (s *PaymentService) ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*domain.Payment, error) {
	if req.TripID == "" {
		return nil, ErrInvalidTripID
	}

	if req.Amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	// Generate idempotency key based on trip ID.
	idempotencyKey := fmt.Sprintf("payment:%s", req.TripID)

	// Check for existing payment (idempotency).
	existingPayment, err := s.paymentRepo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if existingPayment != nil {
		// Payment already exists - return it (idempotent).
		return existingPayment, nil
	}

	// Create payment in PENDING state.
	payment := &domain.Payment{
		ID:             uuid.New().String(),
		TripID:         req.TripID,
		Amount:         req.Amount,
		Status:         domain.PaymentStatusPending,
		IdempotencyKey: idempotencyKey,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	// Call PSP (mocked).
	success, err := s.psp.Charge(ctx, req.Amount)
	if err != nil {
		// PSP error - mark as failed.
		_ = s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed)
		payment.Status = domain.PaymentStatusFailed
		return payment, nil
	}

	// Update payment status based on PSP result.
	if success {
		if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusSuccess); err != nil {
			return nil, err
		}
		payment.Status = domain.PaymentStatusSuccess
	} else {
		if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed); err != nil {
			return nil, err
		}
		payment.Status = domain.PaymentStatusFailed
	}

	return payment, nil
}

// GetPayment retrieves a payment by ID.
func (s *PaymentService) GetPayment(ctx context.Context, paymentID string) (*domain.Payment, error) {
	if paymentID == "" {
		return nil, ErrInvalidPaymentID
	}

	return s.paymentRepo.GetByID(ctx, paymentID)
}
