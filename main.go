package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var httpClient = &http.Client{Timeout: 3 * time.Second}
var debug = os.Getenv("DEBUG") == "1"
var lastUp = -1

var (
	gatewayUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_up", Help: "1 if the last telemetry scrape succeeded, else 0."})
	peersConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_peers_connected", Help: "mump2p peers from /api/v1/self_info."})
	lastBlockAgeSec = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_last_block_age_sec", Help: "Seconds since last block (from /health)."})
	healthStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_health_status", Help: "1 if /health status==healthy, else 0."})
	gatewayUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "optimum_gateway_uptime_seconds", Help: "Gateway uptime seconds (from /health)."})
	apiErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "optimum_gateway_api_errors_total", Help: "Failed telemetry fetches."})
)

func init() {
	prometheus.MustRegister(gatewayUp, peersConnected, lastBlockAgeSec,
		healthStatus, gatewayUptime, apiErrorsTotal)
}

type selfInfo struct {
	PeerID  string `json:"peer_id"`
	Chain   string `json:"chain"`
	Version string `json:"version"`
	Mump2p  struct {
		TotalPeers int `json:"total_peers"`
	} `json:"mump2p"`
}

type healthInfo struct {
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

func fetchJSON(url string, target interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func poll(base string) {
	ok := true

	var si selfInfo
	if err := fetchJSON(base+"/api/v1/self_info", &si); err != nil {
		apiErrorsTotal.Inc()
		ok = false
		if debug {
			log.Printf("self_info error: %v", err)
		}
	} else {
		peersConnected.Set(float64(si.Mump2p.TotalPeers))
	}

	var hi healthInfo
	if err := fetchJSON(base+"/health", &hi); err != nil {
		apiErrorsTotal.Inc()
		healthStatus.Set(0) // never leave a stale "healthy"
		ok = false
		if debug {
			log.Printf("health error: %v", err)
		}
	} else {
		lastBlockAgeSec.Set(hi.Checks.LastBlockAgeSec.Value)
		gatewayUptime.Set(float64(hi.UptimeSeconds))
		if hi.Status == "healthy" {
			healthStatus.Set(1)
		} else {
			healthStatus.Set(0)
		}
	}

	up := 0
	if ok {
		up = 1
	}
	gatewayUp.Set(float64(up))
	if up != lastUp { // log only on state change, not every poll
		if up == 1 {
			log.Printf("gateway telemetry UP")
		} else {
			log.Printf("gateway telemetry DOWN")
		}
		lastUp = up
	}
}

func deriveBase(u string) string {
	if i := strings.Index(u, "/api/"); i >= 0 {
		return u[:i]
	}
	if i := strings.Index(u, "/health"); i >= 0 {
		return u[:i]
	}
	return strings.TrimRight(u, "/")
}

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://host.docker.internal:48123/api/v1/self_info"
	}
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":9188" // avoid node_exporter's 9100
	}
	interval := 10 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}
	base := deriveBase(gatewayURL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		poll(base)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				poll(base)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: listenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("optimum-exporter listening on %s, polling %s", listenAddr, base)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}