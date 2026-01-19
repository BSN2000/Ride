Ride Hailing System – GoComet SDE-2 Assignment

IMPORTANT: This README is the single source of truth for building, validating, and reviewing this system.

All code, structure, tests, and design decisions must strictly follow this document and the original assignment requirements. Any deviation must be explicitly justified.

⸻

1. Purpose of This Document

This README defines:
	•	Exact functional requirements
	•	Non-functional requirements (scale, latency, consistency)
	•	Locked technology choices
	•	Architecture and data flow
	•	Database schema expectations
	•	Redis usage rules
	•	Concurrency and consistency guarantees
	•	API contracts
	•	Coding standards (Uber Go)
	•	Project structure
	•	Testing expectations
	•	Explicitly out-of-scope items

⸻

2. Problem Statement (From Assignment)

Design and implement a multi-tenant, multi-region ride-hailing system (Uber/Ola-like) that supports:
	•	Real-time driver location updates (1–2 updates/sec per driver)
	•	Rider ride requests
	•	Driver–rider matching with p95 < 1 second
	•	Trip lifecycle management
	•	Payments via an external PSP (mocked)
	•	Notifications for key ride events (minimal)

Scale Requirements (Reasoned, Not Simulated)
	•	~100k drivers
	•	~10k ride requests/min
	•	~200k driver location updates/sec

⸻

3. Locked Technology Stack (Non-Negotiable)
	•	Language: Go (Golang)
	•	Database: PostgreSQL (SQL only)
	•	Cache / In-Memory: Redis
	•	HTTP Framework: Gin (or net/http)
	•	Monitoring: New Relic (APM)
	•	Frontend: Minimal React UI (polling-based)

⸻

4. How to Run

### Prerequisites
- Go 1.24 or later
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (optional)

### Quick Start with Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/BSN2000/Ride.git
cd Ride

# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f

# Access the application
# Frontend: http://localhost:3000
# API: http://localhost:8080
# Health check: http://localhost:8080/health
```

### Manual Setup

#### 1. Install Dependencies
```bash
# Install Go 1.24+
# Install PostgreSQL 15+
# Install Redis 7+
```

#### 2. Database Setup
```bash
# Start PostgreSQL and create database
createdb ride_hailing

# Run schema
psql -d ride_hailing -f scripts/schema.sql
```

#### 3. Environment Variables
Create `.env` file or set environment variables:

```bash
# Server
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=10s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ride_hailing
DB_SSLMODE=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=""
REDIS_DB=0

# New Relic (Optional)
NEW_RELIC_ENABLED=true
NEW_RELIC_APP_NAME="ride-hailing-service"
NEW_RELIC_LICENSE_KEY="your-license-key"
```

#### 4. Run the Application
```bash
# Install dependencies
go mod tidy

# Run the server
go run cmd/server/main.go

# Or build and run
go build -o ride-server cmd/server/main.go
./ride-server
```

#### 5. Test the API
```bash
# Health check
curl http://localhost:8080/health

# Import Postman collection for full API testing
# File: postman_collection_comprehensive.json
```

### Testing

```bash
# Run unit tests
go test ./internal/tests/...

# Run specific test
go test -v ./internal/tests/ -run TestRideCreation

# Run with coverage
go test -cover ./internal/tests/...
```

### API Documentation

- **Postman Collection**: `postman_collection_comprehensive.json`
- **API Endpoints**: All endpoints use POST method only
- **Base URL**: `http://localhost:8080/v1`

### Architecture Overview

```
Frontend (React) → API Server (Gin) → Services → Repositories → PostgreSQL/Redis
                                        ↓
                                   Notifications (in-process)
```

### Key Features Demonstrated

- ✅ Real-time driver matching with Redis GEO
- ✅ Distributed locking for concurrency control
- ✅ Idempotent requests with Redis caching
- ✅ Surge pricing based on demand
- ✅ Trip lifecycle management (start/pause/resume/end)
- ✅ Automatic receipt generation
- ✅ Mock payment processing
- ✅ Comprehensive error handling

### Monitoring

- **New Relic APM**: Application performance monitoring
- **Health Checks**: `/health` endpoint
- **Logs**: Structured logging throughout the application

⸻

5. System Design Principles

8.1 Clean Architecture

Layered dependency flow:

Handler  →  Service  →  Repository  →  DB / Redis

Rules:
	•	Handlers never access DB or Redis directly
	•	Services contain business logic only
	•	Repositories are thin persistence layers
	•	Redis is accessed only via dedicated packages

8.2 SOLID Principles
	•	S: Single responsibility per file/module
	•	O: Services depend on interfaces, not implementations
	•	L: Repositories are swappable and mockable
	•	I: Small, focused interfaces
	•	D: Dependency injection via constructors

5.3 Minimalism Rule
	•	No unnecessary abstractions
	•	No premature optimizations
	•	No unused tables or services
	•	Prefer synchronous flows unless explicitly required

⸻

20. High-Level Architecture

8.1 Core Components
	1.	Ride Service
	2.	Driver Service
	3.	Matching Service
	4.	Trip Service
	5.	Payment Service (mocked)
	6.	Notification Layer (minimal, in-process)

