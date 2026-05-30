// rpccheck loads config.yaml and pings the configured Bitcoin Core node.
// Usage: go run ./cmd/rpccheck [path/to/config.yaml]
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alex27riva/mempunk/internal/config"
	"github.com/alex27riva/mempunk/internal/rpc"
)

func main() {
	path := "config.yaml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	log.Info("loaded config", slog.Any("config", cfg))

	client := rpc.New(cfg, log)
	if err := client.Ping(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "ping: %v\n", err)
		os.Exit(1)
	}
}
