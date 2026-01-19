package tests

import (
	"context"
	"testing"

	"ride/internal/domain"
	"ride/internal/service"
)

// ──────────────────────────────────────────────
// 2. DRIVER LOCATION UPDATE EDGE CASES
// ──────────────────────────────────────────────

func TestDriverLocationUpdate_WritesToRedisOnly(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	driverRepo := NewMockDriverRepository()

	// Add a driver to the repo
	driverRepo.AddDriver(&domain.Driver{
		ID:     "driver-1",
		Name:   "Test Driver",
		Phone:  "1234567890",
		Status: domain.DriverStatusOffline,
		Tier:   domain.DriverTierBasic,
	})

	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	req := service.UpdateLocationRequest{
		DriverID: "driver-1",
		Lat:      12.9716,
		Lng:      77.5946,
	}

	err := driverService.UpdateLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Redis was called
	if locationStore.UpdateLocationCallCount != 1 {
		t.Errorf("expected UpdateLocation to be called once, called %d times", locationStore.UpdateLocationCallCount)
	}

	// Verify location was stored
	if !locationStore.HasLocation("driver-1") {
		t.Error("expected driver location to be stored in Redis")
	}
}

func TestDriverLocationUpdate_InvalidLatitude_Rejected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		lat     float64
		lng     float64
		wantErr bool
	}{
		{
			name:    "latitude too high",
			lat:     91.0,
			lng:     77.5946,
			wantErr: true,
		},
		{
			name:    "latitude too low",
			lat:     -91.0,
			lng:     77.5946,
			wantErr: true,
		},
		{
			name:    "longitude too high",
			lat:     12.9716,
			lng:     181.0,
			wantErr: true,
		},
		{
			name:    "longitude too low",
			lat:     12.9716,
			lng:     -181.0,
			wantErr: true,
		},
		{
			name:    "valid coordinates",
			lat:     12.9716,
			lng:     77.5946,
			wantErr: false,
		},
		{
			name:    "edge case: max latitude",
			lat:     90.0,
			lng:     77.5946,
			wantErr: false,
		},
		{
			name:    "edge case: min latitude",
			lat:     -90.0,
			lng:     77.5946,
			wantErr: false,
		},
		{
			name:    "edge case: max longitude",
			lat:     12.9716,
			lng:     180.0,
			wantErr: false,
		},
		{
			name:    "edge case: min longitude",
			lat:     12.9716,
			lng:     -180.0,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			locationStore := NewMockLocationStore()
			driverRepo := NewMockDriverRepository()
			driverRepo.AddDriver(&domain.Driver{
				ID:     "driver-1",
				Status: domain.DriverStatusOffline,
			})

			driverService := service.NewDriverService(locationStore, nil, driverRepo)

			req := service.UpdateLocationRequest{
				DriverID: "driver-1",
				Lat:      tc.lat,
				Lng:      tc.lng,
			}

			err := driverService.UpdateLocation(context.Background(), req)
			if tc.wantErr && err == nil {
				t.Error("expected error for invalid coordinates, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestDriverLocationUpdate_MissingDriverID_Rejected(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	driverRepo := NewMockDriverRepository()
	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	req := service.UpdateLocationRequest{
		DriverID: "", // Missing driver ID
		Lat:      12.9716,
		Lng:      77.5946,
	}

	err := driverService.UpdateLocation(context.Background(), req)
	if err == nil {
		t.Error("expected error for missing driver ID, got nil")
	}
}

func TestDriverLocationUpdate_HighFrequencyUpdates_NoError(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	driverRepo := NewMockDriverRepository()
	driverRepo.AddDriver(&domain.Driver{
		ID:     "driver-1",
		Status: domain.DriverStatusOffline,
	})

	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	// Simulate high-frequency updates (100 updates)
	for i := 0; i < 100; i++ {
		req := service.UpdateLocationRequest{
			DriverID: "driver-1",
			Lat:      12.9716 + float64(i)*0.0001,
			Lng:      77.5946 + float64(i)*0.0001,
		}

		err := driverService.UpdateLocation(context.Background(), req)
		if err != nil {
			t.Fatalf("update %d failed: %v", i, err)
		}
	}

	// Verify all updates were processed
	if locationStore.UpdateLocationCallCount != 100 {
		t.Errorf("expected 100 updates, got %d", locationStore.UpdateLocationCallCount)
	}
}

func TestDriverLocationUpdate_SetsDriverOnline(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	driverRepo := NewMockDriverRepository()

	// Add driver in OFFLINE state
	driverRepo.AddDriver(&domain.Driver{
		ID:     "driver-1",
		Status: domain.DriverStatusOffline,
	})

	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	req := service.UpdateLocationRequest{
		DriverID: "driver-1",
		Lat:      12.9716,
		Lng:      77.5946,
	}

	err := driverService.UpdateLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify driver status was updated to ONLINE
	driver := driverRepo.GetDriver("driver-1")
	if driver == nil {
		t.Fatal("driver not found")
	}

	if driver.Status != domain.DriverStatusOnline {
		t.Errorf("expected driver status %s, got %s", domain.DriverStatusOnline, driver.Status)
	}
}

func TestDriverLocationUpdate_RedisError_PropagatesError(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	locationStore.UpdateLocationError = ErrMockTimeout

	driverRepo := NewMockDriverRepository()
	driverRepo.AddDriver(&domain.Driver{
		ID:     "driver-1",
		Status: domain.DriverStatusOffline,
	})

	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	req := service.UpdateLocationRequest{
		DriverID: "driver-1",
		Lat:      12.9716,
		Lng:      77.5946,
	}

	err := driverService.UpdateLocation(context.Background(), req)
	if err == nil {
		t.Error("expected error when Redis fails, got nil")
	}
}

func TestDriverLocationUpdate_UnknownDriver_StillUpdatesRedis(t *testing.T) {
	t.Parallel()

	locationStore := NewMockLocationStore()
	driverRepo := NewMockDriverRepository()
	// Note: No driver added to repo

	driverService := service.NewDriverService(locationStore, nil, driverRepo)

	req := service.UpdateLocationRequest{
		DriverID: "unknown-driver",
		Lat:      12.9716,
		Lng:      77.5946,
	}

	// Should not error - location update is allowed for unknown drivers
	// (per README: "Ignore error if driver not found")
	err := driverService.UpdateLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Redis should still be updated
	if !locationStore.HasLocation("unknown-driver") {
		t.Error("expected location to be stored even for unknown driver")
	}
}
