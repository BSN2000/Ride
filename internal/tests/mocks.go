package tests

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"ride/internal/domain"
	"ride/internal/redis"
	"ride/internal/repository"
)

// ──────────────────────────────────────────────
// MOCK DRIVER REPOSITORY
// ──────────────────────────────────────────────

// MockDriverRepository is a mock implementation of DriverRepository.
type MockDriverRepository struct {
	mu      sync.RWMutex
	drivers map[string]*domain.Driver

	// Counters for verification
	CreateCallCount       int32
	UpdateStatusCallCount int32

	// Error injection
	CreateError       error
	UpdateStatusError error
}

// NewMockDriverRepository creates a new mock driver repository.
func NewMockDriverRepository() *MockDriverRepository {
	return &MockDriverRepository{
		drivers: make(map[string]*domain.Driver),
	}
}

// AddDriver adds a driver to the mock repository.
func (m *MockDriverRepository) AddDriver(driver *domain.Driver) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[driver.ID] = driver
}

func (m *MockDriverRepository) Create(ctx context.Context, driver *domain.Driver) error {
	atomic.AddInt32(&m.CreateCallCount, 1)
	if m.CreateError != nil {
		return m.CreateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[driver.ID] = driver
	return nil
}

func (m *MockDriverRepository) GetByID(ctx context.Context, id string) (*domain.Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	driver, ok := m.drivers[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	// Return a copy to avoid mutation issues.
	copy := *driver
	return &copy, nil
}

func (m *MockDriverRepository) GetByPhone(ctx context.Context, phone string) (*domain.Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.drivers {
		if d.Phone == phone {
			copy := *d
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *MockDriverRepository) GetAll(ctx context.Context) ([]*domain.Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Driver, 0, len(m.drivers))
	for _, d := range m.drivers {
		copy := *d
		result = append(result, &copy)
	}
	return result, nil
}

func (m *MockDriverRepository) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
	atomic.AddInt32(&m.UpdateStatusCallCount, 1)
	if m.UpdateStatusError != nil {
		return m.UpdateStatusError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	driver, ok := m.drivers[id]
	if !ok {
		return repository.ErrNotFound
	}
	driver.Status = status
	return nil
}

// GetDriver returns driver for test assertions.
func (m *MockDriverRepository) GetDriver(id string) *domain.Driver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.drivers[id]
}

// ──────────────────────────────────────────────
// MOCK RIDE REPOSITORY
// ──────────────────────────────────────────────

// MockRideRepository is a mock implementation of RideRepository.
type MockRideRepository struct {
	mu    sync.RWMutex
	rides map[string]*domain.Ride

	// Counters for verification
	CreateCallCount int32
	UpdateCallCount int32

	// Error injection
	CreateError error
	UpdateError error
}

// NewMockRideRepository creates a new mock ride repository.
func NewMockRideRepository() *MockRideRepository {
	return &MockRideRepository{
		rides: make(map[string]*domain.Ride),
	}
}

// AddRide adds a ride to the mock repository.
func (m *MockRideRepository) AddRide(ride *domain.Ride) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rides[ride.ID] = ride
}

func (m *MockRideRepository) Create(ctx context.Context, ride *domain.Ride) error {
	atomic.AddInt32(&m.CreateCallCount, 1)
	if m.CreateError != nil {
		return m.CreateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rides[ride.ID] = ride
	return nil
}

func (m *MockRideRepository) GetByID(ctx context.Context, id string) (*domain.Ride, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ride, ok := m.rides[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	// Return a copy to avoid mutation issues.
	copy := *ride
	return &copy, nil
}

func (m *MockRideRepository) GetAll(ctx context.Context) ([]*domain.Ride, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Ride, 0, len(m.rides))
	for _, r := range m.rides {
		copy := *r
		result = append(result, &copy)
	}
	return result, nil
}

func (m *MockRideRepository) Update(ctx context.Context, ride *domain.Ride) error {
	atomic.AddInt32(&m.UpdateCallCount, 1)
	if m.UpdateError != nil {
		return m.UpdateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rides[ride.ID]; !ok {
		return repository.ErrNotFound
	}
	m.rides[ride.ID] = ride
	return nil
}

// GetRide returns the ride by ID (for test assertions).
func (m *MockRideRepository) GetRide(id string) *domain.Ride {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rides[id]
}

// GetAllRides returns all rides for assertions.
func (m *MockRideRepository) GetAllRides() []*domain.Ride {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Ride, 0, len(m.rides))
	for _, r := range m.rides {
		result = append(result, r)
	}
	return result
}

// CountRides returns the number of rides.
func (m *MockRideRepository) CountRides() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rides)
}

