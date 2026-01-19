# Low Level Design (LLD) – Ride Hailing System

**Author:** SDE-2 Engineer  
**Version:** 1.0  
**Date:** January 2026  
**Source of Truth:** `read.md` (README)

---

## 1. Document Overview

### 1.1 Purpose of This LLD

This document provides a **code-level design** for a ride-hailing system built as part of an SDE-2 assignment. It explains:

- How each component is structured and why
- What data flows between components
- How concurrency and consistency are guaranteed
- What trade-offs were made and why

This is **not** a high-level architecture document. Every decision here maps directly to code.

### 1.2 Scope of the System

| In Scope | Out of Scope |
|----------|--------------|
| Rider ride requests | Real payment gateways |
| Driver registration & location | Maps/routing APIs |
| Driver-rider matching | ML-based matching |
| Trip lifecycle (start → end) | Authentication/authorization |
| Payment processing (mocked) | Kubernetes/deployment |
| Basic monitoring (New Relic) | WebSocket real-time updates |

### 1.3 Problems This Design Solves

1. **Real-time driver discovery** – Finding nearby drivers in < 1 second
2. **Concurrent assignment safety** – Preventing double-booking of drivers
3. **Data consistency** – Ensuring ride/trip/payment states are always valid
4. **Idempotency** – Safe retries for network failures
5. **Horizontal scalability** – Stateless services that can scale out

### 1.4 What is Explicitly NOT Built

| Feature | Reason |
|---------|--------|
| WebSocket notifications | Polling is sufficient for demo; simpler to implement |
| Async job queues | Synchronous flow meets latency requirements |
| Driver location in SQL | Would cause write bottleneck at scale |
| Multi-region support | Out of scope; design is region-local |
| Surge pricing | Not required per README |

---

## 2. System Component Breakdown

### 2.1 Component Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Handler   │────▶│   Service   │────▶│ Repository  │
│  (HTTP/API) │     │  (Business) │     │    (Data)   │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │    Redis    │
                    │  (Location) │
                    └─────────────┘
```

### 2.2 Ride Service

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Create rides, track ride status |
| **Owns** | Ride entity lifecycle |
| **Does NOT Own** | Driver matching logic, trip management |
| **Key Interactions** | Calls MatchingService after ride creation |

**Why separate from Matching?** Single Responsibility Principle. Ride creation is a distinct operation from finding an available driver.

### 2.3 Driver Service

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Driver registration, location updates |
| **Owns** | Driver entity, location in Redis |
| **Does NOT Own** | Ride assignment, trip lifecycle |
| **Key Interactions** | Updates Redis GEO index on location change |

**Why location updates are separate from status?** Location updates are high-frequency (1-2/sec per driver). They must hit Redis only, not SQL.

### 2.4 Matching Service

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Find and assign available driver to ride |
| **Owns** | Matching algorithm, driver lock acquisition |
| **Does NOT Own** | Ride creation, trip creation |
| **Key Interactions** | Reads Redis GEO, acquires Redis locks, writes to SQL |

**Why this is the most critical component?** It coordinates between Redis (location/locks) and SQL (assignment) with atomic guarantees.

### 2.5 Trip Service

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Start and end trips, calculate fare |
| **Owns** | Trip entity lifecycle |
| **Does NOT Own** | Ride matching, payment processing logic |
| **Key Interactions** | Calls PaymentService on trip end |

**Why trip is separate from ride?** A ride is a *request*; a trip is the *actual journey*. They have different lifecycles and states.

### 2.6 Payment Service

| Aspect | Description |
|--------|-------------|
| **Responsibility** | Process payments with idempotency |
| **Owns** | Payment entity, PSP integration (mocked) |
| **Does NOT Own** | Trip lifecycle, fare calculation |
| **Key Interactions** | Called by TripService after trip ends |

**Why idempotency is critical here?** Network failures during payment can cause retries. Without idempotency, this would charge riders multiple times.

---

## 3. Project Structure (Code Level)

```
internal/
├── app/           # Application wiring (router, dependencies)
├── config/        # Environment configuration
├── domain/        # Entity definitions (Driver, Ride, Trip, Payment)
├── handler/       # HTTP handlers (parse request → call service → return response)
├── service/       # Business logic (validation, orchestration)
├── repository/    # Data access interfaces and implementations
│   └── postgres/  # PostgreSQL implementations
├── redis/         # Redis operations (location, locks)
├── middleware/    # HTTP middleware (CORS, idempotency, New Relic)
└── tests/         # Unit and integration tests
```

### 3.1 Folder Responsibilities

| Folder | Purpose | What Must NOT Live Here |
|--------|---------|-------------------------|
| `domain/` | Pure data structures with typed constants | Business logic, DB/JSON tags |
| `handler/` | HTTP parsing, response formatting | Business logic, direct DB access |
| `service/` | Business rules, orchestration | SQL queries, HTTP parsing |
| `repository/` | Data persistence operations | Business logic, HTTP concerns |
| `redis/` | Redis-specific operations | SQL queries, business logic |
| `middleware/` | Cross-cutting concerns | Business logic |
| `tests/` | Test files only | Production code |

### 3.2 Dependency Flow Rule

```
handler → service → repository → database
              ↓
            redis
