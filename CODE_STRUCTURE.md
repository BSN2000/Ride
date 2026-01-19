# Ride-Hailing System - Complete Code Documentation

> **For Interview Explanation**
> 
> This document provides a comprehensive explanation of every component in the codebase, designed for technical interviews.

---

## Table of Contents

1. [Project Structure Overview](#1-project-structure-overview)
2. [Domain Layer](#2-domain-layer)
3. [Repository Layer](#3-repository-layer)
4. [Redis Layer](#4-redis-layer)
5. [Service Layer](#5-service-layer)
6. [Handler Layer](#6-handler-layer)
7. [Middleware Layer](#7-middleware-layer)
8. [Configuration & App Setup](#8-configuration--app-setup)
9. [Database Schema](#9-database-schema)
10. [Complete API Reference](#10-complete-api-reference)
11. [Data Flow Diagrams](#11-data-flow-diagrams)
12. [Concurrency & Safety Guarantees](#12-concurrency--safety-guarantees)
13. [Testing Strategy](#13-testing-strategy)

---

# 1. Project Structure Overview

```
ride-hailing-system/
│
├── cmd/
│   └── server/
│       └── main.go                 ← Application entry point
│
├── internal/                       ← Private application code (Go convention)
│   │
│   ├── app/                        ← Application wiring & infrastructure
│   │   ├── database.go             ← PostgreSQL connection setup
│   │   ├── redis.go                ← Redis client setup
│   │   └── router.go               ← Gin router with all routes
│   │
│   ├── config/
│   │   └── config.go               ← Environment variable loading
│   │
│   ├── domain/                     ← Core business entities (ZERO dependencies)
│   │   ├── user.go                 ← User (rider) entity
│   │   ├── driver.go               ← Driver entity with status/tier
│   │   ├── ride.go                 ← Ride request entity
│   │   ├── trip.go                 ← Active/completed trip entity
│   │   └── payment.go              ← Payment transaction entity
│   │
│   ├── handler/                    ← HTTP request handlers
│   │   ├── user.go                 ← User registration endpoints
│   │   ├── driver.go               ← Driver location/accept endpoints
│   │   ├── ride.go                 ← Ride creation/status endpoints
│   │   ├── trip.go                 ← Trip end endpoint
│   │   └── response.go             ← Response helpers & error mapping
│   │
│   ├── middleware/                 ← HTTP middleware
│   │   ├── cors.go                 ← Cross-Origin Resource Sharing
│   │   ├── idempotency.go          ← Duplicate request prevention
│   │   └── newrelic.go             ← APM monitoring (custom wrapper)
│   │
│   ├── redis/                      ← Redis operations
│   │   ├── interfaces.go           ← LocationStore & LockStore interfaces
│   │   ├── location.go             ← GEOADD/GEORADIUS for driver locations
│   │   └── lock.go                 ← Distributed locking (SET NX EX)
│   │
│   ├── repository/                 ← Data access layer
│   │   ├── user.go                 ← UserRepository interface
│   │   ├── driver.go               ← DriverRepository interface
│   │   ├── ride.go                 ← RideRepository interface
│   │   ├── trip.go                 ← TripRepository interface
│   │   ├── payment.go              ← PaymentRepository interface
│   │   ├── errors.go               ← ErrNotFound sentinel error
│   │   └── postgres/               ← PostgreSQL implementations
│   │       ├── db.go               ← Querier interface for transactions
│   │       ├── user.go
│   │       ├── driver.go
│   │       ├── ride.go
│   │       ├── trip.go
│   │       └── payment.go
│   │
│   ├── service/                    ← Business logic layer
│   │   ├── ride.go                 ← RideService (create, get status)
│   │   ├── driver.go               ← DriverService (update location)
│   │   ├── matching.go             ← MatchingService (find & assign driver)
│   │   ├── trip.go                 ← TripService (start, end trip)
│   │   ├── payment.go              ← PaymentService (process payment)
│   │   ├── surge.go                ← SurgeService (dynamic pricing)
│   │   └── errors.go               ← Business domain errors
│   │
│   └── tests/                      ← Unit tests
│       ├── mocks.go                ← Mock implementations
│       ├── ride_test.go
│       ├── ride_creation_test.go
│       ├── driver_location_test.go
│       ├── matching_test.go
│       ├── trip_lifecycle_test.go
│       └── concurrency_test.go
│
├── scripts/
│   └── schema.sql                  ← Database schema
│
├── frontend/                       ← Static UI
│   ├── index.html
│   ├── styles.css
│   └── app.js
│
├── docker-compose.yml              ← Container orchestration
├── Dockerfile                      ← Go app container
├── go.mod / go.sum                 ← Dependencies
├── LLD.md                          ← Low-Level Design document
├── postman_collection.json         ← API test collection
└── read.md                         ← Original requirements
```

## Layer Architecture (Clean Architecture)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LAYER DIAGRAM                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                         HANDLER LAYER                                │   │
│   │  • HTTP parsing     • Request validation    • Response formatting   │   │
│   │  driver.go, ride.go, trip.go, user.go, response.go                  │   │
│   └────────────────────────────────┬────────────────────────────────────┘   │
│                                    │ Calls                                   │
│   ┌────────────────────────────────▼────────────────────────────────────┐   │
│   │                         SERVICE LAYER                                │   │
│   │  • Business logic   • Orchestration    • Validation rules            │   │
│   │  ride.go, driver.go, trip.go, matching.go, payment.go, surge.go     │   │
│   └─────────────────┬──────────────────────────────┬────────────────────┘   │
│                     │ Calls                        │ Calls                   │
│   ┌─────────────────▼─────────────┐  ┌─────────────▼────────────────────┐   │
│   │      REPOSITORY LAYER         │  │         REDIS LAYER              │   │
│   │  • Database operations        │  │  • Location (GEO)                │   │
│   │  • CRUD for entities          │  │  • Distributed Locks             │   │
│   │  postgres/driver.go, etc.     │  │  location.go, lock.go            │   │
│   └─────────────────┬─────────────┘  └─────────────┬────────────────────┘   │
│                     │                               │                        │
│   ┌─────────────────▼───────────────────────────────▼────────────────────┐  │
│   │                    EXTERNAL SYSTEMS                                   │  │
│   │              PostgreSQL                    Redis                      │  │
│   └──────────────────────────────────────────────────────────────────────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

# 2. Domain Layer

> **Location:** `internal/domain/`
> 
> **Principle:** Zero external dependencies. Pure Go structs representing business entities.

---

## 2.1 User Entity (`user.go`)

```go
package domain

import "time"

// User represents a rider in the system.
type User struct {
    ID        string
    Name      string
    Phone     string
    CreatedAt time.Time
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, primary key |
| `Name` | `string` | Display name |
| `Phone` | `string` | Unique identifier for registration |
| `CreatedAt` | `time.Time` | Registration timestamp |

### Business Rules:
- Phone must be unique (prevents duplicate registration)
- No authentication in this demo

---

## 2.2 Driver Entity (`driver.go`)

```go
package domain

// DriverStatus represents the current availability of a driver.
type DriverStatus string

const (
    DriverStatusOnline  DriverStatus = "ONLINE"   // Available for rides
    DriverStatusOffline DriverStatus = "OFFLINE"  // Not accepting rides
    DriverStatusOnTrip  DriverStatus = "ON_TRIP"  // Currently on a trip
)

// DriverTier represents the service level of a driver.
type DriverTier string

const (
    DriverTierBasic   DriverTier = "BASIC"    // Standard vehicles
    DriverTierPremium DriverTier = "PREMIUM"  // Luxury vehicles
)

// Driver represents a driver in the system.
type Driver struct {
    ID     string
    Name   string
    Phone  string
    Status DriverStatus
    Tier   DriverTier
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, primary key |
| `Name` | `string` | Display name |
| `Phone` | `string` | Unique, for registration |
| `Status` | `DriverStatus` | ONLINE / OFFLINE / ON_TRIP |
| `Tier` | `DriverTier` | BASIC / PREMIUM |

### State Transitions:

```
            ┌─────────────────────────────────────────┐
            │           DRIVER STATUS FSM             │
            ├─────────────────────────────────────────┤
            │                                         │
            │    ┌──────────┐                         │
            │    │ OFFLINE  │ ← Initial state         │
            │    └────┬─────┘                         │
            │         │ UpdateLocation()              │
            │         ▼                               │
            │    ┌──────────┐     AcceptRide()        │
            │    │  ONLINE  │ ──────────────────┐     │
            │    └────┬─────┘                   │     │
            │         │                         ▼     │
            │         │                  ┌──────────┐ │
            │         │                  │ ON_TRIP  │ │
            │         │                  └────┬─────┘ │
            │         │                       │       │
            │         │        EndTrip()      │       │
            │         ◀───────────────────────┘       │
            │                                         │
            └─────────────────────────────────────────┘
```

### Business Rules:
- Only ONLINE drivers can accept rides
- Driver goes ON_TRIP when assigned to ride
- Driver returns to ONLINE when trip ends
- Only one active trip per driver (DB constraint)

---

## 2.3 Ride Entity (`ride.go`)

```go
package domain

// RideStatus represents the current status of a ride.
type RideStatus string

const (
    RideStatusRequested RideStatus = "REQUESTED"  // Waiting for driver
    RideStatusAssigned  RideStatus = "ASSIGNED"   // Driver matched
    RideStatusCancelled RideStatus = "CANCELLED"  // User cancelled
)

// Ride represents a ride request in the system.
type Ride struct {
    ID               string
    RiderID          string
    PickupLat        float64
    PickupLng        float64
    DestinationLat   float64
    DestinationLng   float64
    Status           RideStatus
    AssignedDriverID string
    SurgeMultiplier  float64  // 1.0 = no surge, 2.0 = 2x pricing
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, primary key |
| `RiderID` | `string` | User who requested the ride |
| `PickupLat/Lng` | `float64` | GPS coordinates for pickup |
| `DestinationLat/Lng` | `float64` | GPS coordinates for destination |
| `Status` | `RideStatus` | REQUESTED / ASSIGNED / CANCELLED |
| `AssignedDriverID` | `string` | NULL until driver assigned |
| `SurgeMultiplier` | `float64` | Dynamic pricing multiplier |

### State Transitions:

```
    ┌─────────────┐   MatchingService   ┌────────────┐
    │  REQUESTED  │ ──────────────────▶ │  ASSIGNED  │
    └──────┬──────┘   (driver found)    └────────────┘
           │
           │ (no driver / timeout)
           ▼
    ┌─────────────┐
    │  CANCELLED  │
    └─────────────┘
```

---

## 2.4 Trip Entity (`trip.go`)

```go
package domain

import "time"

// TripStatus represents the current status of a trip.
type TripStatus string

const (
    TripStatusStarted TripStatus = "STARTED"  // Trip in progress
    TripStatusPaused  TripStatus = "PAUSED"   // Temporarily stopped
    TripStatusEnded   TripStatus = "ENDED"    // Trip completed
)

// Trip represents an active or completed trip in the system.
type Trip struct {
    ID        string
    RideID    string
    DriverID  string
    Status    TripStatus
    Fare      float64
    StartedAt time.Time
    EndedAt   time.Time
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, primary key |
| `RideID` | `string` | Associated ride request |
| `DriverID` | `string` | Driver performing trip |
| `Status` | `TripStatus` | STARTED / PAUSED / ENDED |
| `Fare` | `float64` | Calculated at trip end |
| `StartedAt` | `time.Time` | Trip start timestamp |
| `EndedAt` | `time.Time` | Trip end timestamp |

### Fare Calculation:

```go
// Formula: baseFare + (minutes × perMinuteRate) × surgeMultiplier
fare := (2.0 + (duration.Minutes() × 0.5)) × ride.SurgeMultiplier
if fare < 5.0 {
    fare = 5.0  // Minimum fare
}
```

---

## 2.5 Payment Entity (`payment.go`)

```go
package domain

// PaymentStatus represents the current status of a payment.
type PaymentStatus string

const (
    PaymentStatusPending PaymentStatus = "PENDING"  // Processing
    PaymentStatusSuccess PaymentStatus = "SUCCESS"  // Completed
    PaymentStatusFailed  PaymentStatus = "FAILED"   // Failed
)

// Payment represents a payment for a trip.
type Payment struct {
    ID             string
    TripID         string
    Amount         float64
    Status         PaymentStatus
    IdempotencyKey string
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID, primary key |
| `TripID` | `string` | Associated trip |
| `Amount` | `float64` | Same as trip fare |
| `Status` | `PaymentStatus` | PENDING / SUCCESS / FAILED |
| `IdempotencyKey` | `string` | Format: `payment:{trip_id}` |

### Idempotency Pattern:

```
Request 1: POST /process-payment (TripID: abc)
  → Check: No payment with key "payment:abc"
  → Create: Payment(status=PENDING)
  → Call PSP
  → Update: Payment(status=SUCCESS)
  → Return: Payment

Request 2: POST /process-payment (TripID: abc)  [DUPLICATE]
  → Check: Found payment with key "payment:abc"
  → Return: Existing payment (no duplicate created)
```

---

# 3. Repository Layer

> **Location:** `internal/repository/`
> 
> **Principle:** Define interfaces (contracts), implementations in `postgres/` subfolder.

---

## 3.1 Architecture

```
repository/
├── Interfaces (contracts)          ← What operations exist
│   ├── user.go
│   ├── driver.go
│   ├── ride.go
│   ├── trip.go
│   └── payment.go
│
├── errors.go                       ← Shared error definitions
│
└── postgres/                       ← How operations are implemented
    ├── db.go                       ← Transaction support
    ├── user.go
    ├── driver.go
    ├── ride.go
    ├── trip.go
    └── payment.go
```

---

## 3.2 UserRepository

### Interface:

```go
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    GetByID(ctx context.Context, id string) (*domain.User, error)
    GetByPhone(ctx context.Context, phone string) (*domain.User, error)
    GetAll(ctx context.Context) ([]*domain.User, error)
}
```

### Methods Explained:

| Method | SQL Operation | Used By | Purpose |
|--------|---------------|---------|---------|
| `Create` | `INSERT INTO users` | UserHandler.Register | Create new rider account |
| `GetByID` | `SELECT WHERE id=$1` | - | Fetch user details |
| `GetByPhone` | `SELECT WHERE phone=$1` | Registration flow | Check if user exists |
| `GetAll` | `SELECT ORDER BY created_at` | Monitoring | List all users |

### PostgreSQL Implementation:

```go
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
    query := `INSERT INTO users (id, name, phone) VALUES ($1, $2, $3)`
    _, err := r.q.ExecContext(ctx, query, user.ID, user.Name, user.Phone)
    return err
}

func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
    query := `SELECT id, name, phone, created_at FROM users WHERE phone = $1`
    var user domain.User
    err := r.q.QueryRowContext(ctx, query, phone).Scan(
        &user.ID, &user.Name, &user.Phone, &user.CreatedAt,
    )
    if errors.Is(err, sql.ErrNoRows) {
        return nil, repository.ErrNotFound
    }
    return &user, err
}
```

---

## 3.3 DriverRepository

### Interface:

```go
type DriverRepository interface {
    Create(ctx context.Context, driver *domain.Driver) error
    GetByID(ctx context.Context, id string) (*domain.Driver, error)
    GetByPhone(ctx context.Context, phone string) (*domain.Driver, error)
    GetAll(ctx context.Context) ([]*domain.Driver, error)
    UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error
}
```

### Methods Explained:

| Method | SQL Operation | Used By | Purpose |
|--------|---------------|---------|---------|
| `Create` | `INSERT INTO drivers` | DriverHandler.Register | Register new driver |
| `GetByID` | `SELECT WHERE id=$1` | MatchingService | Verify driver status |
| `GetByPhone` | `SELECT WHERE phone=$1` | Registration | Prevent duplicates |
| `GetAll` | `SELECT ORDER BY created_at` | Driver portal | List all drivers |
| `UpdateStatus` | `UPDATE SET status=$1` | Location update, Assignment, Trip end | Change driver availability |

### Critical: UpdateStatus

```go
func (r *DriverRepository) UpdateStatus(ctx context.Context, id string, status domain.DriverStatus) error {
    query := `UPDATE drivers SET status = $1 WHERE id = $2`
    result, err := r.q.ExecContext(ctx, query, status, id)
    if err != nil {
        return err
    }
    
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return repository.ErrNotFound  // Driver doesn't exist
    }
    return nil
}
```

### Usage in Flow:

```
1. Location Update → UpdateStatus(driverID, ONLINE)
2. Ride Assigned   → UpdateStatus(driverID, ON_TRIP)
3. Trip Ended      → UpdateStatus(driverID, ONLINE)
```

---

## 3.4 RideRepository

### Interface:

```go
type RideRepository interface {
    Create(ctx context.Context, ride *domain.Ride) error
    GetByID(ctx context.Context, id string) (*domain.Ride, error)
    GetAll(ctx context.Context) ([]*domain.Ride, error)
    Update(ctx context.Context, ride *domain.Ride) error
}
```

### Methods Explained:

| Method | SQL Operation | Used By | Purpose |
|--------|---------------|---------|---------|
| `Create` | `INSERT INTO rides` | RideService.CreateRide | Create ride request |
| `GetByID` | `SELECT WHERE id=$1` | Polling, TripService | Get ride status |
| `GetAll` | `SELECT LIMIT 100` | Monitoring, SurgeService | List recent rides |
| `Update` | `UPDATE SET status, driver_id` | MatchingService | Assign driver to ride |

### Update with Transaction:

```go
// In MatchingService.assignDriver()
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

txRideRepo := postgres.NewRideRepositoryWithTx(tx)

ride.Status = domain.RideStatusAssigned
ride.AssignedDriverID = driver.ID

if err := txRideRepo.Update(ctx, ride); err != nil {
    return err  // Transaction rolls back
}

tx.Commit()
```

---

## 3.5 TripRepository

### Interface:

```go
type TripRepository interface {
    Create(ctx context.Context, trip *domain.Trip) error
    GetByID(ctx context.Context, id string) (*domain.Trip, error)
    Update(ctx context.Context, trip *domain.Trip) error
    GetActiveByDriverID(ctx context.Context, driverID string) (*domain.Trip, error)
}
```

### Methods Explained:

| Method | SQL Operation | Used By | Purpose |
|--------|---------------|---------|---------|
| `Create` | `INSERT INTO trips` | TripService.StartTrip | Start new trip |
| `GetByID` | `SELECT WHERE id=$1` | TripHandler | Get trip details |
| `Update` | `UPDATE SET status, fare, ended_at` | TripService.EndTrip | Complete trip |
| `GetActiveByDriverID` | `SELECT WHERE driver_id=$1 AND status!='ENDED'` | TripService | **Prevent double-booking** |

### Critical: GetActiveByDriverID

```go
func (r *TripRepository) GetActiveByDriverID(ctx context.Context, driverID string) (*domain.Trip, error) {
    query := `
        SELECT id, ride_id, driver_id, status, fare, started_at, ended_at 
        FROM trips 
        WHERE driver_id = $1 AND status != 'ENDED'
        LIMIT 1
    `
    var trip domain.Trip
    err := r.q.QueryRowContext(ctx, query, driverID).Scan(...)
    
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil  // No active trip - this is OK
    }
    return &trip, err
}
```

### Usage (prevent driver accepting two rides):

```go
// In TripService.StartTrip()
existingTrip, _ := tripRepo.GetActiveByDriverID(ctx, driverID)
if existingTrip != nil {
    return nil, ErrDriverHasActiveTrip  // Reject!
}
// Safe to create new trip
```

---

## 3.6 PaymentRepository

### Interface:

```go
type PaymentRepository interface {
    Create(ctx context.Context, payment *domain.Payment) error
    GetByID(ctx context.Context, id string) (*domain.Payment, error)
    GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error)
    UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error
}
```

### Methods Explained:

| Method | SQL Operation | Used By | Purpose |
|--------|---------------|---------|---------|
| `Create` | `INSERT INTO payments` | PaymentService | Create payment record |
| `GetByID` | `SELECT WHERE id=$1` | - | Get payment details |
| `GetByIdempotencyKey` | `SELECT WHERE idempotency_key=$1` | PaymentService | **Check for duplicate** |
| `UpdateStatus` | `UPDATE SET status=$1` | PaymentService | PENDING → SUCCESS/FAILED |

### Idempotency Implementation:

```go
func (r *PaymentRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Payment, error) {
    query := `
        SELECT id, trip_id, amount, status, idempotency_key 
        FROM payments 
        WHERE idempotency_key = $1
    `
    var payment domain.Payment
    err := r.q.QueryRowContext(ctx, query, key).Scan(...)
    
    if errors.Is(err, sql.ErrNoRows) {
        return nil, nil  // No existing payment - safe to create
    }
    return &payment, err
}
```

---

## 3.7 Transaction Support (`postgres/db.go`)

```go
// Querier allows the same repository code to work with DB or Transaction
type Querier interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Both *sql.DB and *sql.Tx implement this interface!
```

### Usage Pattern:

```go
// Repository accepts Querier
type RideRepository struct {
    q Querier
}

// Create with DB (normal use)
func NewRideRepository(db *sql.DB) *RideRepository {
    return &RideRepository{q: db}
}

// Create with Transaction (atomic operations)
func NewRideRepositoryWithTx(tx *sql.Tx) *RideRepository {
    return &RideRepository{q: tx}
}
```

### Why This Matters:

```go
// Atomic assignment: update ride AND driver status together
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

txRideRepo := postgres.NewRideRepositoryWithTx(tx)
txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)

txRideRepo.Update(ctx, ride)       // Uses same transaction
txDriverRepo.UpdateStatus(ctx, id) // Uses same transaction

tx.Commit()  // Both succeed or both fail
```

---

## 3.8 Shared Errors (`errors.go`)

```go
package repository

import "errors"

var (
    // ErrNotFound is returned when entity doesn't exist
    ErrNotFound = errors.New("entity not found")
)
```

### Usage:

```go
// In repository
if errors.Is(err, sql.ErrNoRows) {
    return nil, repository.ErrNotFound
}

// In handler
if errors.Is(err, repository.ErrNotFound) {
    c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
}
```

---

# 4. Redis Layer

> **Location:** `internal/redis/`
> 
> **Purpose:** Real-time operations that need sub-millisecond latency

---

## 4.1 Architecture

```
redis/
├── interfaces.go   ← LocationStoreInterface, LockStoreInterface
├── location.go     ← GEOADD, GEORADIUS operations
└── lock.go         ← SET NX EX for distributed locking
```

---

## 4.2 Interfaces (`interfaces.go`)

```go
package redis

import (
    "context"
    "time"
)

// LocationStoreInterface for driver GPS operations
type LocationStoreInterface interface {
    UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error
    FindNearbyDrivers(ctx context.Context, lat, lng, radiusKm float64) ([]DriverLocation, error)
    RemoveLocation(ctx context.Context, driverID string) error
}

// LockStoreInterface for distributed locking
type LockStoreInterface interface {
    AcquireDriverLock(ctx context.Context, driverID string, ttl time.Duration) (bool, error)
    ReleaseDriverLock(ctx context.Context, driverID string) error
}

// DriverLocation contains driver position and distance
type DriverLocation struct {
    DriverID string
    Lat      float64
    Lng      float64
    Distance float64  // Distance from query point in km
}
```

---

## 4.3 LocationStore (`location.go`)

```go
const locationKey = "driver:locations"

type LocationStore struct {
    client *redis.Client
}

// UpdateLocation stores driver position using GEOADD
func (s *LocationStore) UpdateLocation(ctx context.Context, driverID string, lat, lng float64) error {
    return s.client.GeoAdd(ctx, locationKey, &redis.GeoLocation{
        Name:      driverID,
        Longitude: lng,  // Redis expects lng first!
        Latitude:  lat,
    }).Err()
}

// FindNearbyDrivers queries drivers within radius using GEORADIUS
func (s *LocationStore) FindNearbyDrivers(ctx context.Context, lat, lng, radiusKm float64) ([]DriverLocation, error) {
    results, err := s.client.GeoRadius(ctx, locationKey, lng, lat, &redis.GeoRadiusQuery{
        Radius:    radiusKm,
        Unit:      "km",
        WithDist:  true,   // Include distance
        WithCoord: true,   // Include coordinates
        Sort:      "ASC",  // Closest first
    }).Result()
    
    if err != nil {
        return nil, err
    }
    
    locations := make([]DriverLocation, len(results))
    for i, r := range results {
        locations[i] = DriverLocation{
            DriverID: r.Name,
            Lat:      r.Latitude,
            Lng:      r.Longitude,
            Distance: r.Dist,
        }
    }
    return locations, nil
}

// RemoveLocation removes driver from GEO index
func (s *LocationStore) RemoveLocation(ctx context.Context, driverID string) error {
    return s.client.ZRem(ctx, locationKey, driverID).Err()
}
```

### Redis Commands Used:

| Method | Redis Command | Complexity | Description |
|--------|---------------|------------|-------------|
| `UpdateLocation` | `GEOADD driver:locations <lng> <lat> <id>` | O(log N) | Add/update position |
| `FindNearbyDrivers` | `GEORADIUS driver:locations <lng> <lat> <radius> km WITHDIST WITHCOORD ASC` | O(N+log M) | Find nearby |
| `RemoveLocation` | `ZREM driver:locations <id>` | O(log N) | Remove from index |

### Why GEOADD/GEORADIUS:

```
Traditional approach:
  - Store lat/lng in SQL
  - Query: SELECT * WHERE distance(lat, lng, $1, $2) < $3
  - Problem: Full table scan, O(N) for each query

Redis GEO approach:
  - Uses geohash-based sorted set internally
  - Query: GEORADIUS (uses spatial indexing)
  - O(log N) for updates, O(N) only for results within radius
```

---

## 4.4 LockStore (`lock.go`)

```go
type LockStore struct {
    client *redis.Client
}

// AcquireDriverLock tries to acquire exclusive lock on driver
func (s *LockStore) AcquireDriverLock(ctx context.Context, driverID string, ttl time.Duration) (bool, error) {
    key := "lock:driver:" + driverID
    
    // SET key value NX EX ttl
    // NX = only set if not exists
    // EX = expire after ttl seconds
    result, err := s.client.SetNX(ctx, key, "locked", ttl).Result()
    if err != nil {
        return false, err
    }
    return result, nil  // true if acquired, false if already locked
}

// ReleaseDriverLock releases the lock
func (s *LockStore) ReleaseDriverLock(ctx context.Context, driverID string) error {
    key := "lock:driver:" + driverID
    return s.client.Del(ctx, key).Err()
}
```

### Lock Pattern for Matching:

```
Driver 1: ONLINE at location (12.9, 77.5)
Driver 2: ONLINE at location (12.9, 77.5)

Ride Request comes in...

MatchingService:
  1. GEORADIUS → [Driver1, Driver2]
  2. Try lock Driver1:
     - SET lock:driver:driver1 "locked" NX EX 10
     - Returns TRUE (acquired)
  3. Check Driver1 status in PostgreSQL → ONLINE ✓
  4. Assign Driver1 to ride
  5. Release lock: DEL lock:driver:driver1

Concurrent request (same ride):
  1. GEORADIUS → [Driver1, Driver2]
  2. Try lock Driver1:
     - SET lock:driver:driver1 "locked" NX EX 10
     - Returns FALSE (already locked!)
  3. Try lock Driver2:
     - SET lock:driver:driver2 "locked" NX EX 10
     - Returns TRUE
  ... continues
```

### Why TTL is Critical:

```
Without TTL:
  - Service crashes after acquiring lock
  - Lock never released
  - Driver permanently locked out = DEADLOCK

With TTL (10 seconds):
  - Service crashes after acquiring lock
  - After 10 seconds, lock auto-expires
  - Driver available again
```

---

# 5. Service Layer

> **Location:** `internal/service/`
> 
> **Purpose:** Business logic orchestration

---

## 5.1 Architecture

```
service/
├── ride.go        ← RideService: create rides, get status
├── driver.go      ← DriverService: update location
├── matching.go    ← MatchingService: find and assign drivers
├── trip.go        ← TripService: start/end trips
├── payment.go     ← PaymentService: process payments
├── surge.go       ← SurgeService: dynamic pricing
└── errors.go      ← Business domain errors
```

---

## 5.2 RideService (`ride.go`)

```go
type RideService struct {
    rideRepo        repository.RideRepository
    matchingService MatchingServiceInterface
    surgeService    *SurgeService
}

type CreateRideRequest struct {
    RiderID        string
    PickupLat      float64
    PickupLng      float64
    DestinationLat float64
    DestinationLng float64
    Tier           domain.DriverTier
}

type CreateRideResponse struct {
    Ride            *domain.Ride
    DriverAssigned  bool
    DriverID        string
    SurgeMultiplier float64
}
```

### CreateRide Flow:

```go
func (s *RideService) CreateRide(ctx context.Context, req CreateRideRequest) (*CreateRideResponse, error) {
    // 1. Validate input
    if err := s.validateCreateRequest(req); err != nil {
        return nil, err
    }
    
    // 2. Calculate surge
    surgeMultiplier := 1.0
    if s.surgeService != nil {
        surgeMultiplier = s.surgeService.GetMultiplier(ctx, req.PickupLat, req.PickupLng)
    }
    
    // 3. Create ride entity
    ride := &domain.Ride{
        ID:              uuid.New().String(),
        RiderID:         req.RiderID,
        PickupLat:       req.PickupLat,
        PickupLng:       req.PickupLng,
        DestinationLat:  req.DestinationLat,
        DestinationLng:  req.DestinationLng,
        Status:          domain.RideStatusRequested,
        SurgeMultiplier: surgeMultiplier,
    }
    
    // 4. Persist to database
    if err := s.rideRepo.Create(ctx, ride); err != nil {
        return nil, err
    }
    
    // 5. Trigger matching synchronously
    matchResult, err := s.matchingService.Match(ctx, MatchRequest{
        RideID: ride.ID,
        Lat:    req.PickupLat,
        Lng:    req.PickupLng,
        Tier:   req.Tier,
    })
    
    if err == ErrNoDriverAvailable {
        return &CreateRideResponse{Ride: ride, DriverAssigned: false}, nil
    }
    
    return &CreateRideResponse{
        Ride:            matchResult.Ride,
        DriverAssigned:  true,
        DriverID:        matchResult.DriverID,
        SurgeMultiplier: surgeMultiplier,
    }, nil
}
```

---

## 5.3 DriverService (`driver.go`)

```go
type DriverService struct {
    locationStore redis.LocationStoreInterface
    driverRepo    repository.DriverRepository
}

type UpdateLocationRequest struct {
    DriverID string
    Lat      float64
    Lng      float64
}

func (s *DriverService) UpdateLocation(ctx context.Context, req UpdateLocationRequest) error {
    // 1. Validate
    if req.DriverID == "" {
        return ErrInvalidDriverID
    }
    if !isValidLatitude(req.Lat) || !isValidLongitude(req.Lng) {
        return ErrInvalidLocation
    }
    
    // 2. Update Redis (real-time location)
    if err := s.locationStore.UpdateLocation(ctx, req.DriverID, req.Lat, req.Lng); err != nil {
        return err
    }
    
    // 3. Set status to ONLINE in PostgreSQL
    // Ignore error if driver not found (graceful degradation)
    _ = s.driverRepo.UpdateStatus(ctx, req.DriverID, domain.DriverStatusOnline)
    
    return nil
}
```

### Why Two Stores:

| Operation | Store | Reason |
|-----------|-------|--------|
| GPS Location | Redis | Updated every 5s, needs fast writes |
| Driver Status | PostgreSQL | Business-critical, needs durability |

---

## 5.4 MatchingService (`matching.go`)

```go
type MatchingService struct {
    db            *sql.DB
    locationStore redis.LocationStoreInterface
    lockStore     redis.LockStoreInterface
    driverRepo    repository.DriverRepository
    rideRepo      repository.RideRepository
}

const (
    defaultSearchRadiusKm = 5.0
    driverLockTTL         = 10 * time.Second
)

func (s *MatchingService) Match(ctx context.Context, req MatchRequest) (*MatchResult, error) {
    // 1. Find nearby drivers from Redis
    nearbyDrivers, err := s.locationStore.FindNearbyDrivers(
        ctx, req.Lat, req.Lng, defaultSearchRadiusKm,
    )
    if err != nil {
        return nil, err
    }
    
    // 2. Get ride for assignment
    ride, err := s.rideRepo.GetByID(ctx, req.RideID)
    if err != nil {
        return nil, err
    }
    
    // 3. Try to assign each driver (closest first)
    for _, loc := range nearbyDrivers {
        // 3a. Acquire lock
        locked, err := s.lockStore.AcquireDriverLock(ctx, loc.DriverID, driverLockTTL)
        if err != nil || !locked {
            continue  // Try next driver
        }
        
        // 3b. Check driver status in DB (must be ONLINE)
        driver, err := s.driverRepo.GetByID(ctx, loc.DriverID)
        if err != nil || driver.Status != domain.DriverStatusOnline {
            s.lockStore.ReleaseDriverLock(ctx, loc.DriverID)
            continue
        }
        
        // 3c. Filter by tier if requested
        if req.Tier != "" && driver.Tier != req.Tier {
            s.lockStore.ReleaseDriverLock(ctx, loc.DriverID)
            continue
        }
        
        // 3d. Assign driver (in transaction)
        result, err := s.assignDriver(ctx, ride, driver)
        if err != nil {
            s.lockStore.ReleaseDriverLock(ctx, loc.DriverID)
            continue
        }
        
        // 3e. Success - release lock and return
        s.lockStore.ReleaseDriverLock(ctx, loc.DriverID)
        return result, nil
    }
    
    return nil, ErrNoDriverAvailable
}

func (s *MatchingService) assignDriver(ctx context.Context, ride *domain.Ride, driver *domain.Driver) (*MatchResult, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    txRideRepo := postgres.NewRideRepositoryWithTx(tx)
    txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)
    
    // Update ride
    ride.Status = domain.RideStatusAssigned
    ride.AssignedDriverID = driver.ID
    if err = txRideRepo.Update(ctx, ride); err != nil {
        return nil, err
    }
    
    // Update driver status
    if err = txDriverRepo.UpdateStatus(ctx, driver.ID, domain.DriverStatusOnTrip); err != nil {
        return nil, err
    }
    
    if err = tx.Commit(); err != nil {
        return nil, err
    }
    
    return &MatchResult{DriverID: driver.ID, Ride: ride}, nil
}
```

### Matching Algorithm Visualization:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MATCHING ALGORITHM                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Ride Request: Pickup at (12.9716, 77.5946)                        │
│                                                                      │
│   Step 1: GEORADIUS (Redis)                                         │
│   ┌─────────────────────────────────────────┐                       │
│   │  Nearby Drivers (sorted by distance):   │                       │
│   │  1. Driver-A: 0.5km (ONLINE)            │                       │
│   │  2. Driver-B: 1.2km (ON_TRIP)           │                       │
│   │  3. Driver-C: 2.1km (ONLINE)            │                       │
│   └─────────────────────────────────────────┘                       │
│                                                                      │
│   Step 2: Try Driver-A                                              │
│   ┌─────────────────────────────────────────┐                       │
│   │  SET lock:driver:A "locked" NX EX 10    │ → TRUE (acquired)    │
│   │  SELECT status FROM drivers WHERE id=A  │ → ONLINE ✓            │
│   │  BEGIN TRANSACTION                       │                       │
│   │    UPDATE rides SET driver_id=A          │                       │
│   │    UPDATE drivers SET status='ON_TRIP'   │                       │
│   │  COMMIT                                  │                       │
│   │  DEL lock:driver:A                       │                       │
│   └─────────────────────────────────────────┘                       │
│                                                                      │
│   Result: Driver-A assigned to ride                                 │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5.5 TripService (`trip.go`)

```go
type TripService struct {
    db             *sql.DB
    tripRepo       repository.TripRepository
    rideRepo       repository.RideRepository
    driverRepo     repository.DriverRepository
    paymentService *PaymentService
}

// StartTrip creates a new trip when driver accepts ride
func (s *TripService) StartTrip(ctx context.Context, req StartTripRequest) (*domain.Trip, error) {
    // 1. Check if driver already has active trip
    existingTrip, _ := s.tripRepo.GetActiveByDriverID(ctx, req.DriverID)
    if existingTrip != nil {
        return nil, ErrDriverHasActiveTrip
    }
    
    // 2. Verify ride is assigned to this driver
    ride, err := s.rideRepo.GetByID(ctx, req.RideID)
    if err != nil {
        return nil, err
    }
    if ride.Status != domain.RideStatusAssigned {
        return nil, ErrRideNotAssigned
    }
    if ride.AssignedDriverID != req.DriverID {
        return nil, ErrDriverNotAssignedToRide
    }
    
    // 3. Create trip
    trip := &domain.Trip{
        ID:        uuid.New().String(),
        RideID:    req.RideID,
        DriverID:  req.DriverID,
        Status:    domain.TripStatusStarted,
        StartedAt: time.Now(),
    }
    
    return trip, s.tripRepo.Create(ctx, trip)
}

// EndTrip completes trip, calculates fare, triggers payment
func (s *TripService) EndTrip(ctx context.Context, req EndTripRequest) (*EndTripResponse, error) {
    // 1. Get trip
    trip, err := s.tripRepo.GetByID(ctx, req.TripID)
    if err != nil {
        return nil, err
    }
    if trip.Status == domain.TripStatusEnded {
        return nil, ErrTripAlreadyEnded
    }
    
    // 2. Get ride for surge multiplier
    ride, _ := s.rideRepo.GetByID(ctx, trip.RideID)
    
    // 3. Calculate fare with surge
    baseFare := s.calculateFare(trip.StartedAt, time.Now())
    surgeMultiplier := ride.SurgeMultiplier
    if surgeMultiplier < 1.0 {
        surgeMultiplier = 1.0
    }
    fare := baseFare * surgeMultiplier
    
    // 4. Update trip and driver in transaction
    tx, _ := s.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    
    txTripRepo := postgres.NewTripRepositoryWithTx(tx)
    txDriverRepo := postgres.NewDriverRepositoryWithTx(tx)
    
    trip.Status = domain.TripStatusEnded
    trip.Fare = fare
    trip.EndedAt = time.Now()
    txTripRepo.Update(ctx, trip)
    
    // Reset driver to ONLINE
    txDriverRepo.UpdateStatus(ctx, trip.DriverID, domain.DriverStatusOnline)
    
    tx.Commit()
    
    // 5. Process payment (after transaction)
    payment, _ := s.paymentService.ProcessPayment(ctx, ProcessPaymentRequest{
        TripID: trip.ID,
        Amount: fare,
    })
    
    return &EndTripResponse{Trip: trip, Payment: payment}, nil
}

// Fare calculation
func (s *TripService) calculateFare(startTime, endTime time.Time) float64 {
    const (
        baseFare      = 2.0   // ₹2 base
        perMinuteRate = 0.5   // ₹0.50 per minute
        minimumFare   = 5.0   // ₹5 minimum
    )
    
    duration := endTime.Sub(startTime)
    fare := baseFare + (duration.Minutes() * perMinuteRate)
    
    if fare < minimumFare {
        return minimumFare
    }
    return fare
}
```

---

## 5.6 PaymentService (`payment.go`)

```go
type PSP interface {
    Charge(ctx context.Context, amount float64) (bool, error)
}

type MockPSP struct{}

func (p *MockPSP) Charge(ctx context.Context, amount float64) (bool, error) {
    return true, nil  // Always succeeds
}

type PaymentService struct {
    paymentRepo repository.PaymentRepository
    psp         PSP
}

func (s *PaymentService) ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*domain.Payment, error) {
    // 1. Generate idempotency key
    idempotencyKey := fmt.Sprintf("payment:%s", req.TripID)
    
    // 2. Check for existing payment (IDEMPOTENCY)
    existingPayment, _ := s.paymentRepo.GetByIdempotencyKey(ctx, idempotencyKey)
    if existingPayment != nil {
        return existingPayment, nil  // Return existing, don't duplicate
    }
    
    // 3. Create payment in PENDING state
    payment := &domain.Payment{
        ID:             uuid.New().String(),
        TripID:         req.TripID,
        Amount:         req.Amount,
        Status:         domain.PaymentStatusPending,
        IdempotencyKey: idempotencyKey,
    }
    s.paymentRepo.Create(ctx, payment)
    
    // 4. Call PSP
    success, err := s.psp.Charge(ctx, req.Amount)
    
    // 5. Update status based on result
    if err != nil || !success {
        s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusFailed)
        payment.Status = domain.PaymentStatusFailed
    } else {
        s.paymentRepo.UpdateStatus(ctx, payment.ID, domain.PaymentStatusSuccess)
        payment.Status = domain.PaymentStatusSuccess
    }
    
    return payment, nil
}
```

---

## 5.7 SurgeService (`surge.go`)

```go
type SurgeService struct {
    locationStore redis.LocationStoreInterface
    rideRepo      repository.RideRepository
}

func (s *SurgeService) GetMultiplier(ctx context.Context, lat, lng float64) float64 {
    config := DefaultSurgeConfig()  // radius=5km, max=2.0
    
    // Count supply (online drivers)
    supply := s.countDriversInArea(ctx, lat, lng, config.RadiusKm)
    
    // Count demand (active ride requests)
    demand := s.countActiveRequestsInArea(ctx, lat, lng, config.RadiusKm)
    
    // Calculate multiplier
    return s.calculateSurgeMultiplier(supply, demand, config)
}

func (s *SurgeService) calculateSurgeMultiplier(supply, demand int, config SurgeConfig) float64 {
    if supply == 0 {
        if demand > 0 {
            return config.MaxSurge  // 2.0x when no drivers
        }
        return 1.0
    }
    
    ratio := float64(demand) / float64(supply)
    
    switch {
    case ratio >= 2.0:
        return 2.0   // High surge
    case ratio >= 1.5:
        return 1.5   // Medium surge
    case ratio >= 1.2:
        return 1.25  // Low surge
    default:
        return 1.0   // No surge
    }
}
```

### Surge Calculation Example:

```
Area: 5km radius around pickup point

Supply (drivers): 2
Demand (active rides): 5

Ratio = 5/2 = 2.5 → >= 2.0 → 2.0x surge

Fare without surge: ₹10
Fare with surge: ₹10 × 2.0 = ₹20
```

---

## 5.8 Service Errors (`errors.go`)

```go
package service

import "errors"

var (
    // Ride errors
    ErrInvalidRiderID            = errors.New("invalid rider ID")
    ErrInvalidPickupLocation     = errors.New("invalid pickup location")
    ErrInvalidDestinationLocation = errors.New("invalid destination location")
    ErrInvalidRideID             = errors.New("invalid ride ID")
    
    // Driver errors
    ErrInvalidDriverID           = errors.New("invalid driver ID")
    ErrInvalidLocation           = errors.New("invalid location")
    ErrNoDriverAvailable         = errors.New("no driver available")
    
    // Trip errors
    ErrInvalidTripID             = errors.New("invalid trip ID")
    ErrRideNotAssigned           = errors.New("ride not assigned")
    ErrDriverNotAssignedToRide   = errors.New("driver not assigned to this ride")
    ErrDriverHasActiveTrip       = errors.New("driver already has active trip")
    ErrTripAlreadyEnded          = errors.New("trip already ended")
    
    // Payment errors
    ErrInvalidPaymentAmount      = errors.New("invalid payment amount")
    ErrInvalidPaymentID          = errors.New("invalid payment ID")
)
```

---

# 6. Handler Layer

> **Location:** `internal/handler/`
> 
> **Purpose:** HTTP boundary - parse requests, call services, return responses

---

## 6.1 Architecture

```
handler/
├── user.go      ← UserHandler: Register, GetAll
├── driver.go    ← DriverHandler: Register, UpdateLocation, AcceptRide, GetAll
├── ride.go      ← RideHandler: CreateRide, GetRide, GetAll
├── trip.go      ← TripHandler: EndTrip, GetTrip
└── response.go  ← respondJSON(), respondError(), ErrorResponse
```

---

## 6.2 Response Helpers (`response.go`)

```go
package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "ride/internal/repository"
    "ride/internal/service"
)

// ErrorResponse is the standard error format
type ErrorResponse struct {
    Error string `json:"error"`
}

// respondJSON sends a JSON response
func respondJSON(c *gin.Context, status int, data interface{}) {
    c.JSON(status, data)
}

// respondError maps service/repository errors to HTTP status codes
func respondError(c *gin.Context, err error) {
    switch err {
    case repository.ErrNotFound:
        c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
    case service.ErrInvalidRiderID, service.ErrInvalidDriverID,
         service.ErrInvalidPickupLocation, service.ErrInvalidDestinationLocation:
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
    case service.ErrNoDriverAvailable:
        c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: err.Error()})
    case service.ErrRideNotAssigned, service.ErrDriverNotAssignedToRide:
        c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
    case service.ErrDriverHasActiveTrip:
        c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
    case service.ErrTripAlreadyEnded:
        c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
    default:
        c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
    }
}
```

---

## 6.3 RideHandler (`ride.go`)

```go
type RideHandler struct {
    rideService *service.RideService
    rideRepo    repository.RideRepository
}

// POST /v1/rides
func (h *RideHandler) CreateRide(c *gin.Context) {
    var req CreateRideRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
        return
    }
    
    result, err := h.rideService.CreateRide(c.Request.Context(), service.CreateRideRequest{
        RiderID:        req.RiderID,
        PickupLat:      req.PickupLat,
        PickupLng:      req.PickupLng,
        DestinationLat: req.DestinationLat,
        DestinationLng: req.DestinationLng,
        Tier:           domain.DriverTier(req.Tier),
    })
    if err != nil {
        respondError(c, err)
        return
    }
    
    respondJSON(c, http.StatusCreated, CreateRideResponse{
        ID:               result.Ride.ID,
        Status:           string(result.Ride.Status),
        AssignedDriverID: result.DriverID,
        DriverAssigned:   result.DriverAssigned,
        SurgeMultiplier:  result.SurgeMultiplier,
        SurgeActive:      result.SurgeMultiplier > 1.0,
        // ... other fields
    })
}

// GET /v1/rides/:id
func (h *RideHandler) GetRide(c *gin.Context) {
    rideID := c.Param("id")
    
    ride, err := h.rideService.GetRideStatus(c.Request.Context(), rideID)
    if err != nil {
        respondError(c, err)
        return
    }
    
    respondJSON(c, http.StatusOK, GetRideResponse{
        ID:               ride.ID,
        Status:           string(ride.Status),
        AssignedDriverID: ride.AssignedDriverID,
        SurgeMultiplier:  ride.SurgeMultiplier,
        // ... other fields
    })
}
```

---

## 6.4 DriverHandler (`driver.go`)

```go
type DriverHandler struct {
    driverService *service.DriverService
    tripService   *service.TripService
    driverRepo    repository.DriverRepository
}

// POST /v1/drivers/:id/location
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
    
    respondJSON(c, http.StatusOK, gin.H{"status": "location updated"})
}