// ──────────────────────────────────────────────
// MOCK TRIP REPOSITORY
// ──────────────────────────────────────────────

// MockTripRepository is a mock implementation of TripRepository.
type MockTripRepository struct {
	mu    sync.RWMutex
	trips map[string]*domain.Trip

	// Counters
	CreateCallCount int32
	UpdateCallCount int32

	// Error injection
	CreateError error
	UpdateError error
}

// NewMockTripRepository creates a new mock trip repository.
func NewMockTripRepository() *MockTripRepository {
	return &MockTripRepository{
		trips: make(map[string]*domain.Trip),
	}
}

func (m *MockTripRepository) Create(ctx context.Context, trip *domain.Trip) error {
	atomic.AddInt32(&m.CreateCallCount, 1)
	if m.CreateError != nil {
		return m.CreateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trips[trip.ID] = trip
	return nil
}

func (m *MockTripRepository) GetByID(ctx context.Context, id string) (*domain.Trip, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	trip, ok := m.trips[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := *trip
	return &copy, nil
}

func (m *MockTripRepository) GetActiveByDriverID(ctx context.Context, driverID string) (*domain.Trip, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.trips {
		if t.DriverID == driverID && t.Status != domain.TripStatusEnded {
			copy := *t
			return &copy, nil
		}
	}
	return nil, nil // No active trip
}

func (m *MockTripRepository) Update(ctx context.Context, trip *domain.Trip) error {
	atomic.AddInt32(&m.UpdateCallCount, 1)
	if m.UpdateError != nil {
		return m.UpdateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trips[trip.ID] = trip
	return nil
}

// GetTrip returns trip for assertions.
func (m *MockTripRepository) GetTrip(id string) *domain.Trip {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.trips[id]
}

// CountTrips returns the number of trips.
func (m *MockTripRepository) CountTrips() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.trips)
}

// CountActiveTripsForDriver counts active trips for a driver.
func (m *MockTripRepository) CountActiveTripsForDriver(driverID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, t := range m.trips {
		if t.DriverID == driverID && t.Status != domain.TripStatusEnded {
			count++
		}
	}
	return count
}

// ──────────────────────────────────────────────
// MOCK PAYMENT REPOSITORY
// ──────────────────────────────────────────────

// MockPaymentRepository is a mock implementation of PaymentRepository.
type MockPaymentRepository struct {
	mu       sync.RWMutex
	payments map[string]*domain.Payment

	// Counters
	CreateCallCount int32

	// Error injection
	CreateError error
}

// NewMockPaymentRepository creates a new mock payment repository.
func NewMockPaymentRepository() *MockPaymentRepository {
	return &MockPaymentRepository{
		payments: make(map[string]*domain.Payment),
	}
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	atomic.AddInt32(&m.CreateCallCount, 1)
	if m.CreateError != nil {
		return m.CreateError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.payments[payment.ID] = payment
	return nil
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	payment, ok := m.payments[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := *payment
	return &copy, nil
}

func (m *MockPaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.payments {
		if p.IdempotencyKey == key {
			copy := *p
			return &copy, nil
		}
	}
	return nil, nil // Not found, but not an error for idempotency check
}

func (m *MockPaymentRepository) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	payment, ok := m.payments[id]
	if !ok {
		return repository.ErrNotFound
	}
	payment.Status = status
	return nil
}

// CountPayments returns the number of payments.
func (m *MockPaymentRepository) CountPayments() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.payments)
}

// GetPaymentByTripID returns payment for a trip.
func (m *MockPaymentRepository) GetPaymentByTripID(tripID string) *domain.Payment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.payments {
		if p.TripID == tripID {
			return p
		}
	}
	return nil
}

// ──────────────────────────────────────────────
// MOCK LOCATION STORE
// ──────────────────────────────────────────────

// MockLocationStore is a mock implementation of LocationStore.
type MockLocationStore struct {
	mu        sync.RWMutex
	locations []redis.DriverLocation

	// Counters
	UpdateLocationCallCount int32

	// Error injection
	UpdateLocationError    error
	FindNearbyDriversError error
}

// NewMockLocationStore creates a new mock location store.
func NewMockLocationStore() *MockLocationStore {
	return &MockLocationStore{
		locations: make([]redis.DriverLocation, 0),
	}
}