```

**Strict Rule:** Upper layers never bypass lower layers. Handlers never access repositories directly.

---

## 4. Domain Models (Entities)

### 4.1 Driver

```go
type Driver struct {
    ID     string        // UUID
    Name   string        // Display name
    Phone  string        // Unique phone number
    Status DriverStatus  // ONLINE | OFFLINE | ON_TRIP
    Tier   DriverTier    // BASIC | PREMIUM
}
```

**State Transitions:**

```
OFFLINE ──(location update)──▶ ONLINE ──(ride assigned)──▶ ON_TRIP ──(trip ended)──▶ ONLINE
                                  ▲                                                      │
                                  └──────────────────────────────────────────────────────┘
```

**Invariants:**
- A driver in `ON_TRIP` cannot be assigned another ride
- Status must be one of the three defined values (DB CHECK constraint)

### 4.2 Ride

```go
type Ride struct {
    ID               string      // UUID
    RiderID          string      // Who requested the ride
    PickupLat        float64     // Pickup latitude
    PickupLng        float64     // Pickup longitude
    DestinationLat   float64     // Destination latitude
    DestinationLng   float64     // Destination longitude
    Status           RideStatus  // REQUESTED | ASSIGNED | CANCELLED
    AssignedDriverID string      // Nullable: set when matched
}
```

**State Transitions:**

```
REQUESTED ──(driver matched)──▶ ASSIGNED
    │
    └──(timeout/cancel)──▶ CANCELLED
```

**Invariants:**
- A ride can have at most ONE assigned driver
- Once ASSIGNED, cannot go back to REQUESTED
- Coordinates must be valid (-90≤lat≤90, -180≤lng≤180)

### 4.3 Trip

```go
type Trip struct {
    ID        string      // UUID
    RideID    string      // FK to rides table
    DriverID  string      // FK to drivers table
    Status    TripStatus  // STARTED | PAUSED | ENDED
    Fare      float64     // Calculated on trip end
    StartedAt time.Time   // When trip began
    EndedAt   time.Time   // When trip ended (nullable until ENDED)
}
```

**State Transitions:**

```
STARTED ──(pause)──▶ PAUSED ──(resume)──▶ STARTED
    │                    │
    └──(end)─────────────┴──(end)──▶ ENDED
```

**Invariants:**
- **One active trip per driver** (DB unique partial index enforces this)
- Fare is 0 until trip ends
- `ended_at` is NULL until status is ENDED

### 4.4 Payment

```go
type Payment struct {
    ID             string        // UUID
    TripID         string        // FK to trips table
    Amount         float64       // Fare amount
    Status         PaymentStatus // PENDING | SUCCESS | FAILED
    IdempotencyKey string        // Unique: "payment:{trip_id}"
}
```

**State Transitions:**

```
PENDING ──(PSP success)──▶ SUCCESS
    │
    └──(PSP failure)──▶ FAILED
