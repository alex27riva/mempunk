# CLAUDE.md — mempunk

A minimal, cyberpunk Bitcoin block explorer in Go, served directly from a
Bitcoin Core full node over JSON-RPC. Server-side rendered, JavaScript-free
wherever possible. This file is the source of truth for the project; read it at
the start of every session.

Suggested module path: `github.com/alex27riva/mempunk`.

---

## Working agreement (HARD RULE — do not deviate)

**Build exactly one component at a time. Never scaffold the whole project at
once.** For each component:

1. Confirm the component's scope and its exported interface with me **before**
   writing it.
2. Implement only that component plus the minimum needed to compile and test it
   (stub/fake unbuilt dependencies).
3. Ensure `go build ./...` and `go vet ./...` are clean, and `gofmt` is applied.
4. Tell me exactly how to test it in isolation.
5. **Stop and wait for my confirmation before starting the next component.**

Do not implement later-component features "while you're in there." If a
component's interface needs to change, raise it and wait. One logical commit per
component/step, conventional-commit style.

When I say "start", do the bootstrap in the next section, then write Component 1
(`config`) **verbatim** from the canonical source at the end of this file, run
its tests, and stop.

## Bootstrap (one time)

```sh
go mod init github.com/alex27riva/mempunk
go get gopkg.in/yaml.v3
go get github.com/labstack/echo/v4
```

Add a `.gitignore` containing at least: `config.yaml`, `/mempunk`, `*.cookie`.

## Tech stack & constraints

- **Go** (latest stable), idiomatic, `gofmt`/`go vet` clean.
- **Echo v4** for HTTP.
- **`log/slog`** for logging — no third-party logging libraries.
- **`html/template`**, server-side rendered, via a custom Echo renderer. No SPA.
- **No JavaScript** except, if unavoidable, the long address scans (see below) —
  and then only HTMX (one `<script>` tag), never hand-written JS, never a bundler.
- **CSS:** hand-written, no framework; one small stylesheet.
- **Single static binary**; templates + CSS embedded via `embed.FS`.
- Minimal dependencies; standard library where reasonable.
- Read-only: only ever call non-mutating RPC methods. No wallet/broadcast RPCs.

## Project layout

```
cmd/mempunk/main.go          # entrypoint: load config, build logger, wire Echo
internal/config/             # DONE — see canonical source below
internal/rpc/                # JSON-RPC client + connectivity/chain check
internal/cache/              # LRU for immutable blocks/confirmed txs
internal/explorer/           # view-model layer (all RPC orchestration + formatting)
internal/handlers/           # thin Echo handlers, one per view
internal/render/             # custom Echo renderer + embedded templates
web/templates/               # html/template files (embedded)
web/static/style.css         # the one stylesheet (embedded)
config.example.yaml          # committed; copy to config.yaml (gitignored)
```

Dependency direction: `handlers → explorer → {rpc, cache}`; everything may use
`config` and the injected `*slog.Logger`. No import cycles.

## Networks (source of truth)

mempunk serves one network per instance, set by `network` in config. Every
network-dependent value comes from the `config` network table.

| `network`  | Core `chain` | Default RPC port | bech32 HRP | base58 lead | Min Core |
|------------|--------------|------------------|------------|-------------|----------|
| `mainnet`  | `main`       | 8332             | `bc`       | `1`,`3`     | v25+     |
| `testnet3` | `test`       | 18332            | `tb`       | `m`,`n`,`2` | v25+     |
| `testnet4` | `testnet4`   | 48332            | `tb`       | `m`,`n`,`2` | v28+     |
| `regtest`  | `regtest`    | 18443            | `bcrt`     | `m`,`n`,`2` | v25+     |

- testnet3's chain string is `test`, **not** `testnet3`. testnet4 is `testnet4`.
- testnet3 and testnet4 share address formats; the node's `validateaddress` is
  the source of truth, and only one network is active per instance.
