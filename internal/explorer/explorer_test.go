package explorer

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/alex27riva/mempunk/internal/cache"
	"github.com/alex27riva/mempunk/internal/config"
	"github.com/alex27riva/mempunk/internal/rpc"
)

// mockNode dispatches RPC responses by method name. Each method may have
// multiple queued responses consumed in order (for repeated calls).
type mockNode struct {
	mu        sync.Mutex
	responses map[string][]json.RawMessage
	counters  map[string]int
}

func (m *mockNode) queue(method string, result any) {
	raw, _ := json.Marshal(result)
	m.mu.Lock()
	m.responses[method] = append(m.responses[method], raw)
	m.mu.Unlock()
}

func (m *mockNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	m.mu.Lock()
	idx := m.counters[req.Method]
	responses := m.responses[req.Method]
	if idx < len(responses) {
		m.counters[req.Method]++
	}
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if idx >= len(responses) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": nil,
			"error":  map[string]any{"code": -32601, "message": "no mock for: " + req.Method},
			"id":     1,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"result": json.RawMessage(responses[idx]),
		"error":  nil,
		"id":     1,
	})
}

func newMock(t *testing.T) (*mockNode, *Explorer) {
	t.Helper()
	m := &mockNode{
		responses: make(map[string][]json.RawMessage),
		counters:  make(map[string]int),
	}
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)

	host, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	port, _ := strconv.Atoi(portStr)

	cfg := &config.Config{
		Network:  config.Regtest,
		RPC:      config.RPCConfig{Host: host, Port: port, User: "u", Password: "p"},
		Explorer: config.ExplorerConfig{LatestBlocks: 1},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := rpc.New(cfg, log)
	c := cache.New[json.RawMessage](100)
	return m, New(cfg, r, c, log)
}

// --- fixtures ---

var (
	fixBCI = map[string]any{
		"chain": "regtest", "blocks": 100, "headers": 100,
		"bestblockhash":        "aaaa0000000000000000000000000000000000000000000000000000000000aa",
		"difficulty":           4.656542e-10,
		"initialblockdownload": false,
		"verificationprogress": 1.0,
		"size_on_disk":         45000,
		"pruned":               false,
	}

	fixBlockHash = "aaaa0000000000000000000000000000000000000000000000000000000000aa"

	fixBlockV1 = map[string]any{
		"hash": fixBlockHash, "height": 100, "time": 1717000000,
		"nTx": 2, "size": 400, "weight": 1000, "strippedsize": 300,
		"difficulty": 4.656542e-10, "merkleroot": "dead0000000000000000000000000000000000000000000000000000000000ef",
		"previousblockhash": "bbbb0000000000000000000000000000000000000000000000000000000000bb",
		"nonce":             0, "bits": "207fffff", "version": 536870912, "confirmations": 1,
	}

	fixBlockV2 = func() map[string]any {
		b := make(map[string]any)
		for k, v := range fixBlockV1 {
			b[k] = v
		}
		b["tx"] = []map[string]any{
			{
				"txid":    "cafecafe00000000000000000000000000000000000000000000000000cafe01",
				"hash":    "cafecafe00000000000000000000000000000000000000000000000000cafe01",
				"version": 2, "size": 200, "vsize": 141, "weight": 560,
				"locktime": 0, "confirmations": 1,
				"vin": []map[string]any{{"coinbase": "0100", "sequence": 0xffffffff}},
				"vout": []map[string]any{
					{
						"value": 50.0, "n": 0,
						"scriptPubKey": map[string]any{
							"address": "bcrt1qtest00000000000000000000000000000000address",
							"type":    "witness_v0_keyhash",
						},
					},
				},
			},
		}
		return b
	}()

	fixBlockStats = map[string]any{
		"totalfee": 5000, "avgfeerate": 10, "medianfeerate": 8,
	}

	fixTxConfirmed = map[string]any{
		"txid":    "deadbeef00000000000000000000000000000000000000000000000000beef01",
		"hash":    "deadbeef00000000000000000000000000000000000000000000000000beef01",
		"version": 2, "size": 191, "vsize": 110, "weight": 437,
		"locktime": 0, "confirmations": 5,
		"blockhash": fixBlockHash, "blocktime": 1717000000,
		"vin": []map[string]any{
			{
				"txid": "0000000000000000000000000000000000000000000000000000000000000001",
				"vout": 0, "sequence": 0xffffffff,
				"prevout": map[string]any{
					"value": 0.001,
					"scriptPubKey": map[string]any{
						"address": "bcrt1qsender00000000000000000000000000000000",
						"type":    "witness_v0_keyhash",
					},
				},
			},
		},
		"vout": []map[string]any{
			{
				"value": 0.0009, "n": 0,
				"scriptPubKey": map[string]any{
					"address": "bcrt1qrecipient0000000000000000000000000000",
					"type":    "witness_v0_keyhash",
				},
			},
		},
	}

	fixTxMempool = func() map[string]any {
		m := make(map[string]any)
		for k, v := range fixTxConfirmed {
			m[k] = v
		}
		m["confirmations"] = 0
		delete(m, "blockhash")
		delete(m, "blocktime")
		return m
	}()

	fixMempoolEntry = map[string]any{
		"fees":   map[string]any{"base": 0.0001},
		"vsize":  110,
		"weight": 437,
		"time":   1717000100,
		"height": 100,
	}

	fixNetworkInfo = map[string]any{
		"version": 260100, "subversion": "/Satoshi:26.1.0/",
		"protocolversion": 70015,
		"connections":     4, "connections_in": 1, "connections_out": 3,
	}

	fixValidAddr = map[string]any{
		"isvalid":  true,
		"address":  "bcrt1qtest00000000000000000000000000000000address",
		"isscript": false, "iswitness": true,
	}

	fixInvalidAddr = map[string]any{"isvalid": false}

	// gettxout returning non-null means unspent
	fixUTXO = map[string]any{
		"bestblock": fixBlockHash, "confirmations": 5,
		"value":        0.0009,
		"scriptPubKey": map[string]any{"address": "bcrt1qrecipient0000000000000000000000000000", "type": "witness_v0_keyhash"},
	}
)

