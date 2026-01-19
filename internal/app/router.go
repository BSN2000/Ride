package app

import (
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/go-redis/v9"

	"ride/internal/handler"
	"ride/internal/middleware"
)

// RouterDeps contains all dependencies needed for the router.
type RouterDeps struct {
	RideHandler    *handler.RideHandler
	DriverHandler  *handler.DriverHandler
	TripHandler    *handler.TripHandler
	UserHandler    *handler.UserHandler
	PaymentHandler *handler.PaymentHandler
	RedisClient    *redis.Client
	NewRelicApp    *newrelic.Application
}

// NewRouter creates a new Gin router with all routes registered.
func NewRouter(deps RouterDeps) *gin.Engine {
	router := gin.New()

	// Global middleware.
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(middleware.CORSMiddleware())

	// Add New Relic middleware if enabled.
	if deps.NewRelicApp != nil {
		router.Use(nrgin.Middleware(deps.NewRelicApp))
	}

	router.Use(middleware.IdempotencyMiddleware(deps.RedisClient))

	// Health check.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes.
	v1 := router.Group("/v1")
	{
		// User routes.
		users := v1.Group("/users")
		{
			users.POST("/register", deps.UserHandler.Register)
			users.GET("", deps.UserHandler.GetAll)
		}

		// Ride routes.
		rides := v1.Group("/rides")
		{
			rides.POST("", deps.RideHandler.CreateRide)
			rides.GET("", deps.RideHandler.GetAll)
			rides.GET("/:id", deps.RideHandler.GetRide)
			rides.POST("/:id/cancel", deps.RideHandler.CancelRide)
		}

		// Driver routes.
		drivers := v1.Group("/drivers")
		{
			drivers.POST("/register", deps.DriverHandler.Register)
			drivers.GET("", deps.DriverHandler.GetAll)
			drivers.POST("/:id/location", deps.DriverHandler.UpdateLocation)
			drivers.POST("/:id/accept", deps.DriverHandler.AcceptRide)
		}

		// Trip routes.
		trips := v1.Group("/trips")
		{
			trips.GET("", deps.TripHandler.GetAll)
			trips.GET("/:id", deps.TripHandler.GetTrip)
			trips.POST("/:id/pause", deps.TripHandler.PauseTrip)
			trips.POST("/:id/resume", deps.TripHandler.ResumeTrip)
			trips.POST("/:id/end", deps.TripHandler.EndTrip)
		}

		// Payment routes.
		payments := v1.Group("/payments")
		{
			payments.POST("", deps.PaymentHandler.ProcessPayment)
			payments.GET("/:id", deps.PaymentHandler.GetPayment)
		}
	}

	return router
}
