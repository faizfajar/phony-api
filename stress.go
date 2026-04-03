package main

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {
	baseUrl := "http://localhost:8080/mocks/v1/pricing"
	totalRequests := 150
	concurrency := 10

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	fmt.Printf("Memulai load test: %d request...\n", totalRequests)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)

		// Logic Anomali:
		// Request ke 145-150 akan menembak trigger "slow"
		currentUrl := baseUrl + "?type=premium"
		if i >= 145 {
			currentUrl = baseUrl + "?type=slow"
			fmt.Printf("[!] Mengirim request lambat (index %d)...\n", i)
		}

		go func(id int, url string) {
			defer wg.Done()
			semaphore <- struct{}{} // Limit concurrency

			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Request %d Error: %v\n", id, err)
			} else {
				resp.Body.Close()
			}

			<-semaphore
		}(i, currentUrl)
	}

	wg.Wait()
	fmt.Println("Load test selesai. Cek log server kamu!")
}