```

**Invariants:**
- `IdempotencyKey` is unique (prevents duplicate payments)
- One payment per trip
- Amount must be > 0

---

## 5. Database Design (SQL)

### 5.1 Table: `drivers`

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| id | VARCHAR(36) | PRIMARY KEY | UUID identifier |
| name | VARCHAR(100) | NOT NULL | Display name |
| phone | VARCHAR(20) | UNIQUE, NOT NULL | Login identifier |
| status | VARCHAR(20) | CHECK IN ('ONLINE','OFFLINE','ON_TRIP') | Current state |
| tier | VARCHAR(20) | CHECK IN ('BASIC','PREMIUM') | Service level |
| created_at | TIMESTAMP | DEFAULT NOW() | Audit |

**Why no location column?** Location updates happen 200k/sec across all drivers. Storing in SQL would create a write bottleneck.

### 5.2 Table: `rides`

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| id | VARCHAR(36) | PRIMARY KEY | UUID |
| rider_id | VARCHAR(36) | NOT NULL | Who requested |
| pickup_lat | DOUBLE PRECISION | NOT NULL | Pickup location |
| pickup_lng | DOUBLE PRECISION | NOT NULL | Pickup location |
| destination_lat | DOUBLE PRECISION | NOT NULL | Destination |
| destination_lng | DOUBLE PRECISION | NOT NULL | Destination |
| status | VARCHAR(20) | CHECK IN ('REQUESTED','ASSIGNED','CANCELLED') | State |
| assigned_driver_id | VARCHAR(36) | NULLABLE | Matched driver |
| created_at | TIMESTAMP | DEFAULT NOW() | Audit |

**Indexes:**
- `idx_rides_status` – Filter by status for matching
- `idx_rides_assigned_driver` – Find rides by driver

### 5.3 Table: `trips`

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| id | VARCHAR(36) | PRIMARY KEY | UUID |
| ride_id | VARCHAR(36) | FK → rides(id) | Source ride |
| driver_id | VARCHAR(36) | FK → drivers(id) | Assigned driver |
| status | VARCHAR(20) | CHECK IN ('STARTED','PAUSED','ENDED') | State |
| fare | DOUBLE PRECISION | DEFAULT 0 | Calculated fare |
| started_at | TIMESTAMP | NOT NULL | Trip start |
| ended_at | TIMESTAMP | NULLABLE | Trip end |

**Critical Constraint:**
```sql
CREATE UNIQUE INDEX idx_trips_active_driver 
ON trips (driver_id) 
WHERE status != 'ENDED';
```

**Why this partial unique index?** Prevents a driver from having multiple active trips. This is a **database-level guarantee** that works even if application logic fails.

### 5.4 Table: `payments`

| Column | Type | Constraints | Purpose |
|--------|------|-------------|---------|
| id | VARCHAR(36) | PRIMARY KEY | UUID |
| trip_id | VARCHAR(36) | FK → trips(id) | Source trip |
| amount | DOUBLE PRECISION | NOT NULL | Payment amount |
| status | VARCHAR(20) | CHECK IN ('PENDING','SUCCESS','FAILED') | State |
| idempotency_key | VARCHAR(255) | UNIQUE, NOT NULL | Duplicate prevention |
| created_at | TIMESTAMP | DEFAULT NOW() | Audit |

**Why idempotency_key is UNIQUE?** If the same payment request is retried, the database will reject the duplicate.

### 5.5 How DB Constraints Protect Against Race Conditions

| Race Condition | DB Protection |
|----------------|---------------|
| Two rides assigned to same driver | `idx_trips_active_driver` unique partial index |
| Duplicate payment for same trip | `idempotency_key` unique constraint |
| Invalid driver status | CHECK constraint on status column |
| Invalid ride status | CHECK constraint on status column |

---

## 6. Redis Design

### 6.1 Driver Location Storage (GEO)

**Key:** `drivers:locations`  
**Type:** Redis GEO (sorted set with geospatial indexing)  
**Operations:**
- `GEOADD` – Update driver location
- `GEORADIUS` – Find drivers within radius (sorted by distance)
- `ZREM` – Remove driver from index

**Why Redis GEO?**
- O(log N) for inserts and lookups
- Native distance sorting
- 200k updates/sec is trivial for Redis

**Failure Handling:**
- If Redis is unavailable, location update fails (driver appears offline)
- This is acceptable: riders get slightly stale results temporarily
- SQL is NOT used as fallback – that would defeat the purpose

### 6.2 Driver Assignment Locks

**Key Format:** `lock:driver:{driver_id}`  
**Type:** String with SET NX EX  
**TTL:** 10 seconds (configurable)

**Operations:**
```
SET lock:driver:abc123 "1" NX EX 10
```
- `NX` – Only set if not exists (atomic)
- `EX 10` – Expires in 10 seconds (prevents deadlock)

**Why Redis locks?**
- Matching happens before DB write
- Multiple matching attempts for different rides could pick the same driver
- Lock prevents concurrent assignment attempts

**Lock Lifecycle:**
```
1. Match finds driver
2. Acquire lock (SET NX EX)
3. If failed → skip driver, try next
4. If success → begin DB transaction
5. Commit transaction
6. Lock expires via TTL (no explicit release needed on success)
7. On failure → explicitly release lock
```

### 6.3 Why Redis is NOT Source of Truth

| Data | Source of Truth | Why |
|------|-----------------|-----|
| Driver status | PostgreSQL | Durable, transactional |
| Ride assignment | PostgreSQL | Must survive restarts |
| Trip state | PostgreSQL | Auditable, consistent |
| Driver location | Redis | Ephemeral, high-frequency |
| Assignment locks | Redis | Temporary coordination |

**What happens if Redis is unavailable?**
- Location updates fail → drivers appear offline
- Lock acquisition fails → matching fails
- System gracefully degrades but doesn't corrupt data

---

## 7. API Design & Flow

### 7.1 Create Ride

**Endpoint:** `POST /v1/rides`

**Request:**
```json
{
    "rider_id": "user-123",
    "pickup_lat": 12.9716,
    "pickup_lng": 77.5946,
    "destination_lat": 12.2958,
    "destination_lng": 76.6394
}
```

**Headers:**
- `Idempotency-Key: <unique-key>` (optional)

**Response:**
```json
{
    "id": "ride-456",
    "rider_id": "user-123",
    "status": "ASSIGNED",
    "assigned_driver_id": "driver-789",
    "driver_assigned": true
}
```

**Validation Rules:**
- `rider_id` required
- Latitude: -90 ≤ lat ≤ 90
- Longitude: -180 ≤ lng ≤ 180

**Service Flow:**
```
1. Validate request
2. Check idempotency (if key provided)
3. Create ride in REQUESTED state
4. Persist to DB
5. Call MatchingService.Match()
6. If driver found → return ASSIGNED
7. If no driver → return REQUESTED
8. Cache response for idempotency
```

**Failure Cases:**
| Error | HTTP Code | Response |
|-------|-----------|----------|
| Invalid rider ID | 400 | `{"error": "invalid rider id"}` |
| Invalid coordinates | 400 | `{"error": "invalid pickup location"}` |
| DB error | 500 | `{"error": "internal error"}` |

### 7.2 Get Ride Status

**Endpoint:** `GET /v1/rides/{id}`

**Response:**
```json
{
    "id": "ride-456",
    "rider_id": "user-123",
    "status": "ASSIGNED",
    "assigned_driver_id": "driver-789"
}
```

**Service Flow:**
```
1. Validate ride ID (non-empty)
2. Query DB by ride ID
3. Return ride or 404
```

### 7.3 Driver Location Update

**Endpoint:** `POST /v1/drivers/{id}/location`

**Request:**
```json
{
    "lat": 12.9716,
    "lng": 77.5946
}
```

**Response:** `200 OK` (empty body)

**Service Flow:**
```
1. Validate driver ID (non-empty)
2. Validate coordinates (range check)
3. GEOADD to Redis
4. Update driver status to ONLINE (if not already)
```

**Critical Design Decision:**
- **NO SQL WRITE** for location
- Driver status update to ONLINE is fire-and-forget (ignore errors)

### 7.4 Driver Accept Ride

**Endpoint:** `POST /v1/drivers/{id}/accept`

**Request:**
```json
{
    "ride_id": "ride-456"
}
```

**Response:**
```json
{
    "trip_id": "trip-789",
    "ride_id": "ride-456",
    "driver_id": "driver-123",
    "status": "STARTED",
    "started_at": "2026-01-16T12:00:00Z"
}
```

**Service Flow:**
```
1. Validate driver ID and ride ID
2. Check driver has no active trip
3. Verify ride is ASSIGNED to this driver
4. Create trip in STARTED state
5. Return trip details
```

**Failure Cases:**
| Error | HTTP Code | Response |
|-------|-----------|----------|
| Driver has active trip | 400 | `{"error": "driver has active trip"}` |
| Wrong driver | 400 | `{"error": "driver not assigned to this ride"}` |
| Ride not assigned | 400 | `{"error": "ride not assigned"}` |

### 7.5 End Trip

**Endpoint:** `POST /v1/trips/{id}/end`

**Response:**
```json
{
    "trip_id": "trip-789",
    "status": "ENDED",
    "fare": 25.50,
    "payment": {
        "id": "payment-123",
        "status": "SUCCESS"
    }
}
```

**Service Flow:**
```
1. Validate trip ID
2. Get trip from DB
3. Check trip is not already ENDED
4. Calculate fare (base + per-minute)
5. BEGIN TRANSACTION
   - Update trip status to ENDED
   - Update driver status to ONLINE
