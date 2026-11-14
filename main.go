package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	peersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_peers_connected",
		Help: "mump2p peers from /api/v1/self_info.",
	})
	lastBlockAgeSec = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_last_block_age_sec",
		Help: "Seconds since last block arrived (from /health).",
	})
	gatewayUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_uptime_seconds",
		Help: "Gateway uptime in seconds (from /health).",
	})
	healthStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_health_status",
		Help: "1 if overall status is healthy, 0 otherwise.",
	})
	apiErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "optimum_gateway_api_errors_total",
		Help: "Failed HTTP requests to the Optimum Gateway.",
	})
)

func init() {
	prometheus.MustRegister(peersConnected, lastBlockAgeSec, gatewayUptime, healthStatus, apiErrorsTotal)
}

// Exact structure from your real /health JSON
type HealthInfo struct {
	Status        string `json:"status"`
	UptimeSeconds int    `json:"uptime_seconds"`
	Checks        struct {
		LastBlockAgeSec struct {
			Value float64 `json:"value"`
		} `json:"last_block_age_sec"`
		Mump2pPeers struct {
			Value int `json:"value"`
		} `json:"mump2p_peers"`
	} `json:"checks"`
}

// Exact structure from your real /api/v1/self_info JSON
type SelfInfo struct {
	PeerID  string `json:"peer_id"`
	Chain   string `json:"chain"`
	Version string `json:"version"`
	Mump2p  struct {
		TotalPeers int `json:"total_peers"`
	} `json:"mump2p"`
}

func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func poll(baseURL string) {
	// 1. Fetch Topology
	var si SelfInfo
	if err := fetchJSON(baseURL+"/api/v1/self_info", &si); err != nil {
		log.Printf("⚠️ self_info error: %v", err)
		apiErrorsTotal.Inc()
	} else {
		peersConnected.Set(float64(si.Mump2p.TotalPeers))
		fmt.Printf("✅ Topology: chain=%s peers=%d\n", si.Chain, si.Mump2p.TotalPeers)
	}

	// 2. Fetch Real Health & Block Age
	var hi HealthInfo
	if err := fetchJSON(baseURL+"/health", &hi); err != nil {
		log.Printf("⚠️ health error: %v", err)
		apiErrorsTotal.Inc()
	} else {
		lastBlockAgeSec.Set(hi.Checks.LastBlockAgeSec.Value)
		gatewayUptime.Set(float64(hi.UptimeSeconds))
		
		if hi.Status == "healthy" {
			healthStatus.Set(1)
		} else {
			healthStatus.Set(0)
		}
		fmt.Printf("✅ Health: status=%s block_age=%.1fs uptime=%ds\n", 
			hi.Status, hi.Checks.LastBlockAgeSec.Value, hi.UptimeSeconds)
	}
}

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://host.docker.internal:48123/api/v1/self_info"
	}
	
	// Automatically strip the path to get the base URL for the /health endpoint
	baseURL := gatewayURL
	if idx := strings.Index(baseURL, "/api/"); idx != -1 {
		baseURL = baseURL[:idx]
	}

	fmt.Printf("🚀 Optimum Exporter (Production Mode) starting.\nBase URL: %s\n", baseURL)

	go func() {
		for {
			poll(baseURL)
			time.Sleep(10 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9100", nil))
}