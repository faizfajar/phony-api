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

	"github.com/faizfajar/phony-api/internal/delivery"
	"github.com/faizfajar/phony-api/internal/repository"
	"github.com/faizfajar/phony-api/internal/service"
	"github.com/faizfajar/phony-api/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Inisialisasi DB
	database.Connect()

	// Setup Clean Architecture Layers (Wiring)
	repo := repository.NewEndpointRepository(database.DB)
	svc := service.NewEndpointService(repo)
	handler := delivery.NewEndpointHandler(svc)

	mockSvc := service.NewMockEngineService(repo)
	mockHandler := delivery.NewMockHandler(mockSvc)

	// Setup Gin Router
	r := gin.Default()
	r.Any("/mocks/*proxypath", mockHandler.ProcessMockRequest)

	admin := r.Group("/admin")
	{
		admin.POST("/endpoints", handler.CreateEndpoint)
	}

	// Konfigurasi HTTP Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Channel untuk menangkap sinyal terminasi (Ctrl+C / Kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Jalankan Server di Goroutine (agar tidak blocking)
	go func() {
		fmt.Println("[SERVER] Phony-API is starting on :8080...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[ERROR] Listen: %v\n", err)
		}
	}()

	// Tunggu Sinyal Berhenti
	<-quit
	fmt.Println("\n[SYSTEM] Shutdown initiated...")

	// Graceful Shutdown HTTP Server (timeout 5 detik)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[ERROR] Server forced to shutdown: %v", err)
	}

	mockSvc.Shutdown()

	fmt.Println("[SYSTEM] Phony-API exited safely.")
}
