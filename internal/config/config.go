// Package config loads, validates, and resolves mempunk's runtime configuration.
//
// Configuration is read from a YAML file (see Load) and may be overridden by
// MEMPUNK_* environment variables. After loading, every network-dependent value
// (Core chain string, default RPC port, address parameters, UI accent class)
// is derived from a single source of truth: the network table below.
//
// This file holds only pure, dependency-free logic so it is trivially testable;
// the YAML/file glue lives in load.go.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
)

// Network is one of the Bitcoin networks mempunk can run against. A single
// mempunk instance serves exactly one network.
type Network string

const (
	Mainnet  Network = "mainnet"
	Testnet3 Network = "testnet3"
	Testnet4 Network = "testnet4"
	Regtest  Network = "regtest"
)

// NetworkParams is the per-network source of truth. Everything network-specific
// is looked up here so the rest of the codebase never hardcodes a port or HRP.
type NetworkParams struct {
	Network Network
	// Chain is the value Bitcoin Core reports in getblockchaininfo.chain and
	// accepts via -chain. NOTE: testnet3 is "test", not "testnet3".
	Chain string
	// DefaultRPCPort is used when rpc.port is left unset (0).
	DefaultRPCPort int
	// Bech32HRP is the segwit human-readable prefix ("bc", "tb", "bcrt").
	Bech32HRP string
	// Base58LeadChars are the legal leading characters of legacy/base58 addrs
	// on this network (e.g. "13" for mainnet, "mn2" for test/regtest).
	Base58LeadChars string
	// AccentClass is the <body> CSS class that tints the UI per network so the
	// active chain is impossible to mistake at a glance.
	AccentClass string
	// MinCoreVersion is the lowest Bitcoin Core CLIENT_VERSION (e.g. 280000 for
	// v28.0) that supports this network; the rpc component can warn/fail if the
	// connected node reports a lower getnetworkinfo.version.
	MinCoreVersion int
}

// networkParams is the canonical table. Add signet here if ever needed:
// {Chain: "signet", DefaultRPCPort: 38332, Bech32HRP: "tb", ...}.
var networkParams = map[Network]NetworkParams{
	Mainnet:  {Mainnet, "main", 8332, "bc", "13", "net-mainnet", 250000},
	Testnet3: {Testnet3, "test", 18332, "tb", "mn2", "net-testnet3", 250000},
	Testnet4: {Testnet4, "testnet4", 48332, "tb", "mn2", "net-testnet4", 280000},
	Regtest:  {Regtest, "regtest", 18443, "bcrt", "mn2", "net-regtest", 250000},
}

// Params returns the network's parameters and whether the network is known.
func (n Network) Params() (NetworkParams, bool) {
	p, ok := networkParams[n]
	return p, ok
}

// Valid reports whether n is a supported network.
func (n Network) Valid() bool {
	_, ok := networkParams[n]
	return ok
}

// LooksLikeAddress is a cheap, network-aware heuristic for the /search
// classifier. It is deliberately permissive — the node's validateaddress RPC is
// always the source of truth for real validity. It only helps decide whether to
// route an input to the address view rather than treating it as a hash.
func (p NetworkParams) LooksLikeAddress(s string) bool {
	if s == "" {
		return false
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, p.Bech32HRP+"1") {
		return true // bech32 / bech32m
	}
	if n := len(s); n >= 25 && n <= 36 && strings.ContainsRune(p.Base58LeadChars, rune(s[0])) {
		return true // legacy base58check
	}
	return false
}

// Config is the fully resolved runtime configuration.
type Config struct {
	Network  Network        `yaml:"network"`
	RPC      RPCConfig      `yaml:"rpc"`
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	Explorer ExplorerConfig `yaml:"explorer"`
}

// RPCConfig describes how to reach bitcoind's JSON-RPC interface. Authentication
// is either cookie-file based OR user/password — never both (see Validate).
type RPCConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"` // 0 => use the network's default port
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	CookieFile string `yaml:"cookie_file"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"` // host:port for the HTTP server
}

