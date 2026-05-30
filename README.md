# ⚡ mempunk

A minimal, cyberpunk Bitcoin block explorer written in Go. Served directly from a Bitcoin Core full node over JSON-RPC. No database, no indexer, no JavaScript framework — just your node and a single static binary.

## Features

- **Zero external dependencies** beyond Bitcoin Core — no Electrum, no database
- **Server-side rendered** with `html/template`; no JavaScript
- **Single binary** — templates and CSS are embedded at compile time
- **Per-network UI accent** — mainnet green, testnet3 cyan, testnet4 amber, regtest purple
- **LRU cache** for immutable blocks and confirmed transactions
- **Address scans** with CSS-only progress bar and `<meta>` auto-refresh (no polling JS)
- Supports mainnet, testnet3, testnet4, regtest

## Views

| Route | Description |
|---|---|
| `GET /` | Overview — latest N blocks + chain summary |
| `GET /block/:id` | Block detail by hash or height |
| `GET /tx/:txid` | Transaction — confirmed or mempool, inputs/outputs, raw JSON |
| `GET /address/:addr` | Address validation + links to scans |
| `GET /address/:addr/utxos` | UTXO set scan with live progress bar |
| `GET /address/:addr/history` | Block filter tx history (requires `blockfilterindex`) |
| `GET /node` | Node info — version, connections, chain |
| `GET /search` | Routes height / hash / txid / address to the right view |

## Requirements

- **Go 1.22+**
- **Bitcoin Core v25+** (v28+ for testnet4)
- `blockfilterindex=1` in `bitcoin.conf` for address history scans
- `txindex=1` optional — not required (verbosity-3 `getrawtransaction` provides prevouts)

## Build

```sh
git clone https://github.com/alex27riva/mempunk
cd mempunk
go build -o mempunk ./cmd/mempunk
```

The binary embeds all templates and the stylesheet; no extra files needed at runtime.

## Configuration

```sh
cp config.example.yaml config.yaml
$EDITOR config.yaml
```

Minimal config:

```yaml
network: mainnet

rpc:
  host: "127.0.0.1"
  user: "bitcoin"
  password: "your-rpc-password"
  # or cookie auth:
  # cookie_file: "/home/user/.bitcoin/.cookie"

server:
  listen: "127.0.0.1:8080"
```

Every field can be overridden with a `MEMPUNK_*` environment variable:

```sh
MEMPUNK_RPC_PASSWORD=secret MEMPUNK_LOG_LEVEL=debug ./mempunk
```

### Configuration reference

| Key | Default | Description |
|---|---|---|
| `network` | `mainnet` | `mainnet` \| `testnet3` \| `testnet4` \| `regtest` |
| `rpc.host` | `127.0.0.1` | Bitcoin Core RPC host |
| `rpc.port` | network default | 8332 / 18332 / 48332 / 18443 |
| `rpc.user` / `rpc.password` | — | RPC auth (mutually exclusive with cookie) |
| `rpc.cookie_file` | — | Path to `.cookie` file |
| `server.listen` | `127.0.0.1:8080` | HTTP listen address |
| `log.level` | `info` | `debug` \| `info` \| `warn` \| `error` |
| `log.format` | `text` | `text` \| `json` |
| `explorer.latest_blocks` | `12` | Blocks shown on the overview page |
| `explorer.block_tx_details` | `true` | Load full tx list on block pages |
| `explorer.cache_size` | `1000` | LRU cache entries (0 = disabled) |

## Running

```sh
./mempunk
# or during development:
go run ./cmd/mempunk
```

mempunk verifies connectivity and chain on startup (`getblockchaininfo` + `getnetworkinfo`) and exits fast on misconfiguration.

By default it listens on `127.0.0.1:8080`. To expose it publicly, put it behind a reverse proxy:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

## bitcoin.conf

Recommended settings for full mempunk functionality:

```ini
server=1
rpcuser=bitcoin
rpcpassword=your-password

# required for address history scans
blockfilterindex=1

# optional — mempunk works without it
# txindex=1
```

## Project layout

```
cmd/mempunk/          # entrypoint
internal/
  config/             # YAML config, env overrides, network params
  rpc/                # JSON-RPC client (read-only, 30s / 5min timeouts)
  cache/              # generic LRU cache
  explorer/           # RPC orchestration, view models, address scans
  handlers/           # thin Echo HTTP handlers
  render/             # Echo renderer backed by embedded templates
web/
  templates/          # html/template files (embedded)
  static/style.css    # cyberpunk stylesheet (embedded)
config.example.yaml
```

## Tech stack

- [Go](https://go.dev/) — standard library + minimal deps
- [Echo v4](https://echo.labstack.com/) — HTTP framework
- [`html/template`](https://pkg.go.dev/html/template) — server-side rendering
- [`log/slog`](https://pkg.go.dev/log/slog) — structured logging
- [`embed`](https://pkg.go.dev/embed) — single binary, no external assets

## License

MIT
