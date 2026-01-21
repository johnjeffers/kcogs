package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/api/handlers"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cluster"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/store"
)

// NewRouter creates and configures the HTTP router
func NewRouter(calculator *cogs.Calculator, dataProvider store.DataProvider, clusterManager *cluster.Manager) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	algorithmsHandler := handlers.NewAlgorithmsHandler(calculator)
	costsHandler := handlers.NewCostsHandler(calculator, dataProvider)
	clustersHandler := handlers.NewClustersHandler(clusterManager)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health
		r.Get("/health", healthHandler.ServeHTTP)

		// Kubeconfig
		r.Get("/kubeconfig", clustersHandler.GetKubeconfig)
		r.Post("/kubeconfig", clustersHandler.UploadKubeconfig)

		// Contexts
		r.Delete("/contexts/{name}", clustersHandler.RemoveContext)

		// Clusters
		r.Get("/clusters", clustersHandler.ListClusters)
		r.Post("/clusters", clustersHandler.AddCluster)
		r.Delete("/clusters/{name}", clustersHandler.RemoveCluster)

		// Algorithms
		r.Get("/algorithms", algorithmsHandler.List)
		r.Get("/algorithms/{id}", algorithmsHandler.Get)

		// Costs
		r.Get("/costs", costsHandler.GetCosts)
		r.Get("/costs/clusters", costsHandler.GetClusterCosts)
		r.Get("/costs/namespaces", costsHandler.GetNamespaceCosts)
	})

	// Serve embedded frontend for all other routes
	r.Handle("/*", NewSPAHandler())

	return r
}
