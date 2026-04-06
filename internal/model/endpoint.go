package model

import (
	"time"

	"github.com/google/uuid"
)

type UpdateEndpointRequest struct {
	Path         string            `json:"path" binding:"required"`
	Method       string            `json:"method" binding:"required"`
	VUsers       int               `json:"v_users"`
	Duration     int               `json:"duration"`
	ThresholdP95 int               `json:"threshold_p95"`
	Responses    []ResponseRequest `json:"responses" binding:"required"`
}

type ResponseRequest struct {
	Name           string `json:"name"`
	TriggerParam   string `json:"trigger_param"`
	TriggerHeader  string `json:"trigger_header"` // Mendukung matching header
	TriggerBody    string `json:"trigger_body"`   // Mendukung matching JSON body
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `json:"response_body"`
	DelayMS        int    `json:"delay_ms"`
}
type Endpoint struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	Path         string      `gorm:"not null;index" json:"path"`
	Method       string      `gorm:"not null" json:"method"`
	ProjectID    string      `gorm:"index" json:"project_id"`
	VUsers       int         `gorm:"default:10" json:"v_users"`        // Target virtual users for k6
	Duration     int         `gorm:"default:30" json:"duration"`       // Test duration in seconds
	ThresholdP95 int         `gorm:"default:500" json:"threshold_p95"` // Target SLA in ms
	Responses    []Response  `gorm:"foreignKey:EndpointID" json:"responses"`
	Metrics      []APIMetric `gorm:"foreignKey:EndpointID" json:"metrics"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type Response struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	EndpointID     uuid.UUID `gorm:"type:uuid" json:"endpoint_id"`
	Name           string    `json:"name"`
	TriggerParam   string    `json:"trigger_param"`
	TriggerHeader  string    `json:"trigger_header"` // New: Match specific request headers
	TriggerBody    string    `json:"trigger_body"`   // New: Match specific JSON body content
	ResponseStatus int       `json:"response_status"`
	ResponseBody   string    `json:"response_body"`
	DelayMS        int       `json:"delay_ms"`
}
