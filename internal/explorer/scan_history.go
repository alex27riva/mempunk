package explorer

import (
	"context"
	"errors"
	"fmt"

	"github.com/alex27riva/mempunk/internal/rpc"
	"golang.org/x/sync/errgroup"
)

// HistoryScanVM is the view model for GET /address/:addr/history.
type HistoryScanVM struct {
	Addr     string
	Scanning bool
	Done     bool
	ErrMsg   string
	NoIndex  bool // blockfilterindex not enabled on this node
	Txs      []HistoryTxVM
}

// HistoryTxVM is one transaction row in the history list.
type HistoryTxVM struct {
	Txid      string
	BlockHash string
	Height    int64
	Time      int64
	Age       string
}

type rpcScanBlocks struct {
	FromHeight     int64    `json:"from_height"`
	ToHeight       int64    `json:"to_height"`
	RelevantBlocks []string `json:"relevant_blocks"`
	Completed      bool     `json:"completed"`
}

// PollHistoryScan returns the current history scan state for addr.
// Starts a background goroutine on first call for a new addr.
func (e *Explorer) PollHistoryScan(ctx context.Context, addr string) (*HistoryScanVM, error) {
	e.histMu.Lock()
	job := e.histJob

	if job != nil && job.done && job.addr == addr {
		e.histMu.Unlock()
		if job.err != nil {
			return &HistoryScanVM{Addr: addr, Done: true, ErrMsg: job.err.Error()}, nil
		}
		return job.val, nil
	}

	if job != nil && !job.done && job.addr == addr {
		e.histMu.Unlock()
		return &HistoryScanVM{Addr: addr, Scanning: true}, nil
	}

	newJob := &scanJob[*HistoryScanVM]{addr: addr}
	e.histJob = newJob
	e.histMu.Unlock()

	go e.runHistoryScan(addr, newJob)
	return &HistoryScanVM{Addr: addr, Scanning: true}, nil
}

func (e *Explorer) runHistoryScan(addr string, job *scanJob[*HistoryScanVM]) {
	vm, err := e.doHistoryScan(context.Background(), addr)

	e.histMu.Lock()
	defer e.histMu.Unlock()
	job.done = true
	if err != nil {
		job.err = err
	} else {
		job.val = vm
	}
}

func (e *Explorer) doHistoryScan(ctx context.Context, addr string) (*HistoryScanVM, error) {
	raw, err := e.rpc.CallScan(ctx, "scanblocks", "start",
		[]string{"addr(" + addr + ")"}, 0)
	if err != nil {
		var rpcErr *rpc.RPCError
		if errors.As(err, &rpcErr) {
			// -8 or -1: "Index is not enabled for filtertype basic"
			if rpcErr.Code == -8 || rpcErr.Code == -1 {
				return &HistoryScanVM{Addr: addr, Done: true, NoIndex: true}, nil
			}
		}
		return nil, fmt.Errorf("scanblocks: %w", err)
	}

	var sb rpcScanBlocks
	if err := unmarshal(raw, &sb, "scanblocks"); err != nil {
		return nil, err
	}

	if len(sb.RelevantBlocks) == 0 {
		return &HistoryScanVM{Addr: addr, Done: true}, nil
	}

	// Fetch all relevant blocks concurrently, bounded at 8 in-flight.
	blocks := make([]rpcBlockV2, len(sb.RelevantBlocks))
	sem := make(chan struct{}, 8)
	g, gctx := errgroup.WithContext(ctx)
	for i, hash := range sb.RelevantBlocks {
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			braw, err := e.rpc.Call(gctx, "getblock", hash, 2)
			if err != nil {
				return fmt.Errorf("getblock %s: %w", hash, err)
			}
			return unmarshal(braw, &blocks[i], "block")
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Collect txs that include the address in any output.
	var txs []HistoryTxVM
	for _, b := range blocks {
		for _, tx := range b.Tx {
			for _, vout := range tx.Vout {
				if scriptAddress(vout.ScriptPubKey) == addr {
					txs = append(txs, HistoryTxVM{
						Txid:      tx.Txid,
						BlockHash: b.Hash,
						Height:    b.Height,
						Time:      b.Time,
						Age:       FormatAge(b.Time),
					})
					break // one match per tx is enough
				}
			}
		}
	}

	return &HistoryScanVM{Addr: addr, Done: true, Txs: txs}, nil
}
