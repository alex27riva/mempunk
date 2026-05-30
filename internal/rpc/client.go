// Package rpc provides a thin, read-only JSON-RPC client for Bitcoin Core.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alex27riva/mempunk/internal/config"
)

const (
	defaultTimeout = 30 * time.Second
	scanTimeout    = 5 * time.Minute
)

// RPCError is a Bitcoin Core JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
	ID     int             `json:"id"`
}

// Client is a thin JSON-RPC client for Bitcoin Core.
type Client struct {
	cfg        *config.Config
	log        *slog.Logger
	httpClient *http.Client
	scanClient *http.Client
}

// New creates a Client backed by cfg and log.
func New(cfg *config.Config, log *slog.Logger) *Client {
	return &Client{
		cfg:        cfg,
		log:        log,
		httpClient: &http.Client{Timeout: defaultTimeout},
		scanClient: &http.Client{Timeout: scanTimeout},
	}
}

// Call issues a JSON-RPC request using the standard timeout profile.
func (c *Client) Call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return c.do(ctx, c.httpClient, method, params)
}

// CallScan issues a JSON-RPC request using the extended scan timeout profile,
// suitable for scantxoutset and scanblocks which may run for minutes.
func (c *Client) CallScan(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	return c.do(ctx, c.scanClient, method, params)
}

func (c *Client) do(ctx context.Context, hc *http.Client, method string, params []any) (json.RawMessage, error) {
	if params == nil {
		params = []any{}
	}
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://"+c.cfg.RPCAddr(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	user, pass, err := c.cfg.ResolveAuth()
	if err != nil {
		return nil, fmt.Errorf("resolve rpc auth: %w", err)
	}
	req.SetBasicAuth(user, pass)

	start := time.Now()
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc %q: %w", method, err)
	}
	defer resp.Body.Close()

	c.log.Debug("rpc call", "method", method, "status", resp.StatusCode, "ms", time.Since(start).Milliseconds())

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("rpc %q: unauthorized (check rpc credentials)", method)
	}

	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, fmt.Errorf("decode rpc response for %q: %w", method, err)
	}
	if rr.Error != nil {
		return nil, rr.Error
	}
	return rr.Result, nil
}
