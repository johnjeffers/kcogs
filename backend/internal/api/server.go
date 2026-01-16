package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cluster"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/config"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/store"
)

// Server is the HTTP server for the kcogs API
type Server struct {
	server     *http.Server
	config     *config.Config
	calculator *cogs.Calculator
	logger     *slog.Logger
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, calculator *cogs.Calculator, dataProvider store.DataProvider, clusterManager *cluster.Manager, logger *slog.Logger) *Server {
	router := NewRouter(calculator, dataProvider, clusterManager)

	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 5 * time.Minute,
			IdleTimeout:  60 * time.Second,
		},
		config:     cfg,
		calculator: calculator,
		logger:     logger,
	}
}

// Start begins listening for requests
func (s *Server) Start() error {
	s.logger.Info("starting server", "port", s.config.Server.Port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")
	return s.server.Shutdown(ctx)
}
