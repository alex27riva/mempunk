# CLAUDE.md — mempunk

Architecture and development reference for mempunk: a minimal, cyberpunk
Bitcoin block explorer in Go, served directly from a Bitcoin Core full node
over JSON-RPC. Server-side rendered, JavaScript-free wherever possible.

Module path: `github.com/alex27riva/mempunk`.

---

## Tech stack & constraints

- **Go** (latest stable), idiomatic, `gofmt`/`go vet` clean.
- **Echo v4** for HTTP.
- **`log/slog`** for logging — no third-party logging libraries.
- **`html/template`**, server-side rendered, via a custom Echo renderer. No SPA.
- **No JavaScript** — address scan progress uses `<meta http-equiv="refresh">` and a CSS-only bar.
- **CSS:** hand-written, no framework; one small stylesheet.
- **Single static binary**; templates + CSS embedded via `embed.FS`.
- Minimal dependencies; standard library where reasonable.
- Read-only: only ever call non-mutating RPC methods. No wallet/broadcast RPCs.

## Project layout

```
cmd/mempunk/main.go          # entrypoint: load config, build logger, wire Echo
cmd/rpccheck/main.go         # standalone RPC connectivity tester
internal/config/             # YAML config, env overrides, network params table
internal/rpc/                # JSON-RPC client + connectivity/chain check
internal/cache/              # generic LRU for immutable blocks/confirmed txs
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

- testnet3's chain string is `test`, **not** `testnet3`.
- The `rpc` package verifies `getblockchaininfo.chain` matches the configured
  network and fails fast on mismatch; warns if `getnetworkinfo.version` is below
  `MinCoreVersion`.

## Bitcoin Core requirements

Core **v25+** generally; **v28+** for testnet4. `blockfilterindex=1` required
for address history (`scanblocks`); `txindex` optional (verbosity-3
`getrawtransaction` gives prevouts and fees without it).

## Routes

| Route                          | View         | Notes |
|--------------------------------|--------------|-------|
| `GET /`                        | Overview     | latest N blocks + chain summary |
| `GET /block/:id`               | Block        | `:id` = block hash or height |
| `GET /tx/:txid`                | Transaction  | confirmed or mempool |
| `GET /address/:addr`           | Address      | validate + entry to scans |
| `GET /address/:addr/utxos`     | UTXO scan    | scantxoutset progress/result |
| `GET /address/:addr/history`   | History scan | scanblocks-based tx history |
| `GET /node`                    | Node info    | network + chain info |
| `GET /search`                  | Router       | classify input → redirect |
| `GET /static/*`                | Static       | embedded CSS |

`/search`: numeric → height; 64-hex → probes `getblockheader` then
`getrawtransaction` to distinguish block vs tx; address-shaped (via
`NetworkParams.LooksLikeAddress`) → `/address/:addr`; else 404.

## Views → RPC calls

- **Overview:** `getblockchaininfo` + concurrent `getblockhash`/`getblock` v1
  fan-out via `errgroup`.
- **Block:** `getblockhash` if numeric id; parallel `getblock` v2 + `getblockstats`.
- **Transaction:** `getrawtransaction` verbosity 3 (prevouts without txindex);
  `getmempoolentry` for mempool; `gettxout` per output for unspent flag.
- **Address:** `validateaddress` only; scans are separate handlers.
- **Node:** parallel `getnetworkinfo` + `getblockchaininfo`.
- **UTXO scan:** `scantxoutset "start"` in background goroutine; polls
  `scantxoutset "status"` (returns 0–100, normalized to 0–1 for the progress
  bar). Checks status before starting to handle server restarts gracefully.
- **History scan:** `scanblocks "start"` in background goroutine; fetches
  relevant blocks via bounded 8-concurrent `getblock` v2 errgroup; extracts txs
  where address appears in an output. Shows friendly message if
  `blockfilterindex` is off.

## Address scan progress (no JS)

Status page with `<meta http-equiv="refresh" content="2; url=...">` auto-reloads
every 2 seconds. CSS progress bar uses inline `style="width: N%"`. The same
handler returns the status page while scanning and the result page when done.
The base template exposes `{{block "head" .}}` for per-page `<head>` injection.

## UI / cyberpunk design system

Minimal, fast, legible.

- Single centered column, ~1000px max-width.
- Monospace for data (`ui-monospace, "JetBrains Mono", "Cascadia Code", Menlo, monospace`).
- CSS custom properties: `--bg #0a0e12`, `--text #c8d0d8`, `--accent #f59e0b`,
  `--accent-2 #00e5ff`, `--warn #ff2bd6`.
- Per-network accent via `<body class="{{ .AccentClass }}">`:
  mainnet orange · testnet3 green · testnet4 purple · regtest cyan.
- No-JS interactivity: `<details>`/`<summary>`, `:target`, `<table>`.
- Template funcs: `btc` (sats→BTC string), `shortenHash`, `pct` (float 0–1 → "42%"),
  `add`, `not`.

## Logging (`log/slog`)

- One `*slog.Logger` built in `main`, injected into all components.
- Structured attrs: `logger.Info("rpc call", "method", m, "ms", d.Milliseconds())`.
- `Config` implements `slog.LogValuer` and redacts the RPC password/cookie.

## Security

- Validate every path param before it reaches RPC (hex length, numeric height,
  address charset + length). Map RPC errors to friendly messages; never expose
  node internals.
- Default listen `127.0.0.1`; put behind a reverse proxy if exposed publicly.
- RPC client: 30s timeout for normal calls, 5-minute timeout for scan calls.

## Conventions

- `gofmt`/`go vet` clean; conventional commits.
- Tests beside code (`_test.go`); table-driven where applicable.
- `go test ./...` must pass before committing.
