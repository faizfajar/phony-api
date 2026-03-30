package delivery

import (
	"net/http"

	"github.com/faizfajar/phony-api/internal/service"
	"github.com/gin-gonic/gin"
)

type MockHandler struct {
	mockEngineService *service.MockEngineService
}

func NewMockHandler(mockEngineService *service.MockEngineService) *MockHandler {
	return &MockHandler{mockEngineService: mockEngineService}
}

// ProcessMockRequest intercepts the incoming request and delegates the matching logic to the service layer.
func (handler *MockHandler) ProcessMockRequest(context *gin.Context) {
	requestPath := context.Param("proxypath")
	requestMethod := context.Request.Method

	// Extract all query parameters from the incoming URL
	queryParameters := context.Request.URL.Query()

	matchedResponse, error := handler.mockEngineService.ExecuteMockMatching(requestPath, requestMethod, queryParameters)
	if error != nil {
		context.JSON(http.StatusNotFound, gin.H{
			"error":   "Endpoint Configuration Not Found",
			"message": "The system could not locate a mock configuration for the requested path.",
		})
		return
	}

	if matchedResponse == nil {
		context.JSON(http.StatusNotFound, gin.H{
			"error":   "No Match Found",
			"message": "Endpoint exists, but the provided parameters do not match any configured response scenarios.",
		})
		return
	}

	// Output the mocked data with the appropriate status code.
	context.Data(matchedResponse.ResponseStatus, "application/json", []byte(matchedResponse.ResponseBody))
}
