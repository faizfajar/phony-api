package repository

import (
	"github.com/faizfajar/phony-api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EndpointRepository interface {
	CreateEndpoint(endpoint *model.Endpoint) error
	FindAllEndpoints() ([]model.Endpoint, error)
	FindEndpointByID(id uuid.UUID) (*model.Endpoint, error)
	// FindEndpointByPathAndMethod is used by the mock engine to locate specific configurations.
	FindEndpointByPathAndMethod(path string, method string) (*model.Endpoint, error)
	CreateMetricsBatch(metrics []model.APIMetric) error
}

type endpointRepository struct {
	database *gorm.DB
}

func NewEndpointRepository(database *gorm.DB) EndpointRepository {
	return &endpointRepository{database: database}
}

func (repository *endpointRepository) CreateEndpoint(endpoint *model.Endpoint) error {
	return repository.database.Create(endpoint).Error
}

func (repository *endpointRepository) FindAllEndpoints() ([]model.Endpoint, error) {
	var endpoints []model.Endpoint
	error := repository.database.Preload("Responses").Find(&endpoints).Error
	return endpoints, error
}

func (repository *endpointRepository) FindEndpointByID(id uuid.UUID) (*model.Endpoint, error) {
	var endpoint model.Endpoint
	error := repository.database.Preload("Responses").First(&endpoint, "id = ?", id).Error
	return &endpoint, error
}

func (repository *endpointRepository) FindEndpointByPathAndMethod(path string, method string) (*model.Endpoint, error) {
	var endpoint model.Endpoint
	// Using Preload to fetch all associated response scenarios for logic matching.
	error := repository.database.Preload("Responses").Where("path = ? AND method = ?", path, method).First(&endpoint).Error
	return &endpoint, error
}

func (repository *endpointRepository) CreateMetricsBatch(metrics []model.APIMetric) error {
	// GORM will automatically split the metrics into multiple batches if the number is very large
	// insert all at once in a single batch operation
	return repository.database.CreateInBatches(metrics, len(metrics)).Error
}
