package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"math"
)

func TestPollSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/self_info":
			_, _ = w.Write([]byte(`{"peer_id":"x","chain":"mainnet","version":"v1.3.4","mump2p":{"total_peers":7}}`))
		case "/health":
			_, _ = w.Write([]byte(`{"status":"healthy","uptime_seconds":10,"checks":{"last_block_age_sec":{"value":4},"mump2p_peers":{"value":7}}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	poll(srv.URL)
	if testutil.ToFloat64(gatewayUp) != 1 {
		t.Fatal("expected up=1")
	}
	if testutil.ToFloat64(peersConnected) != 7 {
		t.Fatal("expected peers=7")
	}
	if testutil.ToFloat64(lastBlockAgeSec) != 4 {
		t.Fatal("expected block age=4")
	}
	if testutil.ToFloat64(healthStatus) != 1 {
		t.Fatal("expected health=1")
	}
}

func TestPollFailureClearsHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	poll(srv.URL)
	if testutil.ToFloat64(gatewayUp) != 0 {
		t.Fatal("expected up=0 on failure")
	}
	if testutil.ToFloat64(healthStatus) != 0 {
		t.Fatal("expected health cleared to 0 on failure")
	}
}

func TestFetchJSONRejectsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	var out selfInfo
	if err := fetchJSON(srv.URL, &out); err == nil {
		t.Fatal("expected error on non-200")
	}
}

func TestPollFailureSetsFreshnessNaN(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	poll(srv.URL)
	if !math.IsNaN(testutil.ToFloat64(peersConnected)) {
		t.Fatal("expected peers=NaN on failure")
	}
	if !math.IsNaN(testutil.ToFloat64(lastBlockAgeSec)) {
		t.Fatal("expected block age=NaN on failure")
	}
	if !math.IsNaN(testutil.ToFloat64(gatewayUptime)) {
		t.Fatal("expected uptime=NaN on failure")
	}
}