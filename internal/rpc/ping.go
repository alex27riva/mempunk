package rpc

import (
	"context"
	"encoding/json"
	"fmt"
)

type blockchainInfo struct {
	Chain string `json:"chain"`
}

type networkInfo struct {
	Version int `json:"version"`
}

// Ping verifies connectivity and chain match, and warns if node version is
// below the minimum required for the configured network.
func (c *Client) Ping(ctx context.Context) error {
	raw, err := c.Call(ctx, "getblockchaininfo")
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	var bci blockchainInfo
	if err := json.Unmarshal(raw, &bci); err != nil {
		return fmt.Errorf("ping: decode blockchaininfo: %w", err)
	}
	want := c.cfg.Params().Chain
	if bci.Chain != want {
		return fmt.Errorf("chain mismatch: node=%q config=%q", bci.Chain, want)
	}

	raw, err = c.Call(ctx, "getnetworkinfo")
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	var ni networkInfo
	if err := json.Unmarshal(raw, &ni); err != nil {
		return fmt.Errorf("ping: decode networkinfo: %w", err)
	}
	if ni.Version < c.cfg.Params().MinCoreVersion {
		c.log.Warn("bitcoin core below minimum version",
			"version", ni.Version,
			"minimum", c.cfg.Params().MinCoreVersion,
			"network", c.cfg.Network)
	}

	c.log.Info("rpc ping ok", "chain", bci.Chain, "version", ni.Version)
	return nil
}
