package explorer

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// BuildOverview fetches the chain summary and the N most recent block headers.
func (e *Explorer) BuildOverview(ctx context.Context) (*OverviewVM, error) {
	raw, err := e.rpc.Call(ctx, "getblockchaininfo")
	if err != nil {
		return nil, fmt.Errorf("getblockchaininfo: %w", err)
	}
	var bci rpcBlockchainInfo
	if err := unmarshal(raw, &bci, "blockchaininfo"); err != nil {
		return nil, err
	}

	n := e.cfg.Explorer.LatestBlocks
	hashes := make([]string, n)
	hashes[0] = bci.BestHash

	// Phase 1: fetch hashes for blocks tip-1 … tip-(n-1) concurrently.
	g1, gctx1 := errgroup.WithContext(ctx)
	for i := 1; i < n; i++ {
		i := i
		height := bci.Blocks - int64(i)
		g1.Go(func() error {
			raw, err := e.rpc.Call(gctx1, "getblockhash", height)
			if err != nil {
				return fmt.Errorf("getblockhash %d: %w", height, err)
			}
			return json.Unmarshal(raw, &hashes[i])
		})
	}
	if err := g1.Wait(); err != nil {
		return nil, err
	}

	// Phase 2: fetch all N block headers (verbosity 1) concurrently.
	blocks := make([]rpcBlockV1, n)
	g2, gctx2 := errgroup.WithContext(ctx)
	for i, hash := range hashes {
		i, hash := i, hash
		g2.Go(func() error {
			raw, err := e.rpc.Call(gctx2, "getblock", hash, 1)
			if err != nil {
				return fmt.Errorf("getblock %s: %w", hash, err)
			}
			return unmarshal(raw, &blocks[i], "block")
		})
	}
	if err := g2.Wait(); err != nil {
		return nil, err
	}

	summaries := make([]BlockSummaryVM, n)
	for i, b := range blocks {
		summaries[i] = BlockSummaryVM{
			Height:  b.Height,
			Hash:    b.Hash,
			Time:    b.Time,
			Age:     FormatAge(b.Time),
			TxCount: b.NTx,
			Size:    b.Size,
			Weight:  b.Weight,
			SizeFmt: FormatSize(b.Size),
		}
	}

	return &OverviewVM{
		Chain:        bci.Chain,
		Blocks:       bci.Blocks,
		Headers:      bci.Headers,
		BestHash:     bci.BestHash,
		Difficulty:   bci.Difficulty,
		IBD:          bci.IBD,
		RecentBlocks: summaries,
	}, nil
}