- The `rpc` component verifies `getblockchaininfo.chain` matches the configured
  network and fails fast on mismatch; it may also check
  `getnetworkinfo.version >= MinCoreVersion`.

## Bitcoin Core requirements

Core **v25+** generally; **v28+** to run testnet4. `blockfilterindex=1` required
for address history (`scanblocks`); `txindex` optional (verbosity-3
`getrawtransaction` gives prevouts/fees without it).

## Routes

| Route                          | View        | Notes |
|--------------------------------|-------------|-------|
| `GET /`                        | Overview    | latest N blocks + chain summary |
| `GET /block/:id`               | Block       | `:id` = block hash or height |
| `GET /tx/:txid`                | Transaction | confirmed or mempool |
| `GET /address/:addr`           | Address     | validate + entry to scans |
| `GET /address/:addr/utxos`     | Address scan| UTXO-set scan progress/result |
| `GET /address/:addr/history`   | Address scan| blockfilter-based tx history |
| `GET /node`                    | Node info   | network + chain info |
| `GET /search`                  | Router      | classify input → redirect |
| `GET /static/*`                | static      | embedded CSS |

`/search`: numeric → height; 64-hex → block hash then txid; address-shaped (via
`Params().LooksLikeAddress`) → `/address/:addr`; else 404. Node's
`validateaddress` confirms real validity.

## Views → RPC

- **Overview:** `getblockchaininfo` + walk back N blocks via `getblock` v1
  (`previousblockhash`), fetched with a bounded goroutine fan-out (`errgroup`).
- **Block:** hash or height (`getblockhash` if numeric); `getblock` v2 +
  `getblockstats`; tx-detail loading gated by `Explorer.ShowTxDetails()`.
- **Transaction:** `getrawtransaction` **verbosity 3** (prevouts → input addrs +
  fee without txindex); mempool tx → `getmempoolentry`; optional `gettxout` per
  output for unspent flag; raw JSON in `<details>` (no JS).
- **Address:** `validateaddress`, then the two opt-in slow scans.
- **Node info:** `getnetworkinfo` + `getblockchaininfo`.

## Address scans (the JS-sensitive part)

- **UTXO set:** `scantxoutset ["start", ["addr(<a>)"]]`; poll
  `scantxoutset ["status"]`. Core allows one scan at a time — report progress,
  refuse concurrent scans.
- **History:** `scanblocks ["start", ["addr(<a>)"], <from>]` (needs
  `blockfilterindex`), then fetch those blocks and build a received/spent ledger.
  Disable with a clear message if `blockfilterindex` is off.
- **Progress without JS (preferred):** a status page using
  `<meta http-equiv="refresh" content="2; url=...">`; handler returns the
  refreshing status page (CSS-only bar via inline `width:NN%`) until done, then
  the result page.
- **If smoother feedback wanted:** HTMX `hx-get` + `hx-trigger="every 2s"`
  polling a status fragment, **only** on these two endpoints, isolated.

## UI / cyberpunk design system

Minimal, fast, legible; no images except an inline SVG logo.

- Single centered column, ~1000px max-width, generous whitespace.
- Monospace for data (`ui-monospace, "JetBrains Mono", "Cascadia Code", Menlo,
  monospace`); optional clean sans for prose.
- Palette via CSS custom properties at the top of the stylesheet: near-black bg
  `#0a0e12`, off-white text `#c8d0d8`, neon accents green `#00ff9c` / cyan
  `#00e5ff`, magenta `#ff2bd6` for warnings/spent.
- **Per-network accent:** set a `<body>` class from `Params().AccentClass`
  (`net-mainnet|net-testnet3|net-testnet4|net-regtest`) and show the network as a
  header badge, so the active chain is impossible to confuse.
- No-JS interactivity: `<details>`/`<summary>`, `:target` for tabs, `<table>`.
- Responsive + accessible: semantic HTML, contrast, focus styles. Helpers:
  shorten hashes/addresses, format BTC (8 dp + separators), humanize age, sizes.

## Logging conventions (`log/slog`)

