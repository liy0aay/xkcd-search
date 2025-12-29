package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/liy0aay/xkcd-search/api/adapters/aaa"
	"github.com/liy0aay/xkcd-search/api/adapters/explainxkcd"
	"github.com/liy0aay/xkcd-search/api/adapters/rest"
	"github.com/liy0aay/xkcd-search/api/adapters/rest/middleware"
	"github.com/liy0aay/xkcd-search/api/adapters/search"
	"github.com/liy0aay/xkcd-search/api/adapters/update"
	"github.com/liy0aay/xkcd-search/api/adapters/words"
	"github.com/liy0aay/xkcd-search/api/config"
	"github.com/liy0aay/xkcd-search/api/core"
	"github.com/liy0aay/xkcd-search/closers"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("failed to run service", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init words adapter: %v", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init update adapter: %v", err)
	}
	defer closers.CloseOrLog(updateClient, log)

	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init search adapter: %v", err)
	}
	defer closers.CloseOrLog(searchClient, log)

	explainClient, err := explainxkcd.NewClient(cfg.ExplainXKCDURL, 5*time.Second, log)
	if err != nil {
		return fmt.Errorf("cannot init ExplainXKCD client: %v", err)
	}
	defer closers.CloseOrLog(explainClient, log)

	authSrv, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		return fmt.Errorf("cannot init authenticator: %v", err)
	}

	mux := http.NewServeMux()

	mux.Handle("POST /api/login", rest.NewLoginHandler(log, authSrv))
	mux.Handle("POST /api/refresh", rest.NewRefreshTokenHandler(log, authSrv))
	mux.Handle("POST /api/logout", rest.NewLogoutHandler(log))

	mux.Handle("GET /api/db/stats",
		middleware.Auth(
			rest.NewUpdateStatsHandler(log, updateClient), authSrv,
		),
	)
	mux.Handle("GET /api/db/status",
		middleware.Auth(
			rest.NewUpdateStatusHandler(log, updateClient), authSrv,
		),
	)
	mux.Handle("GET /api/explain", rest.NewExplainHandler(log, explainClient))

	// authorize update/delete
	mux.Handle("POST /api/db/update",
		middleware.Auth(
			rest.NewUpdateHandler(log, updateClient), authSrv,
		),
	)
	mux.Handle("DELETE /api/db",
		middleware.Auth(
			rest.NewDropHandler(log, updateClient), authSrv,
		),
	)

	// restrict
	mux.Handle("GET /api/search",
		middleware.Concurrency(
			rest.NewSearchHandler(log, searchClient), cfg.SearchConcurrency,
		),
	)
	mux.Handle("GET /api/isearch",
		middleware.Rate(
			rest.NewSearchIndexHandler(log, searchClient), cfg.SearchRate,
		),
	)

	mux.Handle("GET /api/ping", rest.NewPingHandler(
		log,
		map[string]core.Pinger{
			"words":  wordsClient,
			"update": updateClient,
			"search": searchClient,
		}),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %v", err)
		}
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
