package http

import (
	"io"
	"net/http"

	"github.com/faizfajar/phony-api/internal/service"
	"github.com/gin-gonic/gin"
)

type MockHandler struct {
	mockEngineService *service.MockEngineService
}

func NewMockHandler(service *service.MockEngineService) *MockHandler {
	return &MockHandler{mockEngineService: service}
}

// ProcessMockRequest handles incoming mock calls and performs complex matching
func (h *MockHandler) ProcessMockRequest(c *gin.Context) {
	path := c.Param("proxypath")
	method := c.Request.Method
	queryParams := c.Request.URL.Query()
	headers := c.Request.Header

	var bodyString string
	if c.Request.Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		bodyString = string(bodyBytes)
	}

	matchedResponse, err := h.mockEngineService.ExecuteMockMatching(path, method, queryParams, headers, bodyString)
	if err != nil || matchedResponse == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Match Not Found",
			"message": "No mock configuration matches this request.",
		})
		return
	}

	c.Data(matchedResponse.ResponseStatus, "application/json", []byte(matchedResponse.ResponseBody))
}
