package domain

// DriverStatus represents the current status of a driver.
type DriverStatus string

const (
	DriverStatusOnline  DriverStatus = "ONLINE"
	DriverStatusOffline DriverStatus = "OFFLINE"
	DriverStatusOnTrip  DriverStatus = "ON_TRIP"
)

// DriverTier represents the service tier of a driver.
type DriverTier string

const (
	DriverTierBasic   DriverTier = "BASIC"
	DriverTierPremium DriverTier = "PREMIUM"
)

// Driver represents a driver in the system.
type Driver struct {
	ID     string
	Name   string
	Phone  string
	Status DriverStatus
	Tier   DriverTier
}
