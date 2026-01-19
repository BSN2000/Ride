package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"ride/internal/domain"
	"ride/internal/redis"
)

func TestMatchingLogic_FiltersOfflineDrivers(t *testing.T) {
	ctx := context.Background()

	// Setup mocks.
	driverRepo := NewMockDriverRepository()
	locationStore := NewMockLocationStore()
	_ = NewMockLockStore() // Not used in this test.

	// Add an offline driver and an online driver.
	offlineDriver := &domain.Driver{
		ID:     "driver-offline",
		Status: domain.DriverStatusOffline,
		Tier:   domain.DriverTierBasic,
	}
	onlineDriver := &domain.Driver{
		ID:     "driver-online",
		Status: domain.DriverStatusOnline,
		Tier:   domain.DriverTierBasic,
	}
	driverRepo.AddDriver(offlineDriver)
	driverRepo.AddDriver(onlineDriver)

	// Add locations (offline first, then online).
	locationStore.SetLocations([]redis.DriverLocation{
		{DriverID: "driver-offline", Lat: 12.0, Lng: 77.0},
		{DriverID: "driver-online", Lat: 12.1, Lng: 77.1},
	})

	// Simulate matching logic: iterate through nearby drivers and filter by status.
	nearbyDrivers, err := locationStore.FindNearbyDrivers(ctx, 12.0, 77.0, 5.0)
	if err != nil {
		t.Fatalf("failed to find nearby drivers: %v", err)
	}

	var matchedDriver *domain.Driver
	for _, loc := range nearbyDrivers {
		driver, err := driverRepo.GetByID(ctx, loc.DriverID)
		if err != nil {
			continue
		}
		if driver.Status == domain.DriverStatusOnline {
			matchedDriver = driver
			break
		}
	}

	// Should match the online driver, not the offline one.
	if matchedDriver == nil {
		t.Fatal("expected to match a driver")
	}
	if matchedDriver.ID != "driver-online" {
		t.Errorf("expected driver-online, got %s", matchedDriver.ID)
	}
}

func TestMatchingLogic_FiltersByTier(t *testing.T) {
	ctx := context.Background()

	// Setup mocks.
	driverRepo := NewMockDriverRepository()
	locationStore := NewMockLocationStore()

	// Add drivers with different tiers.
	basicDriver := &domain.Driver{
		ID:     "driver-basic",
		Status: domain.DriverStatusOnline,
		Tier:   domain.DriverTierBasic,
	}
	premiumDriver := &domain.Driver{
		ID:     "driver-premium",
		Status: domain.DriverStatusOnline,
		Tier:   domain.DriverTierPremium,
	}
	driverRepo.AddDriver(basicDriver)
	driverRepo.AddDriver(premiumDriver)

	// Add locations (basic first).
	locationStore.SetLocations([]redis.DriverLocation{
		{DriverID: "driver-basic", Lat: 12.0, Lng: 77.0},
		{DriverID: "driver-premium", Lat: 12.1, Lng: 77.1},
	})

	// Filter for premium tier only.
	requestedTier := domain.DriverTierPremium

	nearbyDrivers, _ := locationStore.FindNearbyDrivers(ctx, 12.0, 77.0, 5.0)

	var matchedDriver *domain.Driver
	for _, loc := range nearbyDrivers {
		driver, err := driverRepo.GetByID(ctx, loc.DriverID)
		if err != nil {
			continue
		}
		if driver.Status == domain.DriverStatusOnline && driver.Tier == requestedTier {
			matchedDriver = driver
			break
		}
	}

	if matchedDriver == nil {
		t.Fatal("expected to match a premium driver")
	}
	if matchedDriver.Tier != domain.DriverTierPremium {
		t.Errorf("expected premium tier, got %s", matchedDriver.Tier)
	}
}

func TestMatchingLogic_NoDriversAvailable(t *testing.T) {
	ctx := context.Background()

	locationStore := NewMockLocationStore()
	// No drivers in location store.

	nearbyDrivers, err := locationStore.FindNearbyDrivers(ctx, 12.0, 77.0, 5.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(nearbyDrivers) != 0 {
		t.Errorf("expected no drivers, got %d", len(nearbyDrivers))
	}
}

func TestDriverLocking_AcquireLock(t *testing.T) {
	ctx := context.Background()
	lockStore := NewMockLockStore()

	driverID := "driver-1"

	// First lock should succeed.
	acquired, err := lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !acquired {
		t.Error("expected to acquire lock")
	}

	// Verify driver is locked.
	if !lockStore.IsLocked(driverID) {
		t.Error("expected driver to be locked")
	}
}

func TestDriverLocking_CannotAcquireLockedDriver(t *testing.T) {
	ctx := context.Background()
	lockStore := NewMockLockStore()

	driverID := "driver-1"

	// First lock.
	acquired1, _ := lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)
	if !acquired1 {
		t.Fatal("expected first lock to succeed")
	}

	// Second lock should fail.
	acquired2, err := lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acquired2 {
		t.Error("expected second lock to fail")
	}
}

