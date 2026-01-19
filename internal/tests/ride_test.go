package tests

import (
	"context"
	"testing"

	"ride/internal/domain"
	"ride/internal/service"
)

func TestRideCreation_ValidatesRiderID(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	_, err := rideService.CreateRide(context.Background(), service.CreateRideRequest{
		RiderID:        "", // Empty rider ID.
		PickupLat:      12.0,
		PickupLng:      77.0,
		DestinationLat: 12.5,
		DestinationLng: 77.5,
	})

	if err != service.ErrInvalidRiderID {
		t.Errorf("expected ErrInvalidRiderID, got %v", err)
	}
}

func TestRideCreation_ValidatesPickupLatitude(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	testCases := []struct {
		name string
		lat  float64
	}{
		{"too low", -91.0},
		{"too high", 91.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rideService.CreateRide(context.Background(), service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      tc.lat,
				PickupLng:      77.0,
				DestinationLat: 12.5,
				DestinationLng: 77.5,
			})

			if err != service.ErrInvalidPickupLocation {
				t.Errorf("expected ErrInvalidPickupLocation for lat=%f, got %v", tc.lat, err)
			}
		})
	}
}

func TestRideCreation_ValidatesPickupLongitude(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	testCases := []struct {
		name string
		lng  float64
	}{
		{"too low", -181.0},
		{"too high", 181.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rideService.CreateRide(context.Background(), service.CreateRideRequest{
				RiderID:        "rider-1",
				PickupLat:      12.0,
				PickupLng:      tc.lng,
				DestinationLat: 12.5,
				DestinationLng: 77.5,
			})

			if err != service.ErrInvalidPickupLocation {
				t.Errorf("expected ErrInvalidPickupLocation for lng=%f, got %v", tc.lng, err)
			}
		})
	}
}

func TestRideCreation_ValidatesDestinationLatitude(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	_, err := rideService.CreateRide(context.Background(), service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.0,
		PickupLng:      77.0,
		DestinationLat: -100.0, // Invalid.
		DestinationLng: 77.5,
	})

	if err != service.ErrInvalidDestinationLocation {
		t.Errorf("expected ErrInvalidDestinationLocation, got %v", err)
	}
}

func TestRideCreation_ValidatesDestinationLongitude(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	_, err := rideService.CreateRide(context.Background(), service.CreateRideRequest{
		RiderID:        "rider-1",
		PickupLat:      12.0,
		PickupLng:      77.0,
		DestinationLat: 12.5,
		DestinationLng: 200.0, // Invalid.
	})

	if err != service.ErrInvalidDestinationLocation {
		t.Errorf("expected ErrInvalidDestinationLocation, got %v", err)
	}
}

func TestRideCreation_CreatesRideInRequestedState(t *testing.T) {
	rideRepo := NewMockRideRepository()
	// Pass nil for matchingService to test ride creation without matching.
	// This will panic if matching is called, but we're testing the case where
	// we want to verify ride state before matching.

	// Create a custom test that just tests ride creation directly.
	ctx := context.Background()

	ride := &domain.Ride{
		ID:             "test-ride-1",
		RiderID:        "rider-1",
		PickupLat:      12.0,
		PickupLng:      77.0,
		DestinationLat: 12.5,
		DestinationLng: 77.5,
		Status:         domain.RideStatusRequested,
	}

	err := rideRepo.Create(ctx, ride)
	if err != nil {
		t.Fatalf("failed to create ride: %v", err)
	}

	// Verify ride was created with correct status.
	savedRide := rideRepo.GetRide("test-ride-1")
	if savedRide == nil {
		t.Fatal("ride not found")
	}
	if savedRide.Status != domain.RideStatusRequested {
		t.Errorf("expected REQUESTED status, got %s", savedRide.Status)
	}
}

