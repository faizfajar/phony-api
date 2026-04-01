package service

import (
	"encoding/json"
	"fmt"
	"sort"
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
	MaxWorkers   = 5               // Jumlah worker paralel untuk memproses metrik
	BatchSize    = 100             // Ukuran maksimal batch sebelum simpan ke DB
	BatchTimeout = 5 * time.Second // Waktu tunggu maksimal jika batch belum penuh
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

// metricWorker memproses data metrik secara asynchronous dengan strategi batching
func (s *MockEngineService) metricWorker(workerID int) {
	defer s.wg.Done()
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
	fmt.Printf("[SUCCESS DB] Saved %d metrics to database.\n", len(metrics))
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

func (s *MockEngineService) ExecuteMockMatching(path string, method string, requestQueryParameters map[string][]string) (*model.Response, error) {
	startTime := time.Now()

	targetEndpoint, err := s.endpointRepository.FindEndpointByPathAndMethod(path, method)
	if err != nil {
		return nil, err
	}

	// Sorting berdasarkan jumlah key JSON (paling spesifik di atas)
	sort.Slice(targetEndpoint.Responses, func(i, j int) bool {
		scoreI := getSpecificityScore(targetEndpoint.Responses[i].TriggerParam)
		scoreJ := getSpecificityScore(targetEndpoint.Responses[j].TriggerParam)

		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return targetEndpoint.Responses[i].ID < targetEndpoint.Responses[j].ID
	})

	for _, responseScenario := range targetEndpoint.Responses {
		// Logic Fallback
		if responseScenario.TriggerParam == "" || responseScenario.TriggerParam == "{}" {
			res := s.applyLatencyAndReturn(&responseScenario)
			s.sendMetricAsync(targetEndpoint.ID, path, res.ResponseStatus, startTime)
			return res, nil
		}

		var triggerMap map[string]string
		if err := json.Unmarshal([]byte(responseScenario.TriggerParam), &triggerMap); err != nil {
			fmt.Printf("--- ERROR UNMARSHAL ID %d: %v ---\n", responseScenario.ID, err)
			continue
		}

		// Matching Parameters
		isMatch := true
		for key, expectedValue := range triggerMap {
			actualValue, exists := requestQueryParameters[key]
			if !exists || actualValue[0] != expectedValue {
				isMatch = false
				break
			}
		}

		if isMatch {
			res := s.applyLatencyAndReturn(&responseScenario)
			s.sendMetricAsync(targetEndpoint.ID, path, res.ResponseStatus, startTime)
			return res, nil
		}
	}

	return nil, nil
}

// sendMetricAsync membungkus pengiriman data ke channel agar rapi
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
		// Channel penuh, drop data agar tidak mengganggu response time API
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
	close(s.metricChan) // Berhenti menerima data baru
	s.wg.Wait()         // Tunggu semua worker selesai save sisa batch ke DB
	fmt.Println("[SHUTDOWN] All metrics sav Goodbye!")
}
