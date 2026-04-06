package repository

import (
	"github.com/faizfajar/phony-api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EndpointRepository interface {
	GetAll() ([]model.Endpoint, error)
	CreateEndpoint(endpoint *model.Endpoint) error
	Update(id uuid.UUID, endpoint *model.Endpoint) error
	FindAllEndpoints() ([]model.Endpoint, error)
	FindEndpointByID(id uuid.UUID) (*model.Endpoint, error)
	// FindEndpointByPathAndMethod is used by the mock engine to locate specific configurations.
	FindEndpointByPathAndMethod(path string, method string) (*model.Endpoint, error)
	CreateMetricsBatch(metrics []model.APIMetric) error
	GetMetricSummary(id uuid.UUID) (map[string]interface{}, error)
	// GetMetricSummary(id uuid.UUID) (map[string]interface{}, error)
}

type endpointRepository struct {
	database *gorm.DB
}

func NewEndpointRepository(database *gorm.DB) EndpointRepository {
	return &endpointRepository{database: database}
}

func (repository *endpointRepository) GetAll() ([]model.Endpoint, error) {
	var endpoints []model.Endpoint
	// Mengambil semua data dan sekaligus "Preload" relasi responses-nya
	err := repository.database.Preload("Responses").Find(&endpoints).Error
	return endpoints, err
}

func (repository *endpointRepository) CreateEndpoint(endpoint *model.Endpoint) error {
	return repository.database.Create(endpoint).Error
}

func (repository *endpointRepository) Update(id uuid.UUID, endpoint *model.Endpoint) error {
	return repository.database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("endpoint_id = ?", id).Delete(&model.Response{}).Error; err != nil {
			return err
		}

		updateData := map[string]interface{}{
			"path":          endpoint.Path,
			"method":        endpoint.Method,
			"v_users":       endpoint.VUsers,
			"duration":      endpoint.Duration,
			"threshold_p95": endpoint.ThresholdP95,
		}

		if err := tx.Model(&model.Endpoint{}).Where("id = ?", id).Updates(updateData).Error; err != nil {
			return err
		}

		for i := range endpoint.Responses {
			endpoint.Responses[i].EndpointID = id
			if err := tx.Create(&endpoint.Responses[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
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

func (repository *endpointRepository) GetMetricSummary(endpointID uuid.UUID) (map[string]interface{}, error) {

	var stats struct {
		AvgLatency float64
		MaxLatency int64
		TotalHits  int64
		P95        float64
		P99        float64
	}

	err := repository.database.Model(&model.APIMetric{}).Select(`
            AVG(duration_ms) as avg_latency, 
            MAX(duration_ms) as max_latency, 
            COUNT(*) as total_hits,
            PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95,
            PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99
        `).
		Where("endpoint_id = ?", endpointID).
		Scan(&stats).Error

	return map[string]interface{}{
		"average_latency_ms": stats.AvgLatency,
		"max_latency_ms":     stats.MaxLatency,
		"p95_latency_ms":     stats.P95,
		"p99_latency_ms":     stats.P99,
		"total_requests":     stats.TotalHits,
	}, err
}
