package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/faizfajar/phony-api/internal/model"
	"github.com/faizfajar/phony-api/internal/repository"
	"github.com/google/uuid"
)

type MockEngineService struct {
	endpointRepository repository.EndpointRepository
	metricChan         chan model.APIMetric
	wg                 sync.WaitGroup
}

const (
	MaxWorkers   = 5
	BatchSize    = 100
	BatchTimeout = 5 * time.Second
)

func NewMockEngineService(repo repository.EndpointRepository) *MockEngineService {
	s := &MockEngineService{
		endpointRepository: repo,
		metricChan:         make(chan model.APIMetric, 1000),
	}

	// Inisialisasi Worker Pool
	for i := 0; i < MaxWorkers; i++ {
		s.wg.Add(1)
		go s.metricWorker(i)
	}

	return s
}

func (s *MockEngineService) metricWorker(workerID int) {
	defer s.wg.Done()
	fmt.Printf("[WORKER %d] Started monitoring metrics channel\n", workerID)

	var batch []model.APIMetric

	ticker := time.NewTicker(BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case m, ok := <-s.metricChan:
			if !ok {
				if len(batch) > 0 {
					s.saveBatchToDB(batch)
				}
				return
			}
			batch = append(batch, m)

			if len(batch) >= BatchSize {
				s.saveBatchToDB(batch)
				batch = []model.APIMetric{}
			}

		case <-ticker.C:
			if len(batch) > 0 {
				s.saveBatchToDB(batch)
				batch = []model.APIMetric{}
			}
		}
	}
}

func (s *MockEngineService) saveBatchToDB(metrics []model.APIMetric) {
	if len(metrics) == 0 {
		return
	}

	err := s.endpointRepository.CreateMetricsBatch(metrics)
	if err != nil {
		fmt.Printf("[BATCH DB] Error saving metrics: %v\n", err)
		return
	}
}

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

func (s *MockEngineService) ExecuteMockMatching(path, method string, params map[string][]string, headers http.Header, body string) (*model.Response, error) {
	startTime := time.Now()

	endpoint, err := s.endpointRepository.FindEndpointByPathAndMethod(path, method)
	if err != nil {
		return nil, err
	}

	var bestMatch *model.Response
	highestScore := -1

	for i := range endpoint.Responses {
		res := &endpoint.Responses[i]
		currentScore := 0
		matchFound := true

		// Match Headers
		if res.TriggerHeader != "" && res.TriggerHeader != "{}" {
			var expectedHeaders map[string]string
			if err := json.Unmarshal([]byte(res.TriggerHeader), &expectedHeaders); err == nil {
				for k, v := range expectedHeaders {
					if headers.Get(k) != v {
						matchFound = false
						break
					}
				}
				if matchFound {
					currentScore += getSpecificityScore(res.TriggerHeader) * 10
				}
			}
		}

		if !matchFound {
			continue
		}

		// Match Body
		if res.TriggerBody != "" {
			if strings.Contains(body, res.TriggerBody) {
				currentScore += 5
			} else {
				matchFound = false
			}
		}

		if !matchFound {
			continue
		}

		// Match Params
		if res.TriggerParam != "" && res.TriggerParam != "{}" {
			currentScore += getSpecificityScore(res.TriggerParam)
		}

		// Evaluate if this is the most specific match
		if matchFound && currentScore > highestScore {
			highestScore = currentScore
			bestMatch = res
		}
	}

	if bestMatch != nil {
		// Log metrics asynchronously to the worker pool
		s.sendMetricAsync(endpoint.ID, path, bestMatch.ResponseStatus, startTime)

		// Apply artificial latency before returning
		return s.applyLatencyAndReturn(bestMatch), nil
	}

	return nil, nil
}

func (s *MockEngineService) RegisterNewEndpoint(endpoint *model.Endpoint) (*model.Endpoint, error) {
	endpoint.ID = uuid.New()
	err := s.endpointRepository.CreateEndpoint(endpoint)
	return endpoint, err
}

func (s *MockEngineService) GetEndpointByID(id string) (*model.Endpoint, error) {
	uID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.endpointRepository.FindEndpointByID(uID)
}

func (s *MockEngineService) GenerateK6Script(endpoint *model.Endpoint) string {
	// Ambil sampel data dari response pertama untuk dijadikan trigger k6
	var sampleHeader = "{}"
	var sampleBody = ""
	if len(endpoint.Responses) > 0 {
		if endpoint.Responses[0].TriggerHeader != "" {
			sampleHeader = endpoint.Responses[0].TriggerHeader
		}
		sampleBody = endpoint.Responses[0].TriggerBody
	}

	// Gunakan Backtick agar penulisan JS di dalam Go lebih rapi
	return fmt.Sprintf(`
		import http from 'k6/http';
		import { check, sleep } from 'k6';

		export let options = {
				vus: %d,
				duration: '%ds',
				thresholds: {
						http_req_duration: ['p(95)<%d'],
				},
		};

		export default function () {
				// Nantinya localhost:8080 bisa lo ganti jadi IP VPS lo di .env
				const url = 'http://localhost:8080/mocks%s';
				
				const payload = '%s';
				const params = {
						headers: %s,
				};

				let res;
				// Logika otomatis pilih Method sesuai database
				if ("%s" === "POST") {
						res = http.post(url, payload, params);
				} else {
						res = http.get(url, params);
				}

				check(res, { 
						'status is 200': (r) => r.status === 200 
				});
				
				sleep(1);
		}`,
		endpoint.VUsers,
		endpoint.Duration,
		endpoint.ThresholdP95,
		endpoint.Path,
		sampleBody,
		sampleHeader,
		endpoint.Method,
	)
}

func (s *MockEngineService) sendMetricAsync(endpointID uuid.UUID, path string, status int, start time.Time) {
	metricData := model.APIMetric{
		EndpointID: endpointID,
		Path:       path,
		DurationMS: time.Since(start).Milliseconds(),
		StatusCode: status,
		CreatedAt:  time.Now(),
	}

	select {
	case s.metricChan <- metricData:
	default:
		// Drop data if channel is full to prevent slowing down the mock response
	}
}

func (s *MockEngineService) applyLatencyAndReturn(response *model.Response) *model.Response {
	if response.DelayMS > 0 {
		time.Sleep(time.Duration(response.DelayMS) * time.Millisecond)
	}
	return response
}

func (s *MockEngineService) Shutdown() {
	fmt.Println("[SHUTDOWN] Closing metric channel...")
	close(s.metricChan)
	s.wg.Wait()
	fmt.Println("[SHUTDOWN] All metrics saveGoodbye!")
}