6. COMMIT TRANSACTION
7. Trigger payment (outside transaction)
8. Return response
```

**Why payment is outside transaction?**
- Payment can fail without invalidating trip end
- Trip end is the critical operation
- Failed payments can be retried

---

## 8. Driver–Rider Matching Logic (CRITICAL)

### 8.1 Step-by-Step Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. GEORADIUS query (Redis)                                      │
│    - Get drivers within 5km of pickup                           │
│    - Sorted by distance (closest first)                         │
└─────────────────────────┬───────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. For each driver:                                             │
│    a. Fetch driver from DB                                      │
│    b. Check status == ONLINE                                    │
│    c. Check tier matches (if specified)                         │
│    d. If not eligible → skip, try next                          │
└─────────────────────────┬───────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Acquire Redis lock: SET lock:driver:{id} NX EX 10            │
│    - If lock fails → skip driver, try next                      │
│    - If lock acquired → proceed                                 │
└─────────────────────────┬───────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. BEGIN SQL TRANSACTION                                        │
│    - Update ride: status = ASSIGNED, assigned_driver_id = X     │
│    - Update driver: status = ON_TRIP                            │
│    - COMMIT                                                     │
└─────────────────────────┬───────────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Return match result                                          │
│    - Lock expires via TTL (no explicit release on success)      │
│    - On failure: explicitly release lock                        │
└─────────────────────────────────────────────────────────────────┘
```

