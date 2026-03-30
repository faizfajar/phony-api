package service

import (
	"encoding/json"
	"fmt"
	"sort" // Tambahkan ini
	"time"

	"github.com/faizfajar/phony-api/internal/model"
	"github.com/faizfajar/phony-api/internal/repository"
)

type MockEngineService struct {
	endpointRepository repository.EndpointRepository
}

func NewMockEngineService(endpointRepository repository.EndpointRepository) *MockEngineService {
	return &MockEngineService{endpointRepository: endpointRepository}
}

// Helper untuk menghitung jumlah key dalam JSON trigger
func getSpecificityScore(triggerJSON string) int {
	if triggerJSON == "" || triggerJSON == "{}" {
		return 0
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(triggerJSON), &m); err != nil {
		return 0
	}
	return len(m)
}

func (service *MockEngineService) ExecuteMockMatching(path string, method string, requestQueryParameters map[string][]string) (*model.Response, error) {
	targetEndpoint, err := service.endpointRepository.FindEndpointByPathAndMethod(path, method)
	if err != nil {
		return nil, err
	}

	// sort the responses by specificity score (number of keys in trigger) in descending order
	sort.Slice(targetEndpoint.Responses, func(i, j int) bool {
		scoreI := getSpecificityScore(targetEndpoint.Responses[i].TriggerParam)
		scoreJ := getSpecificityScore(targetEndpoint.Responses[j].TriggerParam)

		if scoreI != scoreJ {
			return scoreI > scoreJ // Skor lebih tinggi (lebih banyak key) di atas
		}
		return targetEndpoint.Responses[i].ID < targetEndpoint.Responses[j].ID // Tie-breaker pakai ID
	})

	for _, responseScenario := range targetEndpoint.Responses {
		// Sekarang Default Fallback pasti akan berada di urutan terakhir karena skornya 0
		if responseScenario.TriggerParam == "" || responseScenario.TriggerParam == "{}" {
			return service.applyLatencyAndReturn(&responseScenario), nil
		}

		var triggerMap map[string]string
		unmarshalError := json.Unmarshal([]byte(responseScenario.TriggerParam), &triggerMap)

		if unmarshalError != nil {
			fmt.Printf("--- ERROR UNMARSHAL ID %d: %v ---\n", responseScenario.ID, unmarshalError)
			continue
		}

		isMatch := true
		for key, expectedValue := range triggerMap {
			actualValue, exists := requestQueryParameters[key]
			if !exists || actualValue[0] != expectedValue {
				isMatch = false
				break
			}
		}

		if isMatch {
			return service.applyLatencyAndReturn(&responseScenario), nil
		}
	}

	return nil, nil
}

func (service *MockEngineService) applyLatencyAndReturn(response *model.Response) *model.Response {
	if response.DelayMS > 0 {
		time.Sleep(time.Duration(response.DelayMS) * time.Millisecond)
	}
	return response
}
