package model

import (
	"time"

	"github.com/google/uuid"
)

type Endpoint struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Path      string     `gorm:"not null;index" json:"path"` // Index agar pencarian rute cepat
	Method    string     `gorm:"not null" json:"method"`     // GET, POST, dll
	ProjectID string     `gorm:"index" json:"project_id"`    // Untuk memisahkan antar user/project
	Responses []Response `gorm:"foreignKey:EndpointID" json:"responses"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type Response struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	EndpointID     uuid.UUID `gorm:"type:uuid" json:"endpoint_id"`
	Name           string    `json:"name"`
	TriggerParam   string    `json:"trigger_param"`
	ResponseStatus int       `json:"response_status"` // 200, 404, 500
	ResponseBody   string    `json:"response_body"`
	DelayMS        int       `json:"delay_ms"` // latency
}
