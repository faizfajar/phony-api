package main

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {
	url := "http://localhost:8080/mocks/v1/pricing?type=premium"
	totalRequests := 150
	concurrency := 10 // Mengirim 10 request secara paralel sekaligus

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	fmt.Printf("Memulai load test: %d request...\n", totalRequests)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{} // Limit concurrency

			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Request %d Error: %v\n", id, err)
			} else {
				resp.Body.Close()
			}

			<-semaphore
		}(i)
	}

	wg.Wait()
	fmt.Println("Load test selesai. Cek log server kamu!")
}
