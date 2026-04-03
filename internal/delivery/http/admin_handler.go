package http

import (
	"fmt"
	"net/http"

	"github.com/faizfajar/phony-api/internal/model"
	"github.com/faizfajar/phony-api/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	mockEngineService *service.MockEngineService
}

func NewAdminHandler(service *service.MockEngineService) *AdminHandler {
	return &AdminHandler{mockEngineService: service}
}

// CreateEndpoint handles the registration of new mock routes
func (h *AdminHandler) CreateEndpoint(c *gin.Context) {
	var input struct {
		Path         string           `json:"path" binding:"required"`
		Method       string           `json:"method" binding:"required"`
		VUsers       int              `json:"v_users"`
		Duration     int              `json:"duration"`
		ThresholdP95 int              `json:"threshold_p95"`
		Responses    []model.Response `json:"responses" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map input to model
	endpoint := &model.Endpoint{
		Path:         input.Path,
		Method:       input.Method,
		VUsers:       input.VUsers,
		Duration:     input.Duration,
		ThresholdP95: input.ThresholdP95,
		Responses:    input.Responses,
	}

	// Call service to persist (Insert only logic as requested)
	created, err := h.mockEngineService.RegisterNewEndpoint(endpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// GetK6Script serves the dynamically generated k6 script as a downloadable file
func (h *AdminHandler) GetK6Script(c *gin.Context) {
	id := c.Param("id")

	endpoint, err := h.mockEngineService.GetEndpointByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint configuration not found"})
		return
	}

	script := h.mockEngineService.GenerateK6Script(endpoint)

	// Set headers to trigger browser download
	fileName := fmt.Sprintf("stress_%s.js", endpoint.ID.String()[:8])
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Data(http.StatusOK, "application/javascript", []byte(script))
}