// AddDriverLocation adds a driver location to the mock store.
func (m *MockLocationStore) AddDriverLocation(loc redis.DriverLocation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locations = append(m.locations, loc)
}

// SetLocations sets all locations (for test setup).
func (m *MockLocationStore) SetLocations(locations []redis.DriverLocation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locations = locations
}

func (m *MockLocationStore) UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error {
	atomic.AddInt32(&m.UpdateLocationCallCount, 1)
	if m.UpdateLocationError != nil {
		return m.UpdateLocationError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Update existing or add new.
	for i, loc := range m.locations {
		if loc.DriverID == driverID {
			m.locations[i].Lat = lat
			m.locations[i].Lng = lng
			return nil
		}
	}
	m.locations = append(m.locations, redis.DriverLocation{
		DriverID: driverID,
		Lat:      lat,
		Lng:      lng,
	})
	return nil
}

func (m *MockLocationStore) FindNearbyDrivers(ctx context.Context, lat, lng, radiusKm float64) ([]redis.DriverLocation, error) {
	if m.FindNearbyDriversError != nil {
		return nil, m.FindNearbyDriversError
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return all locations (mock doesn't do real geo filtering).
	result := make([]redis.DriverLocation, len(m.locations))
	copy(result, m.locations)
	return result, nil
}

func (m *MockLocationStore) RemoveLocation(ctx context.Context, driverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, loc := range m.locations {
		if loc.DriverID == driverID {
			m.locations = append(m.locations[:i], m.locations[i+1:]...)
			return nil
		}
	}
	return nil
}

// HasLocation checks if a driver location exists.
func (m *MockLocationStore) HasLocation(driverID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, loc := range m.locations {
		if loc.DriverID == driverID {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// MOCK LOCK STORE
// ──────────────────────────────────────────────

// MockLockStore is a mock implementation of LockStore.
type MockLockStore struct {
	mu    sync.Mutex
	locks map[string]time.Time

	// Counters
	AcquireCallCount int32
	ReleaseCallCount int32

	// Error injection
	AcquireError error

	// Force lock failure
	ForceAcquireFailure bool
}

// NewMockLockStore creates a new mock lock store.
func NewMockLockStore() *MockLockStore {
	return &MockLockStore{
		locks: make(map[string]time.Time),
	}
}

func (m *MockLockStore) AcquireDriverLock(ctx context.Context, driverID string, ttl time.Duration) (bool, error) {
	atomic.AddInt32(&m.AcquireCallCount, 1)
	if m.AcquireError != nil {
		return false, m.AcquireError
	}
	if m.ForceAcquireFailure {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := "lock:driver:" + driverID
	if expiry, exists := m.locks[key]; exists {
		if time.Now().Before(expiry) {
			return false, nil // Lock still held.
		}
	}

	m.locks[key] = time.Now().Add(ttl)
	return true, nil
}

func (m *MockLockStore) ReleaseDriverLock(ctx context.Context, driverID string) error {
	atomic.AddInt32(&m.ReleaseCallCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.locks, "lock:driver:"+driverID)
	return nil
}

// IsLocked checks if a driver is locked (for test assertions).
func (m *MockLockStore) IsLocked(driverID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	expiry, exists := m.locks["lock:driver:"+driverID]
	return exists && time.Now().Before(expiry)
}

// ClearLocks clears all locks (for test cleanup).
func (m *MockLockStore) ClearLocks() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locks = make(map[string]time.Time)
}

// ──────────────────────────────────────────────
// MOCK PSP (Payment Service Provider)
// ──────────────────────────────────────────────

// MockPSP is a mock payment service provider.
type MockPSP struct {
	mu sync.Mutex

	// Control behavior
	ShouldFail bool
	FailError  error

	// Counters
	ChargeCallCount int32
}

// NewMockPSP creates a new mock PSP.
func NewMockPSP() *MockPSP {
	return &MockPSP{}
}

func (m *MockPSP) Charge(ctx context.Context, amount float64) (bool, error) {
	atomic.AddInt32(&m.ChargeCallCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailError != nil {
		return false, m.FailError
	}
	if m.ShouldFail {
		return false, nil
	}
	return true, nil
}

// SetFailure configures the PSP to fail.
func (m *MockPSP) SetFailure(shouldFail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldFail = shouldFail
	m.FailError = err
}

// ──────────────────────────────────────────────
// HELPER ERRORS
// ──────────────────────────────────────────────

var (
	ErrMockDBConstraint = errors.New("mock: unique constraint violation")
	ErrMockTimeout      = errors.New("mock: operation timeout")
)
