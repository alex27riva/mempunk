package explorer

import (
	"context"
	"encoding/json"
	"fmt"
)

// UTXOScanVM is the view model for GET /address/:addr/utxos.
// Scanning=true means a scan is in progress; Done=true means it has finished.
type UTXOScanVM struct {
	Addr      string
	Scanning  bool
	BusyAddr  string  // non-empty when Core is scanning a different address
	Progress  float64 // 0.0–1.0 while scanning
	Done      bool
	ErrMsg    string
	UTXOs     []UTXOEntryVM
	TotalSats int64
	TotalBTC  string
}

// UTXOEntryVM is a single unspent output.
type UTXOEntryVM struct {
	Txid    string
	Vout    uint32
	Height  int64
	AmtSats int64
	AmtBTC  string
}

type rpcScanTxOut struct {
	Success     bool           `json:"success"`
	Unspents    []rpcUTXOEntry `json:"unspents"`
	TotalAmount float64        `json:"total_amount"`
}

type rpcUTXOEntry struct {
	Txid   string  `json:"txid"`
	Vout   uint32  `json:"vout"`
	Amount float64 `json:"amount"`
	Height int64   `json:"height"`
}

type rpcScanStatus struct {
	Progress float64 `json:"progress"`
}

// PollUTXOScan returns the current scan state for addr.
// On the first call for a new addr it starts a background scan goroutine.
func (e *Explorer) PollUTXOScan(ctx context.Context, addr string) (*UTXOScanVM, error) {
	e.utxoMu.Lock()
	job := e.utxoJob

	// Cached result for this addr.
	if job != nil && job.done && job.addr == addr {
		e.utxoMu.Unlock()
		if job.err != nil {
			return &UTXOScanVM{Addr: addr, Done: true, ErrMsg: job.err.Error()}, nil
		}
		return job.val, nil
	}

	// Scan in progress for this addr — poll RPC for progress.
	if job != nil && !job.done && job.addr == addr {
		e.utxoMu.Unlock()
		return e.pollUTXOStatus(ctx, addr)
	}

	// Core busy with a different addr.
	if job != nil && !job.done {
		busy := job.addr
		e.utxoMu.Unlock()
		return &UTXOScanVM{Addr: addr, Scanning: true, BusyAddr: busy}, nil
	}

	// No tracked job. Release lock and check if Core is already scanning
	// (e.g. leftover from a server restart).
	e.utxoMu.Unlock()

	statusRaw, err := e.rpc.Call(ctx, "scantxoutset", "status")
	if err == nil && string(statusRaw) != "false" && string(statusRaw) != "null" {
		// Core has an untracked scan in progress. Show busy; meta-refresh will
		// retry until Core is free and we can start our own scan.
		return &UTXOScanVM{Addr: addr, Scanning: true, BusyAddr: "(node)"}, nil
	}

	// Core is free. Re-acquire lock and start a new scan. Double-check in case
	// a concurrent request already started one.
	e.utxoMu.Lock()
	if e.utxoJob != nil && !e.utxoJob.done {
		busy := e.utxoJob.addr
		e.utxoMu.Unlock()
		return &UTXOScanVM{Addr: addr, Scanning: true, BusyAddr: busy}, nil
	}
	newJob := &scanJob[*UTXOScanVM]{addr: addr}
	e.utxoJob = newJob
	e.utxoMu.Unlock()

	go e.runUTXOScan(addr, newJob)
	return &UTXOScanVM{Addr: addr, Scanning: true}, nil
}

func (e *Explorer) pollUTXOStatus(ctx context.Context, addr string) (*UTXOScanVM, error) {
	raw, err := e.rpc.Call(ctx, "scantxoutset", "status")
	if err != nil || string(raw) == "false" || string(raw) == "null" {
		return &UTXOScanVM{Addr: addr, Scanning: true}, nil
	}
	var st rpcScanStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return &UTXOScanVM{Addr: addr, Scanning: true}, nil
	}
	return &UTXOScanVM{Addr: addr, Scanning: true, Progress: st.Progress / 100.0}, nil
}

func (e *Explorer) runUTXOScan(addr string, job *scanJob[*UTXOScanVM]) {
	raw, err := e.rpc.CallScan(context.Background(),
		"scantxoutset", "start", []string{"addr(" + addr + ")"})

	e.utxoMu.Lock()
	defer e.utxoMu.Unlock()
	job.done = true

	if err != nil {
		// -8 means Core is busy with another scan (race between status check and
		// start). The meta-refresh will retry and the status check will catch it.
		job.err = fmt.Errorf("scantxoutset: %w", err)
		return
	}

	var res rpcScanTxOut
	if err := json.Unmarshal(raw, &res); err != nil {
		job.err = fmt.Errorf("decode scantxoutset result: %w", err)
		return
	}

	utxos := make([]UTXOEntryVM, len(res.Unspents))
	for i, u := range res.Unspents {
		sats := BTCToSats(u.Amount)
		utxos[i] = UTXOEntryVM{
			Txid:    u.Txid,
			Vout:    u.Vout,
			Height:  u.Height,
			AmtSats: sats,
			AmtBTC:  FormatBTC(sats),
		}
	}
	totalSats := BTCToSats(res.TotalAmount)
	job.val = &UTXOScanVM{
		Addr:      addr,
		Done:      true,
		UTXOs:     utxos,
		TotalSats: totalSats,
		TotalBTC:  FormatBTC(totalSats),
	}
}
