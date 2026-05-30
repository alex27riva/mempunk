// Package explorer builds typed view models from Bitcoin Core RPC responses.
package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/alex27riva/mempunk/internal/cache"
	"github.com/alex27riva/mempunk/internal/config"
	"github.com/alex27riva/mempunk/internal/rpc"
)

// scanJob holds the state of a background scan goroutine.
// Explorer has one slot per scan type; only one scan of each type runs at a time.
type scanJob[V any] struct {
	addr string
	done bool
	val  V
	err  error
}

// Explorer orchestrates RPC calls and builds view models for HTTP handlers.
type Explorer struct {
	rpc   *rpc.Client
	cache *cache.Cache[json.RawMessage]
	cfg   *config.Config
	log   *slog.Logger

	utxoMu  sync.Mutex
	utxoJob *scanJob[*UTXOScanVM]
	histMu  sync.Mutex
	histJob *scanJob[*HistoryScanVM]
}

// New creates an Explorer. All parameters are required.
func New(cfg *config.Config, r *rpc.Client, c *cache.Cache[json.RawMessage], log *slog.Logger) *Explorer {
	return &Explorer{rpc: r, cache: c, cfg: cfg, log: log}
}

// cachedCall calls method with params, checking and writing the cache.
func (e *Explorer) cachedCall(ctx context.Context, key, method string, params ...any) (json.RawMessage, error) {
	if raw, ok := e.cache.Get(key); ok {
		e.log.Debug("cache hit", "key", key)
		return raw, nil
	}
	raw, err := e.rpc.Call(ctx, method, params...)
	if err != nil {
		return nil, err
	}
	e.cache.Put(key, raw)
	return raw, nil
}

// isNumeric reports whether s is a non-empty decimal integer string.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// scriptAddress extracts the address from an rpcScript, preferring the modern
// single-address field and falling back to the legacy addresses array.
func scriptAddress(s rpcScript) string {
	if s.Address != "" {
		return s.Address
	}
	if len(s.Addresses) > 0 {
		return s.Addresses[0]
	}
	return ""
}

// unmarshal is a convenience wrapper that adds context to decode errors.
func unmarshal(data json.RawMessage, v any, what string) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode %s: %w", what, err)
	}
	return nil
}
