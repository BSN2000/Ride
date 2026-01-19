package service

import "errors"

var (
	// ErrNoDriverAvailable is returned when no driver can be matched.
	ErrNoDriverAvailable = errors.New("no driver available")

	// ErrRideNotInRequestedState is returned when trying to match a ride not in REQUESTED state.
	ErrRideNotInRequestedState = errors.New("ride not in requested state")

	// ErrInvalidRiderID is returned when rider ID is empty.
	ErrInvalidRiderID = errors.New("invalid rider id")

	// ErrInvalidRideID is returned when ride ID is empty.
	ErrInvalidRideID = errors.New("invalid ride id")

	// ErrInvalidPickupLocation is returned when pickup coordinates are invalid.
	ErrInvalidPickupLocation = errors.New("invalid pickup location")

	// ErrInvalidDestinationLocation is returned when destination coordinates are invalid.
	ErrInvalidDestinationLocation = errors.New("invalid destination location")

	// ErrInvalidDriverID is returned when driver ID is empty.
	ErrInvalidDriverID = errors.New("invalid driver id")

	// ErrInvalidTripID is returned when trip ID is empty.
	ErrInvalidTripID = errors.New("invalid trip id")

	// ErrDriverHasActiveTrip is returned when driver already has an active trip.
	ErrDriverHasActiveTrip = errors.New("driver already has an active trip")

	// ErrRideNotAssigned is returned when ride is not in ASSIGNED state.
	ErrRideNotAssigned = errors.New("ride not assigned")

	// ErrDriverNotAssignedToRide is returned when driver is not assigned to the ride.
	ErrDriverNotAssignedToRide = errors.New("driver not assigned to this ride")

	// ErrTripAlreadyEnded is returned when trying to end an already ended trip.
	ErrTripAlreadyEnded = errors.New("trip already ended")

	// ErrTripNotStarted is returned when trying to pause a trip that hasn't started.
	ErrTripNotStarted = errors.New("trip not started")

	// ErrTripNotPaused is returned when trying to resume a trip that isn't paused.
	ErrTripNotPaused = errors.New("trip not paused")

	// ErrInvalidPaymentAmount is returned when payment amount is invalid.
	ErrInvalidPaymentAmount = errors.New("invalid payment amount")

	// ErrInvalidPaymentID is returned when payment ID is empty.
	ErrInvalidPaymentID = errors.New("invalid payment id")

	// ErrInvalidLocation is returned when location coordinates are invalid.
	ErrInvalidLocation = errors.New("invalid location")

	// ErrRideAlreadyCancelled is returned when trying to cancel an already cancelled ride.
	ErrRideAlreadyCancelled = errors.New("ride already cancelled")

	// ErrRideCannotBeCancelled is returned when ride is in a state that cannot be cancelled.
	ErrRideCannotBeCancelled = errors.New("ride cannot be cancelled in current state")

	// ErrTripInProgress is returned when trying to cancel a ride with an active trip.
	ErrTripInProgress = errors.New("cannot cancel ride with trip in progress")

	// ErrInvalidPaymentMethod is returned when payment method is invalid.
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
)
