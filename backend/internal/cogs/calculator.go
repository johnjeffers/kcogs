package cogs

import (
	"context"
	"fmt"
	"time"
)

// Calculator orchestrates cost calculations using the algorithm registry
type Calculator struct {
	registry *Registry
}

// NewCalculator creates a new calculator with the given registry
func NewCalculator(registry *Registry) *Calculator {
	return &Calculator{
		registry: registry,
	}
}

// Calculate computes costs using the specified algorithm (or default if empty)
func (c *Calculator) Calculate(ctx context.Context, algorithmID AlgorithmID, nodes []NodeInfo, pods []PodInfo, params AlgorithmParams) (*CalculationResult, error) {
	var alg Algorithm
	var err error

	if algorithmID == "" {
		alg = c.registry.GetDefault()
		if alg == nil {
			return nil, fmt.Errorf("no default algorithm configured")
		}
	} else {
		alg, err = c.registry.Get(algorithmID)
		if err != nil {
			return nil, fmt.Errorf("getting algorithm: %w", err)
		}
	}

	// Validate params
	if err := alg.ValidateParams(params); err != nil {
		return nil, fmt.Errorf("invalid params for algorithm %s: %w", alg.ID(), err)
	}

	input := CalculationInput{
		Nodes:     nodes,
		Pods:      pods,
		Params:    params,
		Timestamp: time.Now(),
	}

	return alg.Calculate(ctx, input)
}

// ListAlgorithms returns all available algorithms
func (c *Calculator) ListAlgorithms() []AlgorithmInfo {
	return c.registry.List()
}

// GetAlgorithm returns a specific algorithm by ID
func (c *Calculator) GetAlgorithm(id AlgorithmID) (Algorithm, error) {
	return c.registry.Get(id)
}
