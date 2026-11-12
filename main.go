package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
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
	apiErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "optimum_gateway_api_errors_total",
		Help: "Total number of failed API requests to the Optimum Gateway.",
	})
)

func init() {
	prometheus.MustRegister(peersConnected)
	prometheus.MustRegister(blockArrivalMs)
	prometheus.MustRegister(blocksReceivedTotal)
	prometheus.MustRegister(apiErrorsTotal)
}

// 2. Define the expected JSON structure from the Optimum Gateway API
type GatewayResponse struct {
	Peers       int     `json:"peers_connected"`
	ArrivalTime float64 `json:"block_arrival_ms"`
	BlocksTotal int     `json:"blocks_received_total"`
}

// 3. The Real HTTP Fetcher with Fallback
func fetchGatewayData(gatewayURL string) {
	// Try to fetch real data
	resp, err := http.Get(gatewayURL)
	if err != nil {
		// Fallback to simulated data if API is unavailable
		log.Printf("API unavailable, using simulated data: %v", err)
		useSimulatedData()
		apiErrorsTotal.Inc()
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response, using simulated data: %v", err)
		useSimulatedData()
		apiErrorsTotal.Inc()
		return
	}

	var data GatewayResponse
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("Error parsing JSON, using simulated data: %v", err)
		useSimulatedData()
		apiErrorsTotal.Inc()
		return
	}

	// Success! Update with REAL data
	peersConnected.Set(float64(data.Peers))
	blockArrivalMs.Set(data.ArrivalTime)
	blocksReceivedTotal.Add(float64(data.BlocksTotal))
	
	fmt.Printf("✅ Real data fetched: Peers=%d, Arrival=%.2fms, TotalBlocks=%d\n", 
		data.Peers, data.ArrivalTime, data.BlocksTotal)
}

// 4. Simulated Data Fallback
func useSimulatedData() {
	peers := rand.Intn(50) + 10
	arrivalTime := rand.Float64()*100 + 150.0
	
	peersConnected.Set(float64(peers))
	blockArrivalMs.Set(arrivalTime)
	blocksReceivedTotal.Add(1) // Increment by 1 for simulated mode
}

func startMetricsUpdater(gatewayURL string) {
	go func() {
		for {
			fetchGatewayData(gatewayURL)
			time.Sleep(5 * time.Second)
		}
	}()
}

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080/status"
	}

	fmt.Printf("🚀 Starting Optimum Gateway Prometheus Exporter on :9100\n")
	fmt.Printf(" Polling Optimum Gateway at: %s\n", gatewayURL)
	fmt.Printf(" If gateway is unavailable, will use simulated data for testing.\n")
	
	startMetricsUpdater(gatewayURL)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><title>Optimum Exporter</title></head><body><h1>Optimum Gateway Exporter</h1><p><a href="/metrics">View Metrics</a></p></body></html>`))
	})

	log.Fatal(http.ListenAndServe(":9100", nil))
}