### 8.2 Why This Order?

| Step | Why This Position |
|------|-------------------|
| Redis GEO first | Fastest filter; reduces candidates |
| DB status check | Cheap SELECT; filters further |
| Lock acquisition | Prevents concurrent assignment |
| Transaction last | Only executed if all checks pass |

### 8.3 Race Condition Prevention

**Scenario:** Two rides R1 and R2 both want driver D1

```
R1: GEORADIUS → finds D1
R2: GEORADIUS → finds D1
R1: Acquires lock for D1 ✓
R2: Tries to acquire lock for D1 ✗ (already held)
R1: Transaction commits, D1 assigned to R1
R2: Skips D1, tries next driver
```

**Scenario:** Same driver accepts multiple rides

This is prevented at two levels:
1. **Application:** Check `ride.AssignedDriverID == driver.ID`
2. **Database:** Unique partial index `idx_trips_active_driver`

---

## 9. Concurrency & Consistency Guarantees

### 9.1 Known Race Conditions

| Race Condition | Prevention Mechanism |
|----------------|---------------------|
| Two drivers claim same ride | Ride already has `assigned_driver_id`; second fails validation |
| Same driver assigned to multiple rides | Redis lock + DB unique partial index |
| Duplicate payments | `idempotency_key` unique constraint |
| Double trip end | Check `status != ENDED` before update |

### 9.2 Why Both Redis Locks AND DB Constraints?

| Mechanism | Role |
|-----------|------|
| Redis Lock | **Optimistic prevention** – Fast, prevents wasted transactions |
| DB Constraint | **Guaranteed prevention** – Works even if Redis fails |

Redis locks reduce contention. DB constraints are the safety net.

### 9.3 Detailed Examples