type LogConfig struct {
	Level  string `yaml:"level"`  // debug|info|warn|error
	Format string `yaml:"format"` // text|json
}

type ExplorerConfig struct {
	LatestBlocks   int   `yaml:"latest_blocks"`
	BlockTxDetails *bool `yaml:"block_tx_details"` // pointer so default can be true
	CacheSize      int   `yaml:"cache_size"`
}

// ShowTxDetails reports whether block views should load full tx details.
// Defaults to true when unset.
func (e ExplorerConfig) ShowTxDetails() bool {
	return e.BlockTxDetails == nil || *e.BlockTxDetails
}

// Params returns the resolved network parameters. Safe to call after Validate
// has confirmed the network is valid; falls back to mainnet otherwise.
func (c *Config) Params() NetworkParams {
	if p, ok := c.Network.Params(); ok {
		return p
	}
	return networkParams[Mainnet]
}

// RPCAddr returns the host:port to dial for JSON-RPC.
func (c *Config) RPCAddr() string {
	return net.JoinHostPort(c.RPC.Host, strconv.Itoa(c.RPC.Port))
}

// AuthMethod returns a short, secret-free description of how RPC auth resolves.
func (c *Config) AuthMethod() string {
	if c.RPC.CookieFile != "" {
		return "cookie"
	}
	if c.RPC.User != "" {
		return "userpass"
	}
	return "none"
}

// ResolveAuth returns the basic-auth credentials for an RPC call. When a cookie
// file is configured it is read fresh on every call so that cookie rotation
// (bitcoind regenerates .cookie on restart) is picked up without a restart.
func (c *Config) ResolveAuth() (user, pass string, err error) {
	if c.RPC.CookieFile != "" {
		raw, err := os.ReadFile(c.RPC.CookieFile)
		if err != nil {
			return "", "", fmt.Errorf("read rpc cookie %q: %w", c.RPC.CookieFile, err)
		}
		line := strings.TrimSpace(string(raw))
		u, p, ok := strings.Cut(line, ":") // cookie is "__cookie__:<password>"
		if !ok {
			return "", "", fmt.Errorf("malformed rpc cookie %q: expected user:password", c.RPC.CookieFile)
		}
		return u, p, nil
	}
	return c.RPC.User, c.RPC.Password, nil
}

// applyEnv overlays MEMPUNK_* environment variables onto an already-parsed
// Config. Only variables that are actually set take effect.
func (c *Config) applyEnv() error {
	if v, ok := os.LookupEnv("MEMPUNK_NETWORK"); ok {
		c.Network = Network(v)
	}
	if v, ok := os.LookupEnv("MEMPUNK_RPC_HOST"); ok {
		c.RPC.Host = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_RPC_PORT"); ok {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("MEMPUNK_RPC_PORT: %w", err)
		}
		c.RPC.Port = p
	}
	if v, ok := os.LookupEnv("MEMPUNK_RPC_USER"); ok {
		c.RPC.User = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_RPC_PASSWORD"); ok {
		c.RPC.Password = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_RPC_COOKIE_FILE"); ok {
		c.RPC.CookieFile = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_SERVER_LISTEN"); ok {
		c.Server.Listen = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_LOG_LEVEL"); ok {
		c.Log.Level = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_LOG_FORMAT"); ok {
		c.Log.Format = v
	}
	if v, ok := os.LookupEnv("MEMPUNK_EXPLORER_LATEST_BLOCKS"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("MEMPUNK_EXPLORER_LATEST_BLOCKS: %w", err)
		}
		c.Explorer.LatestBlocks = n
	}
	if v, ok := os.LookupEnv("MEMPUNK_EXPLORER_CACHE_SIZE"); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("MEMPUNK_EXPLORER_CACHE_SIZE: %w", err)
		}
		c.Explorer.CacheSize = n
	}
	if v, ok := os.LookupEnv("MEMPUNK_EXPLORER_BLOCK_TX_DETAILS"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("MEMPUNK_EXPLORER_BLOCK_TX_DETAILS: %w", err)
		}
		c.Explorer.BlockTxDetails = &b
	}
	return nil
}

