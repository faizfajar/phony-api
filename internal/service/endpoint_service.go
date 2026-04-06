package service

import (
	"encoding/json"
	"fmt"
	"strings"
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

func (s *EndpointService) GetAll() ([]model.Endpoint, error) {
	return s.endpointRepository.GetAll() // Manggil repo untuk SELECT * FROM endpoints
}

// CreateEndpoint handles the business logic for initializing a new mock configuration.
// It assigns a unique identifier and persists the endpoint along with its response scenarios.
func (s *EndpointService) CreateEndpoint(path string, method string, responses []model.Response) (*model.Endpoint, error) {
	newID := uuid.New()

	// Tambahkan log ini buat mastiin ID-nya beneran ganti tiap kali klik Send
	fmt.Printf("[DEBUG] Creating new endpoint with ID: %s\n", newID.String())

	newEndpoint := &model.Endpoint{
		ID:        newID,
		Path:      path,
		Method:    method,
		Responses: responses,
	}

	err := s.endpointRepository.CreateEndpoint(newEndpoint)
	if err != nil {
		return nil, err
	}

	return newEndpoint, nil
}

func (service *EndpointService) UpdateEndpoint(id uuid.UUID, input model.UpdateEndpointRequest) error {

	var responses []model.Response
	for _, r := range input.Responses {
		responses = append(responses, model.Response{
			Name:           r.Name,
			TriggerParam:   r.TriggerParam,
			TriggerHeader:  r.TriggerHeader,
			TriggerBody:    r.TriggerBody,
			ResponseStatus: r.ResponseStatus,
			ResponseBody:   r.ResponseBody,
			DelayMS:        r.DelayMS,
		})
	}

	endpoint := &model.Endpoint{
		Path:         input.Path,
		Method:       input.Method,
		VUsers:       input.VUsers,
		Duration:     input.Duration,
		ThresholdP95: input.ThresholdP95,
		Responses:    responses,
	}

	return service.endpointRepository.Update(id, endpoint)
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
			// If JSON is invalid (e.missing quotes), skip this specific scenario.
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

func (service *EndpointService) GetEndpointStats(id uuid.UUID) (map[string]interface{}, error) {

	stats, err := service.endpointRepository.GetMetricSummary(id)
	if err != nil {
		return nil, err
	}

	stats["generated_at"] = time.Now()

	return stats, nil
}

// GenerateK6Script produces a ready-to-run JavaScript file for k6 benchmarking
func (s *EndpointService) GenerateK6Script(endpoint *model.Endpoint) string {
	script := fmt.Sprintf(`
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    vus: %d,
    duration: '%ds',
    thresholds: {
        http_req_duration: ['p(95)<%d'], // Benchmark will fail if p95 exceeds this
    },
};

export default function () {
    const url = 'http://your-vps-ip:8080%s';
    
    // Execute request to the mock server
    let res = http.get(url);
    
    check(res, {
        'status is 200': (r) => r.status === 200,
    });
    
    sleep(1);
}`, endpoint.VUsers, endpoint.Duration, endpoint.ThresholdP95, endpoint.Path)

	return script
}

// MatchComplexRequest handles matching logic for Body and Headers (Puzzle Piece #1)
func (s *EndpointService) MatchComplexRequest(endpoint *model.Endpoint, reqHeaders map[string][]string, reqBody string) (*model.Response, error) {
	var defaultResponse *model.Response

	for _, res := range endpoint.Responses {
		// Simpan fallback (default) jika trigger kosong
		if res.TriggerParam == "{}" || res.TriggerParam == "" {
			defaultResponse = &res
			continue
		}

		// Match Headers (Jika ada kriteria di DB)
		if res.TriggerHeader != "" && res.TriggerHeader != "{}" {
			var expectedHeaders map[string]string
			if err := json.Unmarshal([]byte(res.TriggerHeader), &expectedHeaders); err == nil {
				match := true
				for k, v := range expectedHeaders {
					if val, ok := reqHeaders[k]; !ok || val[0] != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
		}

		// Match Body (Jika ada kriteria di DB)
		if res.TriggerBody != "" && res.TriggerBody != "{}" {
			// Kita asumsikan matching sederhana: apakah body mengandung kata kunci tertentu
			// Atau bisa di-unmarshal jika ingin matching JSON key-to-key
			if !strings.Contains(reqBody, res.TriggerBody) {
				continue
			}
		}

		// Jika sampai sini, berarti semua kriteria cocok!
		return &res, nil
	}

	return defaultResponse, nil
}