// POST /v1/drivers/:id/accept
func (h *DriverHandler) AcceptRide(c *gin.Context) {
    driverID := c.Param("id")
    
    var req AcceptRideRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
        return
    }
    
    // Start trip when driver accepts
    trip, err := h.tripService.StartTrip(c.Request.Context(), service.StartTripRequest{
        RideID:   req.RideID,
        DriverID: driverID,
    })
    if err != nil {
        respondError(c, err)
        return
    }
    
    respondJSON(c, http.StatusOK, AcceptRideResponse{
        TripID:   trip.ID,
        RideID:   trip.RideID,
        DriverID: trip.DriverID,
        Status:   string(trip.Status),
    })
}
```

---

# 7. Middleware Layer

> **Location:** `internal/middleware/`

---

## 7.1 CORS Middleware (`cors.go`)

```go
package middleware

import "github.com/gin-gonic/gin"

func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, Idempotency-Key")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

---

## 7.2 Idempotency Middleware (`idempotency.go`)

```go
package middleware

const (
    idempotencyHeader = "Idempotency-Key"
    idempotencyTTL    = 24 * time.Hour
)

type cachedResponse struct {
    StatusCode int             `json:"status_code"`
    Body       json.RawMessage `json:"body"`
    Headers    http.Header     `json:"headers"`
}

func IdempotencyMiddleware(redisClient *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only for mutating methods
        if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
            c.Next()
            return
        }
        
        // Get idempotency key
        key := c.GetHeader(idempotencyHeader)
        if key == "" {
            c.Next()
            return
        }
        
        cacheKey := "idempotency:" + key
        
        // Check for cached response
        cached, err := getCachedResponse(c.Request.Context(), redisClient, cacheKey)
        if err == nil && cached != nil {
            // Return cached response (short-circuit)
            c.Data(cached.StatusCode, "application/json", cached.Body)
            c.Abort()
            return
        }
        
        // Capture response
        w := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
        c.Writer = w
        
        c.Next()
        
        // Cache successful responses
        if c.Writer.Status() >= 200 && c.Writer.Status() < 500 {
            response := cachedResponse{
                StatusCode: c.Writer.Status(),
                Body:       w.body.Bytes(),
            }
            setCachedResponse(c.Request.Context(), redisClient, cacheKey, &response, idempotencyTTL)
        }
    }
}
```

### How Idempotency Works:

```
Request 1: POST /v1/rides (Idempotency-Key: abc123)
  → Check Redis: idempotency:abc123 → NOT FOUND
  → Process request → Create ride
  → Cache response in Redis
  → Return: 201 Created {id: "ride-1"}

Request 2: POST /v1/rides (Idempotency-Key: abc123)  [DUPLICATE]
  → Check Redis: idempotency:abc123 → FOUND
  → Return cached: 201 Created {id: "ride-1"}
  → No new ride created!
```

---

# 8. Configuration & App Setup

---

## 8.1 Configuration (`internal/config/config.go`)

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    NewRelic NewRelicConfig
}

type ServerConfig struct {
    Port         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    Name     string
    SSLMode  string
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
}

type NewRelicConfig struct {
    Enabled    bool
    LicenseKey string
    AppName    string
}

func Load() *Config {
    return &Config{
        Server: ServerConfig{
            Port: getEnv("SERVER_PORT", "8080"),
            // ...
        },
        Database: DatabaseConfig{
            Host: getEnv("DB_HOST", "localhost"),
            // ...
        },
        // ...
    }
}
```

---

## 8.2 Router Setup (`internal/app/router.go`)

```go
func NewRouter(deps RouterDeps) *gin.Engine {
    router := gin.New()
    
    // Global middleware (order matters!)
    router.Use(gin.Logger())                        // 1. Logging
    router.Use(gin.Recovery())                      // 2. Panic recovery
    router.Use(middleware.CORS())                   // 3. CORS
    router.Use(nrgin.Middleware(deps.NewRelicApp))  // 4. New Relic APM
    
    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })
    
    // API v1
    v1 := router.Group("/v1")
    {
        // Users
        users := v1.Group("/users")
        users.POST("/register", deps.UserHandler.Register)
        users.GET("", deps.UserHandler.GetAll)
        
        // Rides
        rides := v1.Group("/rides")
        rides.POST("", deps.RideHandler.CreateRide)
        rides.GET("", deps.RideHandler.GetAll)
        rides.GET("/:id", deps.RideHandler.GetRide)
        
        // Drivers
        drivers := v1.Group("/drivers")
        drivers.POST("/register", deps.DriverHandler.Register)
        drivers.GET("", deps.DriverHandler.GetAll)
        drivers.POST("/:id/location", deps.DriverHandler.UpdateLocation)
        drivers.POST("/:id/accept", deps.DriverHandler.AcceptRide)
        
        // Trips
        trips := v1.Group("/trips")
        trips.POST("/:id/end", deps.TripHandler.EndTrip)
        trips.GET("/:id", deps.TripHandler.GetTrip)
    }
    
    return router
}
```

---

## 8.3 Main Entry Point (`cmd/server/main.go`)

```go
func main() {
    // 1. Load config
    cfg := config.Load()
    
    // 2. Initialize PostgreSQL
    db, _ := app.NewDatabase(ctx, cfg.Database)
    defer db.Close()
    
    // 3. Initialize Redis
    redisClient, _ := app.NewRedisClient(ctx, cfg.Redis)
    defer redisClient.Close()
    
    // 4. Initialize New Relic (optional)
    var nrApp *newrelic.Application
    if cfg.NewRelic.Enabled {
        nrApp, _ = newrelic.NewApplication(...)
    }
    
    // 5. Wire dependencies
    server := wireServer(db, redisClient, nrApp, cfg)
    
    // 6. Start server
    go server.ListenAndServe()
    
    // 7. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    server.Shutdown(ctx)
}
```

---

# 9. Database Schema

**File:** `scripts/schema.sql`

```sql
-- Users (riders)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Drivers
CREATE TABLE IF NOT EXISTS drivers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'OFFLINE',
    tier VARCHAR(20) NOT NULL DEFAULT 'BASIC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT drivers_status_check CHECK (status IN ('ONLINE', 'OFFLINE', 'ON_TRIP')),
    CONSTRAINT drivers_tier_check CHECK (tier IN ('BASIC', 'PREMIUM'))
);

-- Rides
CREATE TABLE IF NOT EXISTS rides (
    id VARCHAR(36) PRIMARY KEY,
    rider_id VARCHAR(36) NOT NULL,
    pickup_lat DOUBLE PRECISION NOT NULL,
    pickup_lng DOUBLE PRECISION NOT NULL,
    destination_lat DOUBLE PRECISION NOT NULL,
    destination_lng DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'REQUESTED',
    assigned_driver_id VARCHAR(36),
    surge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT rides_status_check CHECK (status IN ('REQUESTED', 'ASSIGNED', 'CANCELLED')),
    CONSTRAINT rides_surge_check CHECK (surge_multiplier >= 1.0 AND surge_multiplier <= 5.0)
);

-- Trips
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(36) PRIMARY KEY,
    ride_id VARCHAR(36) NOT NULL UNIQUE,
    driver_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'STARTED',
    fare DOUBLE PRECISION DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    CONSTRAINT trips_status_check CHECK (status IN ('STARTED', 'PAUSED', 'ENDED'))
);

-- Payments
CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    idempotency_key VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT payments_status_check CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id ON trips(driver_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency ON payments(idempotency_key);

-- Constraint: One active trip per driver
CREATE UNIQUE INDEX IF NOT EXISTS idx_trips_active_driver
ON trips (driver_id)
WHERE status != 'ENDED';
```

---

# 10. Complete API Reference

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| `POST` | `/v1/users/register` | Register rider | `{name, phone}` | `{id, name, phone}` |
| `GET` | `/v1/users` | List all users | - | `[{id, name, phone}]` |
| `POST` | `/v1/drivers/register` | Register driver | `{name, phone, tier}` | `{id, name, status, tier}` |
| `GET` | `/v1/drivers` | List all drivers | - | `[{id, name, status, tier}]` |
| `POST` | `/v1/drivers/:id/location` | Update location | `{lat, lng}` | `{status: "updated"}` |
| `POST` | `/v1/drivers/:id/accept` | Accept ride | `{ride_id}` | `{trip_id, status}` |
| `POST` | `/v1/rides` | Request ride | `{rider_id, pickup_lat, pickup_lng, destination_lat, destination_lng}` | `{id, status, surge_multiplier}` |
| `GET` | `/v1/rides/:id` | Get ride status | - | `{id, status, assigned_driver_id}` |
| `GET` | `/v1/rides` | List all rides | - | `[{id, status, ...}]` |
| `POST` | `/v1/trips/:id/end` | End trip | - | `{trip, payment}` |
| `GET` | `/v1/trips/:id` | Get trip details | - | `{id, fare, status}` |
| `GET` | `/health` | Health check | - | `{status: "ok"}` |

---

# 11. Data Flow Diagrams

## Complete Ride Lifecycle:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         COMPLETE RIDE LIFECYCLE                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. USER REGISTRATION                                                        │
│     POST /v1/users/register                                                  │
│     └── UserHandler → UserRepo.Create() → PostgreSQL                        │
│                                                                              │
│  2. DRIVER REGISTRATION                                                      │
│     POST /v1/drivers/register                                                │
│     └── DriverHandler → DriverRepo.Create() → PostgreSQL                    │
│                                                                              │
│  3. DRIVER GOES ONLINE                                                       │
│     POST /v1/drivers/:id/location                                            │
│     └── DriverHandler → DriverService.UpdateLocation()                      │
│         ├── LocationStore.UpdateLocation() → Redis GEOADD                   │
│         └── DriverRepo.UpdateStatus(ONLINE) → PostgreSQL                    │
│                                                                              │
│  4. USER REQUESTS RIDE                                                       │
│     POST /v1/rides                                                           │
│     └── RideHandler → RideService.CreateRide()                              │
│         ├── SurgeService.GetMultiplier() → Calculate surge                  │
│         ├── RideRepo.Create() → PostgreSQL                                  │
│         └── MatchingService.Match()                                         │
│             ├── LocationStore.FindNearbyDrivers() → Redis GEORADIUS         │
│             ├── LockStore.AcquireDriverLock() → Redis SET NX                │
│             ├── DriverRepo.GetByID() → Verify ONLINE                        │
│             ├── BEGIN TRANSACTION                                           │
│             │   ├── RideRepo.Update(ASSIGNED)                               │
│             │   └── DriverRepo.UpdateStatus(ON_TRIP)                        │
│             ├── COMMIT                                                       │
│             └── LockStore.ReleaseDriverLock() → Redis DEL                   │
│                                                                              │
│  5. DRIVER ACCEPTS RIDE                                                      │
│     POST /v1/drivers/:id/accept                                              │
│     └── DriverHandler → TripService.StartTrip()                             │
│         ├── TripRepo.GetActiveByDriverID() → Check no existing trip         │
│         ├── RideRepo.GetByID() → Verify assignment                          │
│         └── TripRepo.Create() → PostgreSQL                                  │
│                                                                              │
│  6. TRIP ENDS                                                                │
│     POST /v1/trips/:id/end                                                   │
│     └── TripHandler → TripService.EndTrip()                                 │
│         ├── TripRepo.GetByID() → Get trip                                   │
│         ├── RideRepo.GetByID() → Get surge multiplier                       │
│         ├── Calculate fare: base × surge                                    │
│         ├── BEGIN TRANSACTION                                               │
│         │   ├── TripRepo.Update(ENDED, fare)                                │
│         │   └── DriverRepo.UpdateStatus(ONLINE)                             │
│         ├── COMMIT                                                           │
│         └── PaymentService.ProcessPayment()                                 │
│             ├── PaymentRepo.GetByIdempotencyKey() → Idempotency check       │
│             ├── PaymentRepo.Create(PENDING)                                 │
│             ├── MockPSP.Charge()                                            │
│             └── PaymentRepo.UpdateStatus(SUCCESS)                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

# 12. Concurrency & Safety Guarantees

## Race Conditions Prevented:

| Scenario | Prevention Mechanism |
|----------|---------------------|
| Two drivers accept same ride | Redis lock on driver + SQL transaction |
| Same driver accepts two rides | `GetActiveByDriverID()` check + DB unique index |
| Double payment for same trip | Idempotency key in payments table |
| Stale driver status | Verify status in DB after acquiring lock |

## Critical Design Decisions:

1. **Redis Lock + DB Transaction**
   - Redis lock prevents concurrent assignment attempts
   - DB transaction ensures atomic state changes
   - Both are needed: lock for speed, transaction for durability

2. **Lock TTL (10 seconds)**
   - Prevents deadlock if service crashes
   - Must be longer than transaction time

3. **Unique Index on Active Trips**
   ```sql
   CREATE UNIQUE INDEX idx_trips_active_driver
   ON trips (driver_id)
   WHERE status != 'ENDED';
   ```
   - Database enforces one active trip per driver
   - Backup safety even if application logic fails

---

# 13. Testing Strategy

## Test Files:

| File | Coverage |
|------|----------|
| `ride_test.go` | RideService validation, creation |
| `ride_creation_test.go` | Edge cases, invalid inputs |
| `driver_location_test.go` | Location updates, validation |
| `matching_test.go` | Driver matching, filtering |
| `trip_lifecycle_test.go` | Start/end trip, fare calculation |
| `concurrency_test.go` | Race conditions, locking |

## Mock Pattern:

```go
// In tests/mocks.go
type MockRideRepository struct {
    rides      map[string]*domain.Ride
    createErr  error
    mu         sync.RWMutex
}

func (m *MockRideRepository) Create(ctx context.Context, ride *domain.Ride) error {
    if m.createErr != nil {
        return m.createErr
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    m.rides[ride.ID] = ride
    return nil
}
```

## Running Tests:

```bash
# Run all tests
go test ./internal/tests/...

# Run with verbose output
go test -v ./internal/tests/...

# Run specific test
go test -v -run TestRideCreation ./internal/tests/...
```