// applyDefaults fills unset fields. It must run after the network is known so it
// can pick the correct default RPC port.
func (c *Config) applyDefaults() {
	if c.Network == "" {
		c.Network = Mainnet
	}
	if c.RPC.Host == "" {
		c.RPC.Host = "127.0.0.1"
	}
	if c.RPC.Port == 0 {
		if p, ok := c.Network.Params(); ok {
			c.RPC.Port = p.DefaultRPCPort
		}
	}
	if c.Server.Listen == "" {
		c.Server.Listen = "127.0.0.1:8080"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}
	if c.Explorer.LatestBlocks == 0 {
		c.Explorer.LatestBlocks = 12
	}
	if c.Explorer.CacheSize == 0 {
		c.Explorer.CacheSize = 1000
	}
	if c.Explorer.BlockTxDetails == nil {
		t := true
		c.Explorer.BlockTxDetails = &t
	}
}

var (
	validLogLevels  = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	validLogFormats = map[string]bool{"text": true, "json": true}
)

// Validate checks the fully-defaulted config and returns the first problem
// found. Call it after applyEnv and applyDefaults.
func (c *Config) Validate() error {
	if !c.Network.Valid() {
		return fmt.Errorf("network %q is not supported (use mainnet|testnet3|testnet4|regtest)", c.Network)
	}
	if c.RPC.Host == "" {
		return errors.New("rpc.host must not be empty")
	}
	if c.RPC.Port < 1 || c.RPC.Port > 65535 {
		return fmt.Errorf("rpc.port %d out of range 1-65535", c.RPC.Port)
	}

	// Auth: cookie XOR user+password.
	hasCookie := c.RPC.CookieFile != ""
	hasUser, hasPass := c.RPC.User != "", c.RPC.Password != ""
	switch {
	case hasCookie && (hasUser || hasPass):
		return errors.New("rpc auth ambiguous: set either rpc.cookie_file or rpc.user/rpc.password, not both")
	case hasCookie:
		// ok
	case hasUser != hasPass:
		return errors.New("rpc.user and rpc.password must both be set")
	case !hasUser && !hasPass:
		return errors.New("no rpc auth configured: set rpc.cookie_file or rpc.user/rpc.password")
	}

	if _, _, err := net.SplitHostPort(c.Server.Listen); err != nil {
		return fmt.Errorf("server.listen %q is not host:port: %w", c.Server.Listen, err)
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("log.level %q invalid (debug|info|warn|error)", c.Log.Level)
	}
	if !validLogFormats[c.Log.Format] {
		return fmt.Errorf("log.format %q invalid (text|json)", c.Log.Format)
	}
	if c.Explorer.LatestBlocks < 1 {
		return fmt.Errorf("explorer.latest_blocks must be >= 1, got %d", c.Explorer.LatestBlocks)
	}
	if c.Explorer.CacheSize < 0 {
		return fmt.Errorf("explorer.cache_size must be >= 0, got %d", c.Explorer.CacheSize)
	}
	return nil
}

// LogValue implements slog.LogValuer so a Config can be logged safely: the RPC
// password and cookie contents are never emitted.
func (c Config) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("network", string(c.Network)),
		slog.String("rpc_addr", c.RPCAddr()),
		slog.String("rpc_auth", c.AuthMethod()),
		slog.String("listen", c.Server.Listen),
		slog.String("log_level", c.Log.Level),
		slog.String("log_format", c.Log.Format),
		slog.Int("latest_blocks", c.Explorer.LatestBlocks),
		slog.Bool("block_tx_details", c.Explorer.ShowTxDetails()),
		slog.Int("cache_size", c.Explorer.CacheSize),
	}
	if c.AuthMethod() == "userpass" {
		attrs = append(attrs, slog.String("rpc_user", c.RPC.User))
	}
	return slog.GroupValue(attrs...)
}