func TestDriverLocking_ReleaseLock(t *testing.T) {
	ctx := context.Background()
	lockStore := NewMockLockStore()

	driverID := "driver-1"

	// Acquire lock.
	lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)

	// Release lock.
	err := lockStore.ReleaseDriverLock(ctx, driverID)
	if err != nil {
		t.Fatalf("unexpected error releasing lock: %v", err)
	}

	// Should be able to acquire again.
	acquired, _ := lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)
	if !acquired {
		t.Error("expected to acquire lock after release")
	}
}

func TestDriverLocking_ConcurrentLockAttempts(t *testing.T) {
	ctx := context.Background()
	lockStore := NewMockLockStore()

	driverID := "driver-1"
	numGoroutines := 10
	successCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			acquired, err := lockStore.AcquireDriverLock(ctx, driverID, 10*time.Second)
			if err != nil {
				return
			}
			if acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Only one goroutine should have acquired the lock.
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful lock, got %d", successCount)
	}
}

func TestMatchingLogic_SkipsLockedDrivers(t *testing.T) {
	ctx := context.Background()

	driverRepo := NewMockDriverRepository()
	locationStore := NewMockLocationStore()
	lockStore := NewMockLockStore()

	// Add two online drivers.
	driver1 := &domain.Driver{ID: "driver-1", Status: domain.DriverStatusOnline, Tier: domain.DriverTierBasic}
	driver2 := &domain.Driver{ID: "driver-2", Status: domain.DriverStatusOnline, Tier: domain.DriverTierBasic}
	driverRepo.AddDriver(driver1)
	driverRepo.AddDriver(driver2)

	locationStore.SetLocations([]redis.DriverLocation{
		{DriverID: "driver-1", Lat: 12.0, Lng: 77.0},
		{DriverID: "driver-2", Lat: 12.1, Lng: 77.1},
	})

	// Lock the first driver.
	lockStore.AcquireDriverLock(ctx, "driver-1", 10*time.Second)

	// Simulate matching: should skip locked driver and match second.
	nearbyDrivers, _ := locationStore.FindNearbyDrivers(ctx, 12.0, 77.0, 5.0)

	var matchedDriver *domain.Driver
	for _, loc := range nearbyDrivers {
		driver, err := driverRepo.GetByID(ctx, loc.DriverID)
		if err != nil {
			continue
		}
		if driver.Status != domain.DriverStatusOnline {
			continue
		}

		// Try to acquire lock.
		acquired, _ := lockStore.AcquireDriverLock(ctx, driver.ID, 10*time.Second)
		if !acquired {
			continue
		}

		matchedDriver = driver
		break
	}

	if matchedDriver == nil {
		t.Fatal("expected to match a driver")
	}
	if matchedDriver.ID != "driver-2" {
		t.Errorf("expected driver-2 (first was locked), got %s", matchedDriver.ID)
	}
}

func TestMatchingLogic_MatchesClosestDriver(t *testing.T) {
	ctx := context.Background()

	driverRepo := NewMockDriverRepository()
	locationStore := NewMockLocationStore()
	lockStore := NewMockLockStore()

	// Add drivers.
	driver1 := &domain.Driver{ID: "driver-far", Status: domain.DriverStatusOnline, Tier: domain.DriverTierBasic}
	driver2 := &domain.Driver{ID: "driver-close", Status: domain.DriverStatusOnline, Tier: domain.DriverTierBasic}
	driverRepo.AddDriver(driver1)
	driverRepo.AddDriver(driver2)

	// Locations returned in order (closest first - simulating Redis GEORADIUS sort).
	locationStore.SetLocations([]redis.DriverLocation{
		{DriverID: "driver-close", Lat: 12.0, Lng: 77.0}, // Closest.
		{DriverID: "driver-far", Lat: 12.5, Lng: 77.5},   // Farther.
	})

	nearbyDrivers, _ := locationStore.FindNearbyDrivers(ctx, 12.0, 77.0, 10.0)

	var matchedDriver *domain.Driver
	for _, loc := range nearbyDrivers {
		driver, _ := driverRepo.GetByID(ctx, loc.DriverID)
		if driver.Status == domain.DriverStatusOnline {
			acquired, _ := lockStore.AcquireDriverLock(ctx, driver.ID, 10*time.Second)
			if acquired {
				matchedDriver = driver
				break
			}
		}
	}

	if matchedDriver == nil {
		t.Fatal("expected to match a driver")
	}
	if matchedDriver.ID != "driver-close" {
		t.Errorf("expected closest driver (driver-close), got %s", matchedDriver.ID)
	}
}
