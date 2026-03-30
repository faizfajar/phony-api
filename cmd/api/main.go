package main

import (
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

	// Register Routes
	admin := r.Group("/admin")
	{
		admin.POST("/endpoints", handler.CreateEndpoint)
		// admin.GET("/endpoints", handler.GetAllEndpoints)
	}

	// Run Server
	r.Run(":8080")
}
