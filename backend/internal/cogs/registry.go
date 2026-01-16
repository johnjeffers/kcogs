package cogs

import (
	"fmt"
	"sync"
)

// Registry manages available cost allocation algorithms.
// It provides thread-safe registration and retrieval of algorithms.
type Registry struct {
	mu         sync.RWMutex
	algorithms map[AlgorithmID]Algorithm
	defaultID  AlgorithmID
}

// NewRegistry creates a new algorithm registry
func NewRegistry() *Registry {
	return &Registry{
		algorithms: make(map[AlgorithmID]Algorithm),
		defaultID:  AlgorithmRequestBased,
	}
}

// Register adds an algorithm to the registry.
// Returns an error if an algorithm with the same ID already exists.
func (r *Registry) Register(alg Algorithm) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.algorithms[alg.ID()]; exists {
		return fmt.Errorf("algorithm %s already registered", alg.ID())
	}

	r.algorithms[alg.ID()] = alg
	return nil
}

// Get retrieves an algorithm by ID
func (r *Registry) Get(id AlgorithmID) (Algorithm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	alg, exists := r.algorithms[id]
	if !exists {
		return nil, fmt.Errorf("algorithm %s not found", id)
	}

	return alg, nil
}

// GetDefault returns the default algorithm
func (r *Registry) GetDefault() Algorithm {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.algorithms[r.defaultID]
}

// SetDefault changes the default algorithm
func (r *Registry) SetDefault(id AlgorithmID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.algorithms[id]; !exists {
		return fmt.Errorf("algorithm %s not found", id)
	}

	r.defaultID = id
	return nil
}

// List returns metadata for all registered algorithms
func (r *Registry) List() []AlgorithmInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]AlgorithmInfo, 0, len(r.algorithms))
	for _, alg := range r.algorithms {
		infos = append(infos, AlgorithmInfo{
			ID:            alg.ID(),
			Name:          alg.Name(),
			Description:   alg.Description(),
			DefaultParams: alg.DefaultParams(),
			IsDefault:     alg.ID() == r.defaultID,
		})
	}
	return infos
}

// DefaultID returns the ID of the default algorithm
func (r *Registry) DefaultID() AlgorithmID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultID
}
