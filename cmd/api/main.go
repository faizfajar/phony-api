package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faizfajar/phony-api/internal/config"
	delivery "github.com/faizfajar/phony-api/internal/delivery/http"
	"github.com/faizfajar/phony-api/internal/repository"
	"github.com/faizfajar/phony-api/internal/service"
	"github.com/faizfajar/phony-api/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {

	config.LoadConfig()
	// Initialize Database connection
	database.Connect(config.AppConfig.DSN)

	// Setup Repository
	repo := repository.NewEndpointRepository(database.DB)

	// Setup Services
	// endpointSvc handles legacy stats/management
	endpointSvc := service.NewEndpointService(repo)
	// mockSvc handles the core mocking engine and k6 generation
	mockSvc := service.NewMockEngineService(repo)

	// Setup Handlers from the updated http delivery package
	mockHandler := delivery.NewMockHandler(mockSvc)
	adminHandler := delivery.NewAdminHandler(mockSvc)
	endpointHandler := delivery.NewEndpointHandler(endpointSvc)

	// Setup Gin Router with
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Account-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Custom Error Middleware to handle internal Gin errors
	r.Use(func(ctx *gin.Context) {
		ctx.Next()
		if len(ctx.Errors) > 0 {
			ctx.JSON(ctx.Writer.Status(), gin.H{
				"errors": ctx.Errors.Errors(),
			})
		}
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP", "time": time.Now().Format(time.RFC3339)})
	})

	// Core Mocking Engine Route
	// Using /mocks/ prefix for the proxy requests
	r.Any("/mocks/*proxypath", mockHandler.ProcessMockRequest)

	// Admin Management and Benchmarking Routes
	admin := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		config.AppConfig.AdminUser: config.AppConfig.AdminPassword,
	}))
	{
		// Endpoint Management
		admin.POST("/endpoints", adminHandler.CreateEndpoint)

		admin.GET("/endpoints/:id/stats", endpointHandler.GetStats)

		admin.GET("/endpoints/:id/k6", adminHandler.GetK6Script)
	}

	// HTTP Server Configuration
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Channel to listen for termination signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start Server in a Goroutine to prevent blocking
	go func() {
		fmt.Println("[SERVER] Phony-API is starting on :8080...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[ERROR] Listen: %v\n", err)
		}
	}()

	// Wait for Signal to stop
	<-quit
	fmt.Println("\n[SYSTEM] Shutdown initiated...")

	// Graceful Shutdown with 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[ERROR] Server forced to shutdown: %v", err)
	}

	// Cleanup service resources
	mockSvc.Shutdown()

	fmt.Println("[SYSTEM] Phony-API exited safely.")
}
