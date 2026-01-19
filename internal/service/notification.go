package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"ride/internal/domain"
)

// NotificationType represents the type of notification.
type NotificationType string

const (
	NotificationRideRequested   NotificationType = "RIDE_REQUESTED"
	NotificationDriverAssigned  NotificationType = "DRIVER_ASSIGNED"
	NotificationDriverArrived   NotificationType = "DRIVER_ARRIVED"
	NotificationTripStarted     NotificationType = "TRIP_STARTED"
	NotificationTripPaused      NotificationType = "TRIP_PAUSED"
	NotificationTripResumed     NotificationType = "TRIP_RESUMED"
	NotificationTripEnded       NotificationType = "TRIP_ENDED"
	NotificationPaymentSuccess  NotificationType = "PAYMENT_SUCCESS"
	NotificationPaymentFailed   NotificationType = "PAYMENT_FAILED"
	NotificationRideCancelled   NotificationType = "RIDE_CANCELLED"
	NotificationReceiptReady    NotificationType = "RECEIPT_READY"
)

// Notification represents a notification to be sent.
type Notification struct {
	ID          string
	Type        NotificationType
	RecipientID string           // User or Driver ID
	Title       string
	Message     string
	Data        map[string]interface{}
	CreatedAt   time.Time
}

// NotificationService handles notification delivery.
type NotificationService struct {
	// In a real system, this would have:
	// - Push notification client (FCM, APNS)
	// - SMS client (Twilio)
	// - Email client (SendGrid)
	// - WebSocket connections for real-time
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

// NotifyRideRequested notifies nearby drivers about a new ride request.
func (s *NotificationService) NotifyRideRequested(ctx context.Context, ride *domain.Ride, nearbyDriverIDs []string) error {
	for _, driverID := range nearbyDriverIDs {
		notification := Notification{
			Type:        NotificationRideRequested,
			RecipientID: driverID,
			Title:       "New Ride Request",
			Message:     fmt.Sprintf("New ride request near you. Pickup at (%.4f, %.4f)", ride.PickupLat, ride.PickupLng),
			Data: map[string]interface{}{
				"ride_id":    ride.ID,
				"pickup_lat": ride.PickupLat,
				"pickup_lng": ride.PickupLng,
				"surge":      ride.SurgeMultiplier,
			},
			CreatedAt: time.Now(),
		}
		s.send(ctx, notification)
	}
	return nil
}

// NotifyDriverAssigned notifies the rider that a driver has been assigned.
func (s *NotificationService) NotifyDriverAssigned(ctx context.Context, ride *domain.Ride, driver *domain.Driver) error {
	notification := Notification{
		Type:        NotificationDriverAssigned,
		RecipientID: ride.RiderID,
		Title:       "Driver Assigned",
		Message:     fmt.Sprintf("Driver %s has been assigned to your ride", driver.Name),
		Data: map[string]interface{}{
			"ride_id":     ride.ID,
			"driver_id":   driver.ID,
			"driver_name": driver.Name,
			"driver_tier": driver.Tier,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyTripStarted notifies the rider that the trip has started.
func (s *NotificationService) NotifyTripStarted(ctx context.Context, trip *domain.Trip, riderID string) error {
	notification := Notification{
		Type:        NotificationTripStarted,
		RecipientID: riderID,
		Title:       "Trip Started",
		Message:     "Your trip has started. Enjoy your ride!",
		Data: map[string]interface{}{
			"trip_id":    trip.ID,
			"started_at": trip.StartedAt,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyTripPaused notifies the rider that the trip has been paused.
func (s *NotificationService) NotifyTripPaused(ctx context.Context, trip *domain.Trip, riderID string) error {
	notification := Notification{
		Type:        NotificationTripPaused,
		RecipientID: riderID,
		Title:       "Trip Paused",
		Message:     "Your trip has been paused by the driver.",
		Data: map[string]interface{}{
			"trip_id":   trip.ID,
			"paused_at": trip.PausedAt,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyTripResumed notifies the rider that the trip has resumed.
func (s *NotificationService) NotifyTripResumed(ctx context.Context, trip *domain.Trip, riderID string) error {
	notification := Notification{
		Type:        NotificationTripResumed,
		RecipientID: riderID,
		Title:       "Trip Resumed",
		Message:     "Your trip has resumed.",
		Data: map[string]interface{}{
			"trip_id": trip.ID,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyTripEnded notifies the rider that the trip has ended.
func (s *NotificationService) NotifyTripEnded(ctx context.Context, trip *domain.Trip, riderID string, fare float64) error {
	notification := Notification{
		Type:        NotificationTripEnded,
		RecipientID: riderID,
		Title:       "Trip Completed",
		Message:     fmt.Sprintf("Your trip has ended. Total fare: $%.2f", fare),
		Data: map[string]interface{}{
			"trip_id":  trip.ID,
			"fare":     fare,
			"ended_at": trip.EndedAt,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyPaymentSuccess notifies the rider of successful payment.
func (s *NotificationService) NotifyPaymentSuccess(ctx context.Context, payment *domain.Payment, riderID string) error {
	notification := Notification{
		Type:        NotificationPaymentSuccess,
		RecipientID: riderID,
		Title:       "Payment Successful",
		Message:     fmt.Sprintf("Payment of $%.2f was successful", payment.Amount),
		Data: map[string]interface{}{
			"payment_id": payment.ID,
			"amount":     payment.Amount,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyPaymentFailed notifies the rider of failed payment.
func (s *NotificationService) NotifyPaymentFailed(ctx context.Context, payment *domain.Payment, riderID string) error {
	notification := Notification{
		Type:        NotificationPaymentFailed,
		RecipientID: riderID,
		Title:       "Payment Failed",
		Message:     fmt.Sprintf("Payment of $%.2f failed. Please try again.", payment.Amount),
		Data: map[string]interface{}{
			"payment_id": payment.ID,
			"amount":     payment.Amount,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyRideCancelled notifies parties about ride cancellation.
func (s *NotificationService) NotifyRideCancelled(ctx context.Context, ride *domain.Ride, cancelledBy string, reason string) error {
	// Notify the other party
	var recipientID string
	var message string

	if cancelledBy == ride.RiderID {
		recipientID = ride.AssignedDriverID
		message = "The rider has cancelled the ride"
	} else {
		recipientID = ride.RiderID
		message = "The driver has cancelled the ride"
	}

	if recipientID == "" {
		return nil // No one to notify
	}

	notification := Notification{
		Type:        NotificationRideCancelled,
		RecipientID: recipientID,
		Title:       "Ride Cancelled",
		Message:     message,
		Data: map[string]interface{}{
			"ride_id":      ride.ID,
			"cancelled_by": cancelledBy,
			"reason":       reason,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// NotifyReceiptReady notifies the rider that the receipt is ready.
func (s *NotificationService) NotifyReceiptReady(ctx context.Context, receipt *domain.Receipt) error {
	notification := Notification{
		Type:        NotificationReceiptReady,
		RecipientID: receipt.RiderID,
		Title:       "Receipt Ready",
		Message:     fmt.Sprintf("Your receipt for $%.2f is ready", receipt.TotalFare),
		Data: map[string]interface{}{
			"receipt_id": receipt.ID,
			"trip_id":    receipt.TripID,
			"total_fare": receipt.TotalFare,
		},
		CreatedAt: time.Now(),
	}
	return s.send(ctx, notification)
}

// send delivers a notification (mock implementation).
func (s *NotificationService) send(ctx context.Context, notification Notification) error {
	// In a real implementation, this would:
	// 1. Store notification in database
	// 2. Send push notification via FCM/APNS
	// 3. Send SMS if enabled
	// 4. Send email if enabled
	// 5. Broadcast via WebSocket for real-time updates

	log.Printf("[NOTIFICATION] Type=%s, Recipient=%s, Title=%s, Message=%s",
		notification.Type, notification.RecipientID, notification.Title, notification.Message)

	return nil
}