8.2 Stateless Services

All HTTP services must be stateless and horizontally scalable.

⸻

20. Data Storage Strategy

8.1 PostgreSQL (SQL)

Used strictly for:
	•	Drivers (status, tier)
	•	Rides
	•	Trips
	•	Payments

8.2 Redis

Used strictly for:
	•	Driver real-time locations (GEO index)
	•	Distributed locks (driver assignment)
	•	Short-lived cached state

❌ Driver location must never be written to SQL

⸻

20. Database Schema Expectations

8.1 Drivers
	•	id (PK)
	•	status: ONLINE | OFFLINE | ON_TRIP
	•	tier: BASIC | PREMIUM

8.2 Rides
	•	id (PK)
	•	rider_id
	•	pickup_lat, pickup_lng
	•	destination_lat, destination_lng
	•	status: REQUESTED | ASSIGNED | CANCELLED
	•	assigned_driver_id (nullable)

8.3 Trips
	•	id (PK)
	•	ride_id (FK)
	•	driver_id (FK)
	•	status: STARTED | PAUSED | ENDED
	•	fare
	•	started_at, ended_at

Constraint: A driver can have only one active trip at a time (enforced via DB constraint/index).

8.4 Payments
	•	id (PK)
	•	trip_id (FK)
	•	amount
	•	status: PENDING | SUCCESS | FAILED
	•	idempotency_key (unique)

⸻

20. API Contracts (Minimum Required)

8.1 Create Ride

POST /v1/rides

Responsibilities:
	•	Validate request
	•	Create ride in REQUESTED state
	•	Trigger matching logic
	•	Support idempotency

⸻

8.2 Get Ride Status

GET /v1/rides/{id}

⸻

8.3 Driver Location Update

POST /v1/drivers/{id}/location

Responsibilities:
	•	Update Redis GEO index
	•	No SQL writes

⸻

8.4 Driver Accept Ride

POST /v1/drivers/{id}/accept

Responsibilities:
	•	Atomic DB transaction
	•	Update ride and driver state
	•	Create trip

⸻

8.5 End Trip

POST /v1/trips/{id}/end

Responsibilities:
	•	End trip
	•	Calculate fare
	•	Trigger payment (mocked)

⸻

20. Driver–Rider Matching Logic (CRITICAL)

Matching must:
	1.	Query Redis GEO for nearby drivers
	2.	Filter by tier and availability
	3.	Acquire per-driver Redis lock
	4.	Assign driver to ride
	5.	Persist assignment atomically

Redis Locking

Key format:

lock:driver:{driver_id}

Rules:
	•	SET NX EX
	•	TTL required
	•	Released on failure or timeout

⸻

20. Consistency & Concurrency Guarantees

The system must guarantee:
	•	No driver is assigned to multiple rides
	•	No ride has multiple drivers
	•	Payment retries are safe (idempotent)

Techniques used:
	•	SQL transactions
	•	Redis distributed locks
	•	Unique DB constraints

⸻

20. Performance & Scalability Requirements
	•	Driver lookup via Redis only
	•	Indexed SQL queries only
	•	No full table scans
	•	No global locks
	•	Writes are region-local

⸻

20. Monitoring (New Relic)

Required metrics:
	•	API p95 latency
	•	DB slow queries
	•	Error rate

Screenshots of New Relic dashboards are mandatory in final submission.

⸻

20. Frontend Requirements (Minimal)
	•	Simple React UI
	•	Create ride
	•	Display ride status and assigned driver
	•	Poll backend every 2 seconds for updates

❌ No WebSockets required
❌ No advanced UI/UX work expected

⸻

20. Project Structure (MANDATORY)

internal/
├── app/
├── config/
├── domain/
├── repository/
├── service/
├── handler/
├── redis/
├── middleware/
└── tests/


⸻

20. Coding Standards (Uber Go)

Mandatory rules:
	•	context.Context passed everywhere
	•	Early returns
	•	No global mutable state
	•	Explicit error handling
	•	Small, readable functions
	•	Clear and descriptive naming

⸻

20. Testing Expectations

Required
	•	Matching logic tests
	•	Concurrency tests (driver assignment)
	•	Ride creation and lifecycle tests

Not Required
	•	HTTP router tests
	•	Redis client library tests

Testing focus is on correctness, safety, and concurrency, not coverage percentage.

⸻

20. Explicitly Out of Scope
	•	Real payment gateways
	•	Maps or routing APIs
	•	ML-based matching
	•	Kubernetes / deployment infra
	•	Authentication & authorization

⸻

20. Final Deliverables Checklist
	•	Backend code (Go)
	•	Frontend code (React)
	•	New Relic dashboard screenshots
	•	This README
	•	Architecture explanation
	•	Demo walkthrough

⸻

20. Final Note

This system is evaluated on:
	•	Correctness
	•	Concurrency safety
	•	Scalability reasoning
	•	Code quality
	•	Engineering judgment

Not on feature completeness.

Any deviation from this document must be explicitly justified.