**Example 1: Two Drivers Accept Same Ride**
```
Driver A: AcceptRide(ride-1)
Driver B: AcceptRide(ride-1)

Both check: ride.AssignedDriverID == "driver-A"
Driver A: Match ✓ → Creates trip
Driver B: Mismatch ✗ → Returns error "driver not assigned to this ride"
```

**Example 2: Same Driver Accepts Multiple Rides Concurrently**
```
Request 1: StartTrip(driver-1, ride-1)
Request 2: StartTrip(driver-1, ride-2)

Both attempt:
INSERT INTO trips (driver_id, ...) VALUES ('driver-1', ...)
WHERE status != 'ENDED'

Request 1: Success (no existing active trip)
Request 2: FAILS (unique index violation)
```

**Example 3: Payment Retry**
```
Request 1: ProcessPayment(trip-1, $25)
Network timeout, client retries
Request 2: ProcessPayment(trip-1, $25)

Request 1: INSERT payment with idempotency_key = "payment:trip-1" ✓
Request 2: Checks idempotency_key exists → Returns existing payment ✓
```

---

## 10. Error Handling Strategy

### 10.1 Error Propagation Rules

```
Repository → Service → Handler → HTTP Response
   (raw)      (wrap)    (map)      (clean)
```

| Layer | Responsibility |
|-------|----------------|
| Repository | Return raw errors (sql.ErrNoRows, etc.) |
| Service | Wrap with semantic errors (ErrNotFound, ErrInvalidInput) |
| Handler | Map to HTTP codes (400, 404, 500) |

### 10.2 Semantic Errors

```go
var (
    ErrNotFound           = errors.New("entity not found")
    ErrInvalidRiderID     = errors.New("invalid rider id")
    ErrNoDriverAvailable  = errors.New("no driver available")
    ErrDriverHasActiveTrip = errors.New("driver has active trip")
)
```

### 10.3 Retry vs Fail Fast

| Scenario | Strategy | Reason |
|----------|----------|--------|
| DB connection error | Fail fast | Connection pool handles retries |
| Redis timeout | Fail fast | Caller can retry with backoff |
| Payment PSP error | Mark as FAILED | Trip end succeeded; payment can retry |
| Invalid input | Fail fast | Client error, no retry will help |

### 10.4 Idempotency Prevents Corruption

Without idempotency:
```
Client sends: CreateRide
Server processes: Creates ride, matching succeeds
Response lost in network
Client retries: CreateRide
Server creates ANOTHER ride
```

With idempotency:
```
Client sends: CreateRide + Idempotency-Key: abc123
Server processes: Creates ride, caches response with key abc123
Response lost in network
Client retries: CreateRide + Idempotency-Key: abc123
Server finds cached response for abc123
Server returns same response (no new ride)
```

---

## 11. Testing Strategy

### 11.1 What Is Tested

| Test Category | Coverage |
|---------------|----------|
| **Ride Creation** | Input validation, state persistence, field integrity |
| **Driver Location** | Redis-only writes, coordinate validation, ONLINE status |
| **Matching Logic** | Filtering, lock acquisition, transaction safety |
| **Concurrency** | Lock contention, race conditions, state transitions |
| **Trip Lifecycle** | State transitions, fare calculation, driver status |
| **Payment** | Idempotency, PSP failure handling |

### 11.2 What Is NOT Tested

| Not Tested | Reason |
|------------|--------|
| HTTP routing | Framework responsibility (Gin is battle-tested) |
| Redis client internals | Library is mature; we test our usage |
| Full integration (E2E) | Out of scope; manual API testing covers this |

### 11.3 Why These Choices

**Focus on correctness over coverage.** A 100% coverage metric means nothing if concurrency bugs exist.

**Key test patterns:**
- Table-driven tests for validation
- Goroutines + WaitGroups for concurrency
- Mock repositories for isolation
- Real timing tests for TTL behavior

---

## 12. Monitoring & Observability

### 12.1 Metrics Tracked (New Relic)

| Metric | Why It Matters |
|--------|----------------|
| API p95 latency | Matching must be < 1 second |
| Error rate | Spike indicates bugs or downstream failures |
| DB query time | Slow queries indicate missing indexes |
| Redis operation time | GEO lookups should be < 10ms |

