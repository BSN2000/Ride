package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/redis/go-redis/v9"

	"ride/internal/app"
	"ride/internal/config"
	"ride/internal/handler"
	internalRedis "ride/internal/redis"
	"ride/internal/repository/postgres"
	"ride/internal/service"
)

func main() {
	// Load configuration.
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize New Relic FIRST (before database so we can instrument DB).
	var nrApp *newrelic.Application
	var err error
	if cfg.NewRelic.Enabled && cfg.NewRelic.LicenseKey != "" {
		nrApp, err = newrelic.NewApplication(
			newrelic.ConfigAppName(cfg.NewRelic.AppName),
			newrelic.ConfigLicense(cfg.NewRelic.LicenseKey),
			newrelic.ConfigDistributedTracerEnabled(true),
			newrelic.ConfigAppLogForwardingEnabled(true),
		)
		if err != nil {
			log.Printf("failed to initialize New Relic: %v", err)
		} else {
			log.Printf("New Relic enabled: app=%s (with DB instrumentation)", cfg.NewRelic.AppName)
		}
	}

	// Initialize database with New Relic instrumentation.
	db, err := app.NewDatabase(ctx, cfg.Database, nrApp)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Initialize Redis with New Relic instrumentation.
	redisClient, err := app.NewRedisClient(ctx, cfg.Redis, nrApp)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Wire dependencies.
	server := wireServer(db, redisClient, nrApp, cfg)

	// Start server in goroutine.
	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// wireServer wires all dependencies and returns the HTTP server.
func wireServer(db *sql.DB, redisClient *redis.Client, nrApp *newrelic.Application, cfg *config.Config) *http.Server {
	// Initialize Redis stores.
	locationStore := internalRedis.NewLocationStore(redisClient)
	lockStore := internalRedis.NewLockStore(redisClient)
	cacheStore := internalRedis.NewCacheStore(redisClient)

	// Initialize repositories.
	userRepo := postgres.NewUserRepository(db)
	driverRepo := postgres.NewDriverRepository(db)
	rideRepo := postgres.NewRideRepository(db)
	tripRepo := postgres.NewTripRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)

	// Initialize services.
	notificationService := service.NewNotificationService()
	receiptService := service.NewReceiptService(notificationService)
	matchingService := service.NewMatchingService(db, locationStore, lockStore, cacheStore, driverRepo, rideRepo)
	surgeService := service.NewSurgeService(locationStore, rideRepo)
	rideService := service.NewRideService(rideRepo, matchingService, surgeService, notificationService)
	driverService := service.NewDriverService(locationStore, cacheStore, driverRepo)
	psp := service.NewMockPSP()
	paymentService := service.NewPaymentService(paymentRepo, psp)
	tripService := service.NewTripService(db, tripRepo, rideRepo, driverRepo, paymentService, notificationService, receiptService)

	// Initialize handlers.
	userHandler := handler.NewUserHandler(userRepo)
	rideHandler := handler.NewRideHandler(rideService, rideRepo)
	driverHandler := handler.NewDriverHandler(driverService, tripService, driverRepo)
	tripHandler := handler.NewTripHandler(tripService)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// Create router.
	router := app.NewRouter(app.RouterDeps{
		UserHandler:    userHandler,
		RideHandler:    rideHandler,
		DriverHandler:  driverHandler,
		TripHandler:    tripHandler,
		PaymentHandler: paymentHandler,
		RedisClient:    redisClient,
		NewRelicApp:    nrApp,
	})

	// Create HTTP server.
	return &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}
}
