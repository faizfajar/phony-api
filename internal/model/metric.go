package model

import (
	"time"

	"github.com/google/uuid"
)

// APIMetric captures performance data for a specific request execution.
// It stores the duration and status to calculate p95/p99 latency later.
type APIMetric struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	EndpointID uuid.UUID `gorm:"type:uuid;index" json:"endpoint_id"`
	DurationMS int64     `json:"duration_ms"` // Total execution time in milliseconds
	StatusCode int       `json:"status_code"` // Response status returned to client
	CreatedAt  time.Time `json:"created_at"`  // Timestamp for time-series analysis
}