### 12.2 How New Relic Is Used

```go
// Gin middleware
router.Use(nrgin.Middleware(nrApp))

// All HTTP transactions automatically traced
// Database calls instrumented via driver
```

### 12.3 Production Alerts (Recommended)

| Alert | Threshold | Action |
|-------|-----------|--------|
| p95 latency > 500ms | Sustained for 5 min | Check DB queries |
| Error rate > 1% | Sustained for 2 min | Check logs |
| Redis connection failures | Any | Check Redis cluster |

---

## 13. Design Trade-Offs & Justifications

### 13.1 Why Polling Instead of WebSockets?

| Factor | Polling | WebSockets |
|--------|---------|------------|
| Complexity | Low | High |
| Server resources | Higher | Lower |
| Client complexity | Trivial | Requires reconnection logic |
| Scale (for demo) | Sufficient | Overkill |

**Decision:** 2-second polling is acceptable latency for a demo. WebSockets would add complexity without proportional benefit.

### 13.2 Why Redis + SQL Combination?

| Data | Redis | SQL |
|------|-------|-----|
| Driver location | ✓ (fast writes) | ✗ (bottleneck) |
| Driver status | ✗ | ✓ (durable) |
| Ride/Trip state | ✗ | ✓ (transactional) |
| Locks | ✓ (distributed) | ✗ (no native lock) |

**Decision:** Each store is used for what it does best. Redis for speed. SQL for durability.

### 13.3 Why No Async Queues?

| Approach | Pros | Cons |
|----------|------|------|
| Synchronous | Simple, predictable, debuggable | Slower if heavy processing |
| Async Queue | Decoupled, resilient | Complex, eventual consistency |

**Decision:** Matching takes < 100ms. Synchronous is simpler and meets latency requirements.

### 13.4 Why Minimal Frontend?

| Factor | Decision |
|--------|----------|
| Evaluation criteria | Backend engineering |
| Time constraints | Limited |
| Complexity budget | Backend deserves it |

**Decision:** Plain HTML/CSS/JS. No React build complexity. Works for demo.

---

## 14. Limitations & Future Improvements

### 14.1 Known Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| Single region | No geo-redundancy | Design is region-local |
| No auth | No user isolation | Out of scope |
| Mocked PSP | No real payments | Easily replaceable interface |
| Basic fare calculation | Time-based only | Could add distance-based |

### 14.2 Scaling to 10x

| Component | Current | At 10x Scale |
|-----------|---------|--------------|
| Redis | Single instance | Cluster with sharding |
| PostgreSQL | Single instance | Read replicas + connection pooling |
| Matching | Per-request | Batch matching with priority queue |
| Location updates | Direct writes | Buffer + batch writes |

### 14.3 SDE-3 Level Improvements

1. **Async matching** – Push to queue, process in worker
2. **Surge pricing** – Supply/demand ratio per zone
3. **Driver routing** – Optimize pickup based on traffic
4. **Event sourcing** – Full audit trail of state changes
5. **Multi-region** – Replicated data with conflict resolution

---

## 15. Final Summary

### 15.1 Design Guarantees

| Guarantee | Mechanism |
|-----------|-----------|
| No double-booking of drivers | Redis lock + DB unique index |
| No duplicate payments | Idempotency key unique constraint |
| Consistent state transitions | SQL transactions |
| Safe retries | Idempotency middleware |

### 15.2 Why This Design Is Safe

1. **Defense in depth** – Redis locks prevent most contention; DB constraints catch the rest
2. **Fail-safe locks** – TTL ensures no permanent deadlocks
3. **Idempotent operations** – Network failures don't corrupt data
4. **Explicit validation** – Invalid input rejected at entry point

### 15.3 Why This Design Is Production-Viable

1. **Battle-tested patterns** – Lock + transaction is industry standard
2. **Horizontal scalability** – Stateless services, partitioned data
3. **Observable** – New Relic captures latency, errors, throughput
4. **Simple to debug** – Synchronous flow, explicit errors
5. **Minimal dependencies** – PostgreSQL, Redis, Go – no exotic tech

---

**Document End**

*This LLD is a living document. Any changes to the system should update this document first.*
