package service

import (
	"encoding/json"
	"time"

	"github.com/faizfajar/phony-api/internal/model"
	"github.com/faizfajar/phony-api/internal/repository"
	"github.com/google/uuid"
)

type EndpointService struct {
	endpointRepository repository.EndpointRepository
}

func NewEndpointService(endpointRepository repository.EndpointRepository) *EndpointService {
	return &EndpointService{endpointRepository: endpointRepository}
}

// CreateEndpoint handles the business logic for initializing a new mock configuration.
// It assigns a unique identifier and persists the endpoint along with its response scenarios.
func (service *EndpointService) CreateEndpoint(path string, method string, responses []model.Response) (*model.Endpoint, error) {
	newEndpoint := &model.Endpoint{
		ID:        uuid.New(),
		Path:      path,
		Method:    method,
		Responses: responses,
	}

	// Persisting the complete object graph to the database through the repository.
	error := service.endpointRepository.CreateEndpoint(newEndpoint)
	if error != nil {
		return nil, error
	}
	return newEndpoint, nil
}

// FindAllEndpoints retrieves a list of all registered mock configurations.
func (service *EndpointService) FindAllEndpoints() ([]model.Endpoint, error) {
	return service.endpointRepository.FindAllEndpoints()
}

// ExecuteMockMatching refined to handle default responses correctly as a fallback.
func (service *EndpointService) ExecuteMockMatching(path string, method string, requestQueryParameters map[string][]string) (*model.Response, error) {
	targetEndpoint, error := service.endpointRepository.FindEndpointByPathAndMethod(path, method)
	if error != nil {
		return nil, error
	}

	var defaultResponse *model.Response

	for _, responseScenario := range targetEndpoint.Responses {
		// Identify the default response (fallback) but do not return it yet.
		if responseScenario.TriggerParam == "" || responseScenario.TriggerParam == "{}" {
			defaultResponse = &responseScenario
			continue
		}

		// Try to match specific triggers.
		var triggerMap map[string]string
		unmarshalError := json.Unmarshal([]byte(responseScenario.TriggerParam), &triggerMap)
		if unmarshalError != nil {
			// If JSON is invalid (e.g. missing quotes), skip this specific scenario.
			continue
		}

		isMatched := true
		for key, expectedValue := range triggerMap {
			actualValue, exists := requestQueryParameters[key]
			if !exists || actualValue[0] != expectedValue {
				isMatched = false
				break
			}
		}

		if isMatched {
			return service.applyLatencyAndReturn(&responseScenario), nil
		}
	}

	// If no specific match is found, return the default response if it exists.
	if defaultResponse != nil {
		return service.applyLatencyAndReturn(defaultResponse), nil
	}

	return nil, nil
}

// applyLatencyAndReturn centralizes the logic for simulating artificial network delay.
func (service *EndpointService) applyLatencyAndReturn(response *model.Response) *model.Response {
	if response.DelayMS > 0 {
		time.Sleep(time.Duration(response.DelayMS) * time.Millisecond)
	}
	return response
}