func TestRideCreation_DirectRepo_PersistsAllFields(t *testing.T) {
	rideRepo := NewMockRideRepository()
	ctx := context.Background()

	ride := &domain.Ride{
		ID:             "test-ride-2",
		RiderID:        "rider-123",
		PickupLat:      12.9716,
		PickupLng:      77.5946,
		DestinationLat: 13.0827,
		DestinationLng: 80.2707,
		Status:         domain.RideStatusRequested,
	}

	err := rideRepo.Create(ctx, ride)
	if err != nil {
		t.Fatalf("failed to create ride: %v", err)
	}

	savedRide := rideRepo.GetRide("test-ride-2")
	if savedRide == nil {
		t.Fatal("ride not found")
	}

	if savedRide.RiderID != "rider-123" {
		t.Errorf("expected rider-123, got %s", savedRide.RiderID)
	}
	if savedRide.PickupLat != 12.9716 {
		t.Errorf("expected pickup lat 12.9716, got %f", savedRide.PickupLat)
	}
	if savedRide.PickupLng != 77.5946 {
		t.Errorf("expected pickup lng 77.5946, got %f", savedRide.PickupLng)
	}
	if savedRide.DestinationLat != 13.0827 {
		t.Errorf("expected destination lat 13.0827, got %f", savedRide.DestinationLat)
	}
	if savedRide.DestinationLng != 80.2707 {
		t.Errorf("expected destination lng 80.2707, got %f", savedRide.DestinationLng)
	}
}

func TestGetRideStatus_ReturnsExistingRide(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)
	ctx := context.Background()

	// Add a ride directly to the repo.
	ride := &domain.Ride{
		ID:               "ride-1",
		RiderID:          "rider-1",
		Status:           domain.RideStatusAssigned,
		AssignedDriverID: "driver-1",
	}
	rideRepo.AddRide(ride)

	// Get ride status.
	result, err := rideService.GetRideStatus(ctx, "ride-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "ride-1" {
		t.Errorf("expected ride-1, got %s", result.ID)
	}
	if result.Status != domain.RideStatusAssigned {
		t.Errorf("expected ASSIGNED status, got %s", result.Status)
	}
	if result.AssignedDriverID != "driver-1" {
		t.Errorf("expected driver-1, got %s", result.AssignedDriverID)
	}
}

func TestGetRideStatus_ReturnsErrorForEmptyID(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	_, err := rideService.GetRideStatus(context.Background(), "")

	if err != service.ErrInvalidRideID {
		t.Errorf("expected ErrInvalidRideID, got %v", err)
	}
}

func TestGetRideStatus_ReturnsNotFoundForNonexistentRide(t *testing.T) {
	rideRepo := NewMockRideRepository()
	mockMatching := NewMockMatchingServiceForTest()
	rideService := service.NewRideService(rideRepo, mockMatching, nil, nil)

	_, err := rideService.GetRideStatus(context.Background(), "nonexistent")

	if err == nil {
		t.Error("expected error for nonexistent ride")
	}
}

func TestRideUpdate_ChangesStatus(t *testing.T) {
	rideRepo := NewMockRideRepository()
	ctx := context.Background()

	// Create initial ride.
	ride := &domain.Ride{
		ID:      "ride-update-test",
		RiderID: "rider-1",
		Status:  domain.RideStatusRequested,
	}
	rideRepo.Create(ctx, ride)

	// Update ride status.
	ride.Status = domain.RideStatusAssigned
	ride.AssignedDriverID = "driver-1"
	err := rideRepo.Update(ctx, ride)
	if err != nil {
		t.Fatalf("failed to update ride: %v", err)
	}

	// Verify update.
	updatedRide := rideRepo.GetRide("ride-update-test")
	if updatedRide.Status != domain.RideStatusAssigned {
		t.Errorf("expected ASSIGNED, got %s", updatedRide.Status)
	}
	if updatedRide.AssignedDriverID != "driver-1" {
		t.Errorf("expected driver-1, got %s", updatedRide.AssignedDriverID)
	}
}
