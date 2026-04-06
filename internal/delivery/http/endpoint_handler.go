package http

import (
	"net/http"

	"github.com/faizfajar/phony-api/internal/model"
	"github.com/faizfajar/phony-api/internal/service"
	response "github.com/faizfajar/phony-api/pkg/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EndpointHandler struct {
	service *service.EndpointService
}

func NewEndpointHandler(s *service.EndpointService) *EndpointHandler {
	return &EndpointHandler{service: s}
}

func (h *EndpointHandler) ListEndpoints(c *gin.Context) {
	// Pastikan di service layer lo sudah ada method GetAll atau sejenisnya
	endpoints, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar endpoint: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, endpoints)
}

// CreateEndpoint processes incoming HTTP requests to define a new mock route.
// It validates the input payload and transforms it into the domain model.
func (h *EndpointHandler) CreateEndpoint(c *gin.Context) {
	var input struct {
		Path      string `json:"path" binding:"required"`
		Method    string `json:"method" binding:"required"`
		Responses []struct {
			Name           string `json:"name"`
			TriggerParam   string `json:"trigger_param"`
			ResponseStatus int    `json:"response_status"`
			ResponseBody   string `json:"response_body"`
			DelayMS        int    `json:"delay_ms"`
		} `json:"responses"`
	}

	// Validate JSON binding against the expected structure.
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map input DTO to internal domain models.
	var responses []model.Response
	for _, r := range input.Responses {
		responses = append(responses, model.Response{
			Name:           r.Name,
			TriggerParam:   r.TriggerParam,
			ResponseStatus: r.ResponseStatus,
			ResponseBody:   r.ResponseBody,
			DelayMS:        r.DelayMS,
		})
	}

	// Execute creation logic through the service layer.
	mock, err := h.service.CreateEndpoint(input.Path, input.Method, responses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mock)
}

func (h *EndpointHandler) UpdateEndpoint(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var input model.UpdateEndpointRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Passing DTO ke Service
	if err := h.service.UpdateEndpoint(id, input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update endpoint"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Endpoint updated successfully"})
}

func (h *EndpointHandler) GetStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, 400, "Invalid UUID format", err.Error())
		return
	}

	stats, err := h.service.GetEndpointStats(id)
	if err != nil {
		response.Error(c, 500, "Failed to fetch stats", err.Error())
		return
	}

	response.Success(c, 200, "Stats retrieved successfully", stats)
}
