package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/api"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cluster"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs/algorithms"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/config"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/pricing"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/store"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup algorithm registry
	registry := cogs.NewRegistry()
	if err := registry.Register(&algorithms.RequestBasedAlgorithm{}); err != nil {
		logger.Error("failed to register algorithm", "error", err)
		os.Exit(1)
	}

	// Create calculator
	calculator := cogs.NewCalculator(registry)

	// Create cluster manager
	clusterManager := cluster.NewManager()

	// Create pricing provider
	ctx := context.Background()
	pricingProvider, err := pricing.NewAWSProvider(ctx, cfg.Pricing.RefreshIntervalMinutes)
	if err != nil {
		logger.Error("failed to initialize AWS pricing provider", "error", err)
		os.Exit(1)
	}

	// Create cluster-backed data provider
	dataProvider := store.NewClusterDataProvider(clusterManager, pricingProvider)

	// Create and start server
	server := api.NewServer(cfg, calculator, dataProvider, clusterManager, logger)

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", "error", err)
		}
	}()

	logger.Info("kcogs started", "port", cfg.Server.Port)

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("kcogs stopped")
}