- Build one `*slog.Logger` in `main` from `Log.Level`/`Log.Format` (text or JSON
  handler); inject it into components (don't use the package global in libs).
- Structured attrs, not formatted strings:
  `logger.Info("rpc call", "method", m, "ms", d.Milliseconds())`.
- **Never log secrets.** `Config` implements `slog.LogValuer` and redacts the
  password/cookie, so log it via `slog.Any("config", cfg)`.
- Echo request-logging middleware routes through the same logger.

## Security & ops

- Validate every path param before it reaches RPC (hex length, numeric height,
  address shape). Map RPC errors to friendly messages; never leak node internals.
- Default listen `127.0.0.1`; document a reverse proxy if exposed.
- HTTP server read/write timeouts; cap concurrent expensive scans.

## Component roadmap & status

1. **`config`** — ✅ DONE. Canonical source at the end of this file; write verbatim.
2. **`rpc`** — ✅ DONE. `internal/rpc/`: `Client` with `Call` (30s) and `CallScan`
   (5min), auth via `config.ResolveAuth()` per call, slog debug logging, `Ping`
   verifies chain and warns on low version. Uses `"jsonrpc":"2.0"` (nodes reject
   "1.1"). `cmd/rpccheck/` is a throwaway connectivity tester. Tested with
   `httptest` mocks: success, RPC error, 401, chain mismatch, version warn.
3. **`cache`** — ✅ DONE. Generic `Cache[V any]` backed by `container/list` +
   `sync.Mutex`. capacity ≤ 0 disables. Tests: hit/miss, eviction order,
   update-promotes-to-front, disabled, capacity-1.
4. **`explorer`** — ✅ DONE. `internal/explorer/`: view models (`OverviewVM`,
   `BlockVM`, `TxVM`, `AddressVM`, `NodeVM`), all RPC orchestration via errgroup
   fan-out, fee math, formatting helpers (`BTCToSats`, `FormatBTC`, `FormatAge`,
   `FormatSize`, `FormatVersion`). Cache keyed `"blk:<hash>"`, `"blkstats:<hash>"`,
   `"tx:<txid>"` (confirmed only). `golang.org/x/sync` added for errgroup. Tested
   with httptest mock dispatcher (8 tests).
5. **`render`** — ✅ DONE. `web/` holds `//go:embed templates static; var FS embed.FS`.
   `internal/render/`: `Renderer` (implements `echo.Renderer`), `Page` struct,
   `New(fs.FS)` auto-discovers page templates, `funcMap` (`btc`, `shortenHash`,
   `add`, `not`). `base.html` = full layout (SVG logo, net badge, search, footer).
   `overview.html` = first styled page. `style.css` = cyberpunk palette with CSS
   vars, per-network accent, stat cards, tables, details/raw JSON. 3 tests pass.
6. **`handlers`** — NEXT. One view at a time: overview → block → tx → search/404 →
   node → address basics → address scans.

## Conventions

- `gofmt`/`go vet` clean before every commit; conventional commits.
- Tests live beside code (`_test.go`); prefer table-driven tests.
- `go test ./...` must pass before moving to the next component.

---

# Canonical source — Component 1: `config` (write these files verbatim)

These files are finalized and tested (`go test`, `go vet`, `gofmt` all clean).
Create them exactly as below, then run `go test ./internal/config/...` and
`go vet ./internal/config/...`, and `gofmt -l internal/config` (expect no
output). Also create `config.example.yaml` at the repo root. Then **stop**.

## internal/config/config.go

```go
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
```

## internal/config/load.go

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse unmarshals raw YAML into a Config without applying env overrides,
// defaults, or validation. Useful in tests.
func Parse(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Load reads the YAML config at path and returns a fully resolved Config:
// parsed, overlaid with MEMPUNK_* env vars, defaulted, and validated.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if err := c.applyEnv(); err != nil {
		return nil, err
	}
	c.applyDefaults()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return c, nil
}
```

## internal/config/config_test.go

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

// resolve mimics Load's post-parse pipeline without YAML/file IO.
func resolve(t *testing.T, c *Config) error {
	t.Helper()
	if err := c.applyEnv(); err != nil {
		return err
	}
	c.applyDefaults()
	return c.Validate()
}

func validUserPass() *Config {
	return &Config{
		Network: Testnet4,
		RPC:     RPCConfig{User: "u", Password: "p"},
	}
}

func TestNetworkTable(t *testing.T) {
	cases := []struct {
		net  Network
		chn  string
		port int
		hrp  string
	}{
		{Mainnet, "main", 8332, "bc"},
		{Testnet3, "test", 18332, "tb"}, // chain is "test", not "testnet3"
		{Testnet4, "testnet4", 48332, "tb"},
		{Regtest, "regtest", 18443, "bcrt"},
	}
	for _, tc := range cases {
		p, ok := tc.net.Params()
		if !ok {
			t.Fatalf("%s: not in table", tc.net)
		}
		if p.Chain != tc.chn || p.DefaultRPCPort != tc.port || p.Bech32HRP != tc.hrp {
			t.Errorf("%s: got chain=%q port=%d hrp=%q", tc.net, p.Chain, p.DefaultRPCPort, p.Bech32HRP)
		}
	}
	if Network("signet").Valid() {
		t.Error("signet should be out of scope")
	}
}

func TestDefaultPortPerNetwork(t *testing.T) {
	for net, want := range map[Network]int{
		Mainnet: 8332, Testnet3: 18332, Testnet4: 48332, Regtest: 18443,
	} {
		c := &Config{Network: net, RPC: RPCConfig{User: "u", Password: "p"}}
		if err := resolve(t, c); err != nil {
			t.Fatalf("%s: %v", net, err)
		}
		if c.RPC.Port != want {
			t.Errorf("%s: default port = %d, want %d", net, c.RPC.Port, want)
		}
	}
}

func TestExplicitPortOverridesDefault(t *testing.T) {
	c := &Config{Network: Mainnet, RPC: RPCConfig{Port: 9999, User: "u", Password: "p"}}
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.RPC.Port != 9999 {
		t.Errorf("port = %d, want 9999", c.RPC.Port)
	}
}

func TestDefaults(t *testing.T) {
	c := validUserPass()
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.RPC.Host != "127.0.0.1" || c.Server.Listen != "127.0.0.1:8080" {
		t.Errorf("host/listen defaults wrong: %q %q", c.RPC.Host, c.Server.Listen)
	}
	if c.Log.Level != "info" || c.Log.Format != "text" {
		t.Errorf("log defaults wrong: %q %q", c.Log.Level, c.Log.Format)
	}
	if c.Explorer.LatestBlocks != 12 || c.Explorer.CacheSize != 1000 {
		t.Errorf("explorer defaults wrong: %d %d", c.Explorer.LatestBlocks, c.Explorer.CacheSize)
	}
	if !c.Explorer.ShowTxDetails() {
		t.Error("block_tx_details should default to true")
	}
}

func TestBlockTxDetailsExplicitFalse(t *testing.T) {
	f := false
	c := validUserPass()
	c.Explorer.BlockTxDetails = &f
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.Explorer.ShowTxDetails() {
		t.Error("explicit false must be honored")
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid userpass", func(*Config) {}, false},
		{"valid cookie", func(c *Config) { c.RPC = RPCConfig{CookieFile: "/x/.cookie"} }, false},
		{"bad network", func(c *Config) { c.Network = "testnet5" }, true},
		{"ambiguous auth", func(c *Config) { c.RPC.CookieFile = "/x/.cookie" }, true},
		{"user without password", func(c *Config) { c.RPC.Password = "" }, true},
		{"no auth", func(c *Config) { c.RPC = RPCConfig{} }, true},
		{"bad log level", func(c *Config) { c.Log.Level = "trace" }, true},
		{"bad log format", func(c *Config) { c.Log.Format = "xml" }, true},
		{"bad listen", func(c *Config) { c.Server.Listen = "nope" }, true},
		{"zero latest blocks rejected after explicit -1", func(c *Config) { c.Explorer.LatestBlocks = -1 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validUserPass()
			c.applyDefaults() // start from a valid, fully-defaulted baseline
			tc.mutate(c)
			// re-default fields the mutation may have zeroed (e.g. cookie case)
			c.applyDefaults()
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveAuthUserPass(t *testing.T) {
	c := &Config{RPC: RPCConfig{User: "alice", Password: "secret"}}
	u, p, err := c.ResolveAuth()
	if err != nil || u != "alice" || p != "secret" {
		t.Fatalf("got %q %q err=%v", u, p, err)
	}
	if c.AuthMethod() != "userpass" {
		t.Errorf("AuthMethod = %q", c.AuthMethod())
	}
}

func TestResolveAuthCookie(t *testing.T) {
	dir := t.TempDir()
	cookie := filepath.Join(dir, ".cookie")
	if err := os.WriteFile(cookie, []byte("__cookie__:deadbeef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &Config{RPC: RPCConfig{CookieFile: cookie}}
	u, p, err := c.ResolveAuth()
	if err != nil || u != "__cookie__" || p != "deadbeef" {
		t.Fatalf("got %q %q err=%v", u, p, err)
	}
	if c.AuthMethod() != "cookie" {
		t.Errorf("AuthMethod = %q", c.AuthMethod())
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("MEMPUNK_NETWORK", "regtest")
	t.Setenv("MEMPUNK_RPC_PASSWORD", "fromenv")
	c := &Config{Network: Mainnet, RPC: RPCConfig{User: "u", Password: "fromfile"}}
	if err := resolve(t, c); err != nil {
		t.Fatal(err)
	}
	if c.Network != Regtest {
		t.Errorf("network = %q, want regtest", c.Network)
	}
	if c.RPC.Password != "fromenv" {
		t.Errorf("password = %q, want fromenv", c.RPC.Password)
	}
	if c.RPC.Port != 18443 {
		t.Errorf("port = %d, want regtest default 18443", c.RPC.Port)
	}
}

func TestLooksLikeAddress(t *testing.T) {
	main, _ := Mainnet.Params()
	test, _ := Testnet4.Params()
	if !main.LooksLikeAddress("bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4") {
		t.Error("mainnet bech32 should match")
	}
	if main.LooksLikeAddress("tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxxxxxx") {
		t.Error("tb1 must not match mainnet HRP")
	}
	if !test.LooksLikeAddress("tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxxxxxx") {
		t.Error("testnet bech32 should match")
	}
	if !main.LooksLikeAddress("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa") {
		t.Error("mainnet base58 P2PKH should match")
	}
}
```

## config.example.yaml

```yaml
# mempunk configuration. Copy to config.yaml and edit.
# Any field can be overridden with a MEMPUNK_* env var, e.g. MEMPUNK_RPC_PASSWORD.

# Which Bitcoin network this instance serves. One of:
#   mainnet | testnet3 | testnet4 | regtest
# (Determines the default RPC port and address formats. testnet4 needs Core v28+.)
network: mainnet

rpc:
  host: "127.0.0.1"
  # port: optional. Omit to use the network default
  #   mainnet 8332 | testnet3 18332 | testnet4 48332 | regtest 18443
  # port: 8332

  # Authentication: choose ONE method.
  #   (a) user + password
  user: "bitcoin"
  password: "CHANGEME"
  #   (b) cookie file (mutually exclusive with user/password):
  # cookie_file: "/home/alex/.bitcoin/.cookie"

server:
  listen: "127.0.0.1:8080" # HTTP listen address; put behind nginx if exposed

log:
  level: "info" # debug | info | warn | error
  format: "text" # text | json

explorer:
  latest_blocks: 12 # blocks shown on the overview
  block_tx_details: true # load full tx details on block pages (slower)
  cache_size: 1000 # LRU entries for immutable blocks/txs (0 disables sizing)
```
