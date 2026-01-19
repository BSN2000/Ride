-- Create database schema for ride-hailing system

-- Users table (riders)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Drivers table
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

-- Rides table
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
    payment_method VARCHAR(20) NOT NULL DEFAULT 'CASH',
    cancelled_at TIMESTAMP,
    cancel_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT rides_status_check CHECK (status IN ('REQUESTED', 'ASSIGNED', 'IN_TRIP', 'COMPLETED', 'CANCELLED')),
    CONSTRAINT rides_surge_check CHECK (surge_multiplier >= 1.0 AND surge_multiplier <= 5.0),
    CONSTRAINT rides_payment_method_check CHECK (payment_method IN ('CASH', 'CARD', 'WALLET', 'UPI'))
);

-- Trips table
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(36) PRIMARY KEY,
    ride_id VARCHAR(36) NOT NULL REFERENCES rides(id),
    driver_id VARCHAR(36) NOT NULL REFERENCES drivers(id),
    status VARCHAR(20) NOT NULL DEFAULT 'STARTED',
    fare DOUBLE PRECISION DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    paused_at TIMESTAMP,
    total_paused_seconds INTEGER DEFAULT 0,
    CONSTRAINT trips_status_check CHECK (status IN ('STARTED', 'PAUSED', 'ENDED'))
);

-- Receipts table
CREATE TABLE IF NOT EXISTS receipts (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL REFERENCES trips(id),
    ride_id VARCHAR(36) NOT NULL REFERENCES rides(id),
    driver_id VARCHAR(36) NOT NULL REFERENCES drivers(id),
    rider_id VARCHAR(36) NOT NULL,
    pickup_lat DOUBLE PRECISION NOT NULL,
    pickup_lng DOUBLE PRECISION NOT NULL,
    destination_lat DOUBLE PRECISION NOT NULL,
    destination_lng DOUBLE PRECISION NOT NULL,
    base_fare DOUBLE PRECISION NOT NULL,
    surge_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    surge_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_fare DOUBLE PRECISION NOT NULL,
    payment_method VARCHAR(20) NOT NULL,
    payment_status VARCHAR(20) NOT NULL,
    duration_seconds INTEGER NOT NULL,
    distance_km DOUBLE PRECISION NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Constraint: A driver can have only ONE active trip at a time
CREATE UNIQUE INDEX IF NOT EXISTS idx_trips_active_driver 
ON trips (driver_id) 
WHERE status != 'ENDED';

-- Payments table
CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(36) PRIMARY KEY,
    trip_id VARCHAR(36) NOT NULL REFERENCES trips(id),
    amount DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT payments_status_check CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED'))
);

-- ============================================
-- OPTIMIZED INDEXES FOR HIGH-PERFORMANCE QUERIES
-- ============================================

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- Drivers indexes
CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);
CREATE INDEX IF NOT EXISTS idx_drivers_tier ON drivers(tier);
-- Composite index for matching: status + tier (covers most matching queries)
CREATE INDEX IF NOT EXISTS idx_drivers_status_tier ON drivers(status, tier);
-- Partial index for ONLINE drivers only (smaller, faster for matching)
CREATE INDEX IF NOT EXISTS idx_drivers_online ON drivers(id) WHERE status = 'ONLINE';
CREATE INDEX IF NOT EXISTS idx_drivers_online_tier ON drivers(id, tier) WHERE status = 'ONLINE';

-- Rides indexes
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);
CREATE INDEX IF NOT EXISTS idx_rides_assigned_driver ON rides(assigned_driver_id);
CREATE INDEX IF NOT EXISTS idx_rides_rider ON rides(rider_id);
CREATE INDEX IF NOT EXISTS idx_rides_created_at ON rides(created_at DESC);
-- Composite index for active rides
CREATE INDEX IF NOT EXISTS idx_rides_status_created ON rides(status, created_at DESC);
-- Partial index for REQUESTED rides only (for surge calculation)
CREATE INDEX IF NOT EXISTS idx_rides_requested ON rides(id, created_at) WHERE status = 'REQUESTED';
-- Covering index for ride status queries (avoids table lookup)
CREATE INDEX IF NOT EXISTS idx_rides_status_covering ON rides(id, status, assigned_driver_id, surge_multiplier);

-- Trips indexes
CREATE INDEX IF NOT EXISTS idx_trips_driver ON trips(driver_id);
CREATE INDEX IF NOT EXISTS idx_trips_ride ON trips(ride_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips(status);
-- Composite index for active trip lookup
CREATE INDEX IF NOT EXISTS idx_trips_driver_status ON trips(driver_id, status);
-- Partial index for active trips only
CREATE INDEX IF NOT EXISTS idx_trips_active ON trips(driver_id) WHERE status != 'ENDED';

-- Payments indexes
CREATE INDEX IF NOT EXISTS idx_payments_trip ON payments(trip_id);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

-- Receipts indexes
CREATE INDEX IF NOT EXISTS idx_receipts_trip ON receipts(trip_id);
CREATE INDEX IF NOT EXISTS idx_receipts_rider ON receipts(rider_id);
CREATE INDEX IF NOT EXISTS idx_receipts_driver ON receipts(driver_id);
CREATE INDEX IF NOT EXISTS idx_receipts_created ON receipts(created_at DESC);

-- ============================================
-- VERSION COLUMN FOR OPTIMISTIC LOCKING
-- ============================================
-- Add version column to rides for optimistic locking (run as ALTER if table exists)
-- ALTER TABLE rides ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1;
-- ALTER TABLE drivers ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1;