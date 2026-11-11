package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 1. Define the Prometheus Metrics
var (
	peersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_peers_connected",
		Help: "Current number of connected mump2p peers.",
	})
	blockArrivalMs = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_block_arrival_ms",
		Help: "Time in milliseconds it took for the latest block to arrive via mump2p.",
	})
	blocksReceivedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "optimum_gateway_blocks_received_total",
		Help: "Total number of blocks received via the mump2p network.",
	})
)

func init() {
	// Register the metrics with Prometheus's default registry
	prometheus.MustRegister(peersConnected)
	prometheus.MustRegister(blockArrivalMs)
	prometheus.MustRegister(blocksReceivedTotal)
}

// 2. Simulate fetching data from the Optimum Gateway API
// (Later, we will replace this with the actual HTTP client code)
func fetchGatewayData() {
	// Simulate network data
	peers := rand.Intn(50) + 10          // Random number between 10 and 60
	arrivalTime := rand.Float64()*100 + 150.0 // Random time between 150ms and 250ms

	// Update the Prometheus metrics
	peersConnected.Set(float64(peers))
	blockArrivalMs.Set(arrivalTime)
	blocksReceivedTotal.Inc()
}

// 3. Background worker to poll the API periodically
func startMetricsUpdater() {
	go func() {
		for {
			fetchGatewayData()
			time.Sleep(5 * time.Second) // Poll every 5 seconds
		}
	}()
}

func main() {
	fmt.Println("Starting Optimum Gateway Prometheus Exporter on :9100")
	
	// Start the background worker
	startMetricsUpdater()

	// Expose the /metrics endpoint for Prometheus to scrape
	http.Handle("/metrics", promhttp.Handler())
	
	// Add a simple root endpoint for humans
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>Optimum Exporter</title></head><body><h1>Optimum Gateway Exporter</h1><p><a href="/metrics">View Metrics</a></p></body></html>`))
	})

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(":9100", nil))
}