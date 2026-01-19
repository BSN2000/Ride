package tests

import (
	"context"
	"sync"
	"testing"

	"ride/internal/domain"
	"ride/internal/service"
)

// ──────────────────────────────────────────────
// 1. RIDE CREATION EDGE CASES
// ──────────────────────────────────────────────

func TestRideCreation_ValidInput_Succeeds(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()

	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	resp, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp == nil || resp.Ride == nil {
		t.Fatal("expected ride to be created")
	}

	if resp.Ride.ID == "" {
		t.Error("expected ride ID to be set")
	}

	if resp.Ride.RiderID != req.RiderID {
		t.Errorf("expected rider ID %s, got %s", req.RiderID, resp.Ride.RiderID)
	}
}

func TestRideCreation_MissingCoordinates_Fails(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		req     service.CreateRideRequest
		wantErr bool
	}{
		{
			name: "missing pickup latitude (zero)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      0,
				PickupLng:      77.5946,
				DestinationLat: 12.2958,
				DestinationLng: 76.6394,
			},
			wantErr: false, // 0 is a valid latitude
		},
		{
			name: "missing destination latitude (zero)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.9716,
				PickupLng:      77.5946,
				DestinationLat: 0,
				DestinationLng: 76.6394,
			},
			wantErr: false, // 0 is a valid latitude
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rideRepo := NewMockRideRepository()
			matchingService := NewMockMatchingServiceForTest()
			rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

			_, err := rideService.CreateRide(context.Background(), tc.req)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestRideCreation_MissingRiderID_Fails(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "", // Missing rider ID
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	_, err := rideService.CreateRide(context.Background(), req)
	if err == nil {
		t.Error("expected error for missing rider ID, got nil")
	}
}

func TestRideCreation_InvalidCoordinates_Rejected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		req     service.CreateRideRequest
		wantErr bool
	}{
		{
			name: "latitude out of range (too high)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      91.0, // Invalid: max is 90
				PickupLng:      77.5946,
				DestinationLat: 12.2958,
				DestinationLng: 76.6394,
			},
			wantErr: true,
		},
		{
			name: "latitude out of range (too low)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      -91.0, // Invalid: min is -90
				PickupLng:      77.5946,
				DestinationLat: 12.2958,
				DestinationLng: 76.6394,
			},
			wantErr: true,
		},
		{
			name: "longitude out of range (too high)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.9716,
				PickupLng:      181.0, // Invalid: max is 180
				DestinationLat: 12.2958,
				DestinationLng: 76.6394,
			},
			wantErr: true,
		},
		{
			name: "longitude out of range (too low)",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.9716,
				PickupLng:      -181.0, // Invalid: min is -180
				DestinationLat: 12.2958,
				DestinationLng: 76.6394,
			},
			wantErr: true,
		},
		{
			name: "destination latitude out of range",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.9716,
				PickupLng:      77.5946,
				DestinationLat: 95.0, // Invalid
				DestinationLng: 76.6394,
			},
			wantErr: true,
		},
		{
			name: "destination longitude out of range",
			req: service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.9716,
				PickupLng:      77.5946,
				DestinationLat: 12.2958,
				DestinationLng: 200.0, // Invalid
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rideRepo := NewMockRideRepository()
			matchingService := NewMockMatchingServiceForTest()
			rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

			_, err := rideService.CreateRide(context.Background(), tc.req)
			if tc.wantErr && err == nil {
				t.Error("expected error for invalid coordinates, got nil")
			}
		})
	}
}

func TestRideCreation_AlwaysInRequestedState(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	resp, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When no driver is matched, status should be REQUESTED
	if resp.Ride.Status != domain.RideStatusRequested {
		t.Errorf("expected ride status %s, got %s", domain.RideStatusRequested, resp.Ride.Status)
	}

	// Verify in repository as well
	storedRide := rideRepo.GetRide(resp.Ride.ID)
	if storedRide == nil {
		t.Fatal("expected ride to be stored in repository")
	}
	if storedRide.Status != domain.RideStatusRequested {
		t.Errorf("expected stored ride status %s, got %s", domain.RideStatusRequested, storedRide.Status)
	}
}

func TestRideCreation_PersistsAllFields(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "rider-123",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	resp, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are correctly persisted
	storedRide := rideRepo.GetRide(resp.Ride.ID)
	if storedRide == nil {
		t.Fatal("ride not found in repository")
	}

	if storedRide.RiderID != req.RiderID {
		t.Errorf("rider ID mismatch: got %s, want %s", storedRide.RiderID, req.RiderID)
	}
	if storedRide.PickupLat != req.PickupLat {
		t.Errorf("pickup lat mismatch: got %f, want %f", storedRide.PickupLat, req.PickupLat)
	}
	if storedRide.PickupLng != req.PickupLng {
		t.Errorf("pickup lng mismatch: got %f, want %f", storedRide.PickupLng, req.PickupLng)
	}
	if storedRide.DestinationLat != req.DestinationLat {
		t.Errorf("destination lat mismatch: got %f, want %f", storedRide.DestinationLat, req.DestinationLat)
	}
	if storedRide.DestinationLng != req.DestinationLng {
		t.Errorf("destination lng mismatch: got %f, want %f", storedRide.DestinationLng, req.DestinationLng)
	}
}

func TestRideCreation_MultipleRidesAreDistinct(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	// First creation
	resp1, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("first creation failed: %v", err)
	}

	// Second creation with same request
	resp2, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("second creation failed: %v", err)
	}

	// Both should succeed but be different rides (without idempotency key)
	if resp1.Ride.ID == resp2.Ride.ID {
		t.Error("expected different ride IDs for separate requests")
	}

	// Verify both exist
	if rideRepo.CountRides() != 2 {
		t.Errorf("expected 2 rides, got %d", rideRepo.CountRides())
	}
}

func TestRideCreation_RepoCreateIsCalled(t *testing.T) {
	t.Parallel()

	rideRepo := NewMockRideRepository()
	matchingService := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, matchingService, nil, nil)

	req := service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 12.2958,
		DestinationLng: 76.6394,
	}

	_, err := rideService.CreateRide(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rideRepo.CreateCallCount != 1 {
		t.Errorf("expected Create to be called once, called %d times", rideRepo.CreateCallCount)
	}
}

// ──────────────────────────────────────────────
// MOCK MATCHING SERVICE FOR TESTS
// ──────────────────────────────────────────────

// MockMatchingServiceForTest implements MatchingServiceInterface for testing.
type MockMatchingServiceForTest struct {
	mu        sync.Mutex
	callCount int
	result    *service.MatchResult
	err       error
}

// NewMockMatchingServiceForTest creates a new mock matching service.
func NewMockMatchingServiceForTest() *MockMatchingServiceForTest {
	return &MockMatchingServiceForTest{
		err: service.ErrNoDriverAvailable, // Default: no driver available
	}
}

var _ service.MatchingServiceInterface = (*MockMatchingServiceForTest)(nil)

func (m *MockMatchingServiceForTest) Match(ctx context.Context, req service.MatchRequest) (*service.MatchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func (m *MockMatchingServiceForTest) SetResult(result *service.MatchResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.result = result
	m.err = err
}

func (m *MockMatchingServiceForTest) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}
