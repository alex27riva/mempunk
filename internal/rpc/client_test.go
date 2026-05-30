package rpc

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alex27riva/mempunk/internal/config"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	host, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	port, _ := strconv.Atoi(portStr)

	cfg := &config.Config{
		Network: config.Regtest,
		RPC: config.RPCConfig{
			Host:     host,
			Port:     port,
			User:     "user",
			Password: "pass",
		},
	}
	return New(cfg, slog.Default())
}

// okHandler returns a valid JSON-RPC success response wrapping result.
func okHandler(result any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, _ := json.Marshal(result)
		resp := map[string]any{"result": json.RawMessage(raw), "error": nil, "id": 1}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// errHandler returns a valid JSON-RPC error response.
func errHandler(code int, msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"result": nil,
			"error":  map[string]any{"code": code, "message": msg},
			"id":     1,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func TestCallSuccess(t *testing.T) {
	client := newTestClient(t, okHandler(map[string]string{"chain": "regtest"}))
	raw, err := client.Call(context.Background(), "getblockchaininfo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw) == 0 {
		t.Error("result should not be empty")
	}
}

func TestCallRPCError(t *testing.T) {
	client := newTestClient(t, errHandler(-32601, "Method not found"))
	_, err := client.Call(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	rpcErr, ok := err.(*RPCError)
	if !ok {
		t.Fatalf("expected *RPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32601 {
		t.Errorf("code = %d, want -32601", rpcErr.Code)
	}
}

func TestCallUnauthorized(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := client.Call(context.Background(), "getblockchaininfo")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestPingSuccess(t *testing.T) {
	var callCount int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch callCount {
		case 1: // getblockchaininfo
			result = map[string]any{"chain": "regtest", "blocks": 0}
		case 2: // getnetworkinfo
			result = map[string]any{"version": 260000}
		}
		resp := map[string]any{"result": result, "error": nil, "id": 1}
		_ = json.NewEncoder(w).Encode(resp)
	})
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestPingChainMismatch(t *testing.T) {
	// Node reports "main" but config is regtest.
	client := newTestClient(t, okHandler(map[string]string{"chain": "main"}))
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected chain mismatch error, got nil")
	}
}

func TestPingVersionWarning(t *testing.T) {
	// Version below MinCoreVersion (250000 for regtest): warn but do not error.
	var callCount int
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch callCount {
		case 1:
			result = map[string]any{"chain": "regtest"}
		case 2:
			result = map[string]any{"version": 240000}
		}
		resp := map[string]any{"result": result, "error": nil, "id": 1}
		_ = json.NewEncoder(w).Encode(resp)
	})
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("ping should succeed with low version (warn only): %v", err)
	}
}
