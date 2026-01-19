# Low-Level Design Document - Ride Hailing System

## 1. Introduction & Scope

### Purpose
This LLD provides detailed design specifications for a ride hailing system supporting real-time driver matching, trip lifecycle management, and payment processing.

### Functional Requirements
- User registration and authentication
- Driver registration with location tracking
- Real-time ride requests and driver matching
- Trip lifecycle (start/pause/resume/end)
- Payment processing with multiple methods
- Surge pricing based on demand

### Non-Functional Requirements
- **Performance**: P95 < 1 second for driver matching
- **Scale**: 100k drivers, 10k ride requests/min, 200k location updates/sec
- **Availability**: 99.9% uptime
- **Data Consistency**: ACID transactions for critical operations

## 2. System Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend UI   │    │   Gin HTTP API  │    │   PostgreSQL    │
│   (React)       │◄──►│   (Go Backend)  │◄──►│   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │                        ▲
                              ▼                        │
                       ┌─────────────────┐             │
                       │     Redis       │             │
                       │   Cache/Queue   │             │
                       └─────────────────┘             │
                              ▲                        │
                              │                        │
                       ┌─────────────────┐             │
                       │   New Relic     │             │
                       │   Monitoring    │             │
                       └─────────────────┘             │
```

### Core Components
1. **HTTP Layer**: Gin router handling REST API requests
2. **Service Layer**: Business logic for rides, drivers, trips, payments
3. **Repository Layer**: Data access abstraction
4. **Cache Layer**: Redis for real-time data and locking
5. **Database Layer**: PostgreSQL for persistent storage

## 3. Database Schema Design

### Core Tables

```sql
-- Users (Riders)
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Drivers
CREATE TABLE drivers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'OFFLINE',
    tier VARCHAR(20) NOT NULL DEFAULT 'BASIC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Rides (Ride Requests)
CREATE TABLE rides (
    id VARCHAR(36) PRIMARY KEY,
    rider_id VARCHAR(36) NOT NULL,
    pickup_lat DOUBLE PRECISION NOT NULL,
    pickup_lng DOUBLE PRECISION NOT NULL,
    destination_lat DOUBLE PRECISION NOT NULL,
    destination_lng DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'REQUESTED',
    assigned_driver_id VARCHAR(36),
    surge_multiplier DOUBLE PRECISION DEFAULT 1.0,
    payment_method VARCHAR(20) DEFAULT 'CASH',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trips (Active Rides)
CREATE TABLE trips (
    id VARCHAR(36) PRIMARY KEY,
    ride_id VARCHAR(36) NOT NULL REFERENCES rides(id),
    driver_id VARCHAR(36) NOT NULL REFERENCES drivers(id),
    status VARCHAR(20) NOT NULL DEFAULT 'STARTED',
    fare DOUBLE PRECISION DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    total_paused_seconds INTEGER DEFAULT 0
);

-- Payments
CREATE TABLE payments (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL REFERENCES trips(id),
    amount DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Receipts
CREATE TABLE receipts (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL REFERENCES trips(id),
    base_fare DOUBLE PRECISION NOT NULL,
    surge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    surge_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_fare DOUBLE PRECISION NOT NULL,
    payment_method VARCHAR(20) NOT NULL,
    duration_minutes DOUBLE PRECISION NOT NULL,
    distance_km DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Key Constraints & Indexes
- Unique constraint: One active trip per driver
- Foreign key relationships maintain data integrity
- Optimized indexes for status queries and location-based searches

## 4. API Design

### REST Endpoints (POST-only as per requirements)

| Endpoint | Purpose | Request Body |
|----------|---------|--------------|
| `POST /v1/users/register` | Register new user | `{"name": "string", "phone": "string"}` |
| `POST /v1/drivers/register` | Register new driver | `{"name": "string", "phone": "string", "tier": "BASIC|PREMIUM"}` |
| `POST /v1/drivers/{id}/location` | Update driver location | `{"lat": float, "lng": float}` |
| `POST /v1/drivers/{id}/accept` | Accept ride request | `{"ride_id": "string"}` |
| `POST /v1/rides` | Create ride request | Full ride creation payload |
| `POST /v1/rides/{id}/cancel` | Cancel ride | `{"cancelled_by": "string", "reason": "string"}` |
| `POST /v1/trips/{id}/pause` | Pause active trip | `{}` |
| `POST /v1/trips/{id}/resume` | Resume paused trip | `{}` |
| `POST /v1/trips/{id}/end` | End trip & generate receipt | `{}` |
| `POST /v1/payments` | Process payment | `{"trip_id": "string", "amount": float}` |

### Response Format
```json
{
  "success": true,
  "data": { /* response data */ },
  "error": null
}
```

## 5. Class Design & Key Components

### Service Layer Classes

```go
// Core Services
type RideService struct {
    rideRepo repository.RideRepository
    matchingService *MatchingService
    surgeService *SurgeService
}

type DriverService struct {
    locationStore redis.LocationStoreInterface
    cacheStore *redis.CacheStore
    driverRepo repository.DriverRepository
}

type TripService struct {
    db *sql.DB
    tripRepo repository.TripRepository
    paymentService *PaymentService
}

// Supporting Services
type MatchingService struct {
    db *sql.DB
    locationStore redis.LocationStoreInterface
    lockStore redis.LockStoreInterface
}

type SurgeService struct {
    locationStore redis.LocationStoreInterface
    rideRepo repository.RideRepository
}
```

### Key Data Structures

```go
// Domain Entities
type Ride struct {
    ID               string
    RiderID          string
    PickupLat        float64
    PickupLng        float64
    DestinationLat   float64
    DestinationLng   float64
    Status           RideStatus
    AssignedDriverID *string
    SurgeMultiplier  float64
    PaymentMethod    PaymentMethod
}

type Driver struct {
    ID     string
    Name   string
    Phone  string
    Status DriverStatus
    Tier   DriverTier
}
```

## 6. Data Flow & Business Logic

### Ride Creation Flow
1. **User Request** → Validate coordinates & payment method
2. **Surge Calculation** → Check demand vs supply ratio
3. **Driver Matching** → Find nearby available drivers
4. **Assignment** → Lock and assign optimal driver
5. **Notification** → Alert driver of new ride

### Trip Lifecycle
```
REQUESTED → ASSIGNED → IN_TRIP → COMPLETED
     ↓         ↓         ↓         ↓
   CANCELLED  TIMEOUT   CANCELLED  RECEIPT
```

### Payment Flow
1. **Trip End** → Calculate final fare (base + surge)
2. **Payment Processing** → Mock PSP integration
3. **Receipt Generation** → Store transaction details
4. **Notification** → Send confirmation to user

## 7. Concurrency & Performance

### Distributed Locking
- **Redis-based locks** prevent double-booking drivers
- **TTL protection** against lock leaks
- **Atomic operations** for critical state changes

### Caching Strategy
- **Driver locations**: Redis GEO for proximity queries
- **Entity cache**: Hot driver/driver data
- **Idempotency**: 24-hour request deduplication

### Performance Optimizations
- **Database indexes** for status-based queries
- **Connection pooling** for DB and Redis
- **Async processing** for notifications

## 8. Error Handling & Edge Cases

### Error Categories
- **Validation Errors**: Invalid coordinates, missing fields
- **Business Logic**: Driver unavailable, insufficient balance
- **System Errors**: Database connection, Redis failure
- **Concurrency**: Race conditions, timeouts

### Key Edge Cases
- Driver accepts ride while processing another request
- Trip ends while payment is processing
- Network failures during state transitions
- Invalid GPS coordinates or payment amounts

## 9. Security Considerations

### Input Validation
- Coordinate bounds checking (-90 to 90 lat, -180 to 180 lng)
- Phone number format validation
- Amount limits and payment method validation

### API Security
- **Idempotency keys** prevent duplicate requests
- **Rate limiting** considerations (not implemented)
- **Input sanitization** via Gin framework

## 10. Testing Strategy

### Unit Tests
- Service layer business logic
- Repository data operations
- Mock external dependencies (Redis, DB)

### Integration Tests
- End-to-end API flows
- Database transactions
- External service integrations

### Performance Tests
- Concurrent ride creation
- Driver matching under load
- Cache hit/miss ratios

## 11. Deployment & Monitoring

### Docker Setup
```yaml
services:
  postgres: PostgreSQL database
  redis: Caching and real-time data
  app: Go backend service
  frontend: React UI (optional)
```

### Monitoring
- **New Relic APM**: Performance monitoring
- **Health checks**: Service availability
- **Error tracking**: Exception reporting

## 12. Future Enhancements

### Phase 2 Features
- Real payment gateway integration
- WebSocket real-time notifications
- ML-based driver matching
- Multi-region deployment

### Scalability Improvements
- Read replicas for reporting
- Horizontal service scaling
- Advanced caching strategies

---

## Implementation Checklist

- [x] Clean Architecture (Handler → Service → Repository)
- [x] Database schema with constraints
- [x] Redis integration for caching/locking
- [x] REST API with proper error handling
- [x] Unit tests for core logic
- [x] Docker containerization
- [x] Monitoring setup (New Relic)
- [x] Postman collection for testing

## Technology Stack
- **Language**: Go 1.24
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL
- **Cache**: Redis
- **Monitoring**: New Relic
- **Container**: Docker

This LLD provides sufficient detail for implementation while maintaining simplicity and focus on the core ride hailing functionality.