package explorer

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// BuildNode fetches network and chain info and builds a NodeVM.
func (e *Explorer) BuildNode(ctx context.Context) (*NodeVM, error) {
	var netRaw, chainRaw json.RawMessage
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		raw, err := e.rpc.Call(gctx, "getnetworkinfo")
		if err != nil {
			return fmt.Errorf("getnetworkinfo: %w", err)
		}
		netRaw = raw
		return nil
	})
	g.Go(func() error {
		raw, err := e.rpc.Call(gctx, "getblockchaininfo")
		if err != nil {
			return fmt.Errorf("getblockchaininfo: %w", err)
		}
		chainRaw = raw
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var ni rpcNetworkInfo
	if err := unmarshal(netRaw, &ni, "networkinfo"); err != nil {
		return nil, err
	}
	var bci rpcBlockchainInfo
	if err := unmarshal(chainRaw, &bci, "blockchaininfo"); err != nil {
		return nil, err
	}

	return &NodeVM{
		Version:         ni.Version,
		VersionFmt:      FormatVersion(ni.Version),
		Subversion:      ni.Subversion,
		ProtocolVersion: ni.ProtocolVersion,
		Connections:     ni.Connections,
		ConnectionsIn:   ni.ConnectionsIn,
		ConnectionsOut:  ni.ConnectionsOut,
		Chain:           bci.Chain,
		Blocks:          bci.Blocks,
		Headers:         bci.Headers,
		BestHash:        bci.BestHash,
		Difficulty:      bci.Difficulty,
		SizeOnDisk:      bci.SizeOnDisk,
		SizeFmt:         FormatSizeI64(bci.SizeOnDisk),
		Pruned:          bci.Pruned,
		IBD:             bci.IBD,
	}, nil
}
