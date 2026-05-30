package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alex27riva/mempunk/internal/cache"
	"github.com/alex27riva/mempunk/internal/config"
	"github.com/alex27riva/mempunk/internal/explorer"
	"github.com/alex27riva/mempunk/internal/handlers"
	"github.com/alex27riva/mempunk/internal/render"
	"github.com/alex27riva/mempunk/internal/rpc"
	"github.com/alex27riva/mempunk/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mempunk:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := buildLogger(cfg)
	log.Info("starting mempunk", slog.Any("config", cfg))

	rpcClient := rpc.New(cfg, log)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := rpcClient.Ping(ctx); err != nil {
		return fmt.Errorf("node connectivity check failed: %w", err)
	}

	lruCache := cache.New[json.RawMessage](cfg.Explorer.CacheSize)
	ex := explorer.New(cfg, rpcClient, lruCache, log)

	renderer, err := render.New(web.FS)
	if err != nil {
		return fmt.Errorf("build renderer: %w", err)
	}

	h := handlers.New(ex, cfg, log)

	e := echo.New()
	e.HideBanner = true
	e.Renderer = renderer

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod: true,
		LogURI:    true,
		LogStatus: true,
		LogError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"err", v.Error,
			)
			return nil
		},
	}))

	e.Static("/static", "") // served from embed via custom handler below
	e.GET("/static/*", staticHandler())

	e.GET("/", h.Overview)
	e.GET("/block/:id", h.Block)
	e.GET("/tx/:txid", h.Tx)
	e.GET("/search", h.Search)

	e.HTTPErrorHandler = h.ErrorHandler

	log.Info("listening", "addr", cfg.Server.Listen)
	return e.Start(cfg.Server.Listen)
}

// staticHandler serves embedded static assets from web.FS.
func staticHandler() echo.HandlerFunc {
	fs := http.FS(web.FS)
	fileServer := http.FileServer(fs)
	return func(c echo.Context) error {
		// Rewrite /static/foo → static/foo so http.FileServer finds it in embed.FS
		req := c.Request()
		req.URL.Path = req.URL.Path[1:] // strip leading /
		fileServer.ServeHTTP(c.Response(), req)
		return nil
	}
}

func buildLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cfg.Log.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}
	return slog.New(handler)
}
