package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"github.com/liy0aay/xkcd-search/closers"
	searchpb "github.com/liy0aay/xkcd-search/proto/search"
	"github.com/liy0aay/xkcd-search/search/adapters/db"
	searchgrpc "github.com/liy0aay/xkcd-search/search/adapters/grpc"
	"github.com/liy0aay/xkcd-search/search/adapters/initiator"
	searchnats "github.com/liy0aay/xkcd-search/search/adapters/nats"
	"github.com/liy0aay/xkcd-search/search/adapters/words"
	"github.com/liy0aay/xkcd-search/search/config"
	"github.com/liy0aay/xkcd-search/search/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {

	// config
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	// logger
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	// context for Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer closers.CloseOrLog(storage, log)

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer closers.CloseOrLog(words, log)

	// nats subscriber
	subscriber, err := searchnats.New(log, cfg.BrokerAddress)
	if err != nil {
		return fmt.Errorf("failed to create NATS subscriber: %v", err)
	}
	defer closers.CloseOrLog(subscriber, log)

	// service
	searcher, err := core.NewService(log, storage, words)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	// initiator
	initiator.RunIndexUpdate(ctx, searcher, cfg.IndexTTL, log)

	// nats event index update
	if err := subscriber.RunEventHandlers(ctx,
		func() {
			log.Info("rebuilding index after db update")
			if err := searcher.BuildIndex(ctx); err != nil {
				log.Error("failed to rebuild index", "error", err)
			}
		},
		func() {
			log.Info("clearing index after db drop")
			if err := searcher.BuildIndex(ctx); err != nil {
				log.Error("failed to clear index", "error", err)
			}
		},
	); err != nil {
		return fmt.Errorf("failed to run eventhandlers: %v", err)
	}

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(searcher))
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
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