// --- tests ---

func TestBuildOverview(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getblockchaininfo", fixBCI)
	m.queue("getblock", fixBlockV1)

	vm, err := ex.BuildOverview(context.Background())
	if err != nil {
		t.Fatalf("BuildOverview: %v", err)
	}
	if vm.Chain != "regtest" {
		t.Errorf("Chain = %q, want regtest", vm.Chain)
	}
	if vm.Blocks != 100 {
		t.Errorf("Blocks = %d, want 100", vm.Blocks)
	}
	if len(vm.RecentBlocks) != 1 {
		t.Fatalf("len(RecentBlocks) = %d, want 1", len(vm.RecentBlocks))
	}
	if vm.RecentBlocks[0].TxCount != 2 {
		t.Errorf("TxCount = %d, want 2", vm.RecentBlocks[0].TxCount)
	}
}

func TestBuildBlock(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getblock", fixBlockV2)
	m.queue("getblockstats", fixBlockStats)

	vm, err := ex.BuildBlock(context.Background(), fixBlockHash, 0)
	if err != nil {
		t.Fatalf("BuildBlock: %v", err)
	}
	if vm.Height != 100 {
		t.Errorf("Height = %d, want 100", vm.Height)
	}
	if vm.TotalFeeSats != 5000 {
		t.Errorf("TotalFeeSats = %d, want 5000", vm.TotalFeeSats)
	}
	if len(vm.Txs) != 1 {
		t.Errorf("len(Txs) = %d, want 1 (ShowTxDetails default true)", len(vm.Txs))
	}
}

func TestBuildBlockByHeight(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getblockhash", fixBlockHash)
	m.queue("getblock", fixBlockV2)
	m.queue("getblockstats", fixBlockStats)

	vm, err := ex.BuildBlock(context.Background(), "100", 0)
	if err != nil {
		t.Fatalf("BuildBlock by height: %v", err)
	}
	if vm.Hash != fixBlockHash {
		t.Errorf("Hash = %q, want %q", vm.Hash, fixBlockHash)
	}
}

func TestBuildTxConfirmed(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getrawtransaction", fixTxConfirmed)
	// gettxout for the single output: non-null = unspent
	m.queue("gettxout", fixUTXO)

	vm, err := ex.BuildTx(context.Background(), "deadbeef00000000000000000000000000000000000000000000000000beef01")
	if err != nil {
		t.Fatalf("BuildTx: %v", err)
	}
	if !vm.Confirmed {
		t.Error("Confirmed should be true")
	}
	// fee = 0.001 - 0.0009 = 0.0001 BTC = 10000 sats
	if vm.Fee != 10000 {
		t.Errorf("Fee = %d, want 10000 sats", vm.Fee)
	}
	if len(vm.Outputs) != 1 {
		t.Fatalf("len(Outputs) = %d, want 1", len(vm.Outputs))
	}
	if vm.Outputs[0].Spent {
		t.Error("output should be unspent (gettxout returned non-null)")
	}
}

func TestBuildTxMempool(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getrawtransaction", fixTxMempool)
	m.queue("getmempoolentry", fixMempoolEntry)

	vm, err := ex.BuildTx(context.Background(), "deadbeef00000000000000000000000000000000000000000000000000beef01")
	if err != nil {
		t.Fatalf("BuildTx mempool: %v", err)
	}
	if vm.Confirmed {
		t.Error("Confirmed should be false for mempool tx")
	}
	if vm.MempoolFee != 10000 {
		t.Errorf("MempoolFee = %d, want 10000 sats", vm.MempoolFee)
	}
}

func TestBuildNode(t *testing.T) {
	m, ex := newMock(t)
	m.queue("getnetworkinfo", fixNetworkInfo)
	m.queue("getblockchaininfo", fixBCI)

	vm, err := ex.BuildNode(context.Background())
	if err != nil {
		t.Fatalf("BuildNode: %v", err)
	}
	if vm.VersionFmt != "26.1.0" {
		t.Errorf("VersionFmt = %q, want 26.1.0", vm.VersionFmt)
	}
	if vm.Connections != 4 {
		t.Errorf("Connections = %d, want 4", vm.Connections)
	}
	if vm.Chain != "regtest" {
		t.Errorf("Chain = %q, want regtest", vm.Chain)
	}
}

func TestBuildAddressValid(t *testing.T) {
	m, ex := newMock(t)
	m.queue("validateaddress", fixValidAddr)

	vm, err := ex.BuildAddress(context.Background(), "bcrt1qtest00000000000000000000000000000000address")
	if err != nil {
		t.Fatalf("BuildAddress: %v", err)
	}
	if !vm.IsValid {
		t.Error("IsValid should be true")
	}
	if vm.Network != "regtest" {
		t.Errorf("Network = %q, want regtest", vm.Network)
	}
}

func TestBuildAddressInvalid(t *testing.T) {
	m, ex := newMock(t)
	m.queue("validateaddress", fixInvalidAddr)

	vm, err := ex.BuildAddress(context.Background(), "notanaddress")
	if err != nil {
		t.Fatalf("BuildAddress: %v", err)
	}
	if vm.IsValid {
		t.Error("IsValid should be false")
	}
}
