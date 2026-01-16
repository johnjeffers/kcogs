package cogs

import "context"

// AlgorithmID uniquely identifies a cost allocation algorithm
type AlgorithmID string

const (
	AlgorithmRequestBased    AlgorithmID = "request-based"
	AlgorithmUsageBased      AlgorithmID = "usage-based"
	AlgorithmMaxRequestUsage AlgorithmID = "max-request-usage"
	AlgorithmWeightedBlend   AlgorithmID = "weighted-blend"
)

// Algorithm defines the interface for COGS calculation algorithms.
// All cost allocation strategies must implement this interface.
type Algorithm interface {
	// ID returns the unique identifier for this algorithm
	ID() AlgorithmID

	// Name returns a human-readable name for display
	Name() string

	// Description returns a detailed description of how the algorithm works
	Description() string

	// DefaultParams returns the default parameters for this algorithm
	DefaultParams() AlgorithmParams

	// ValidateParams validates the provided parameters
	ValidateParams(params AlgorithmParams) error

	// Calculate computes costs for the given workloads using node pricing.
	// This is the core method that each algorithm implements differently.
	Calculate(ctx context.Context, input CalculationInput) (*CalculationResult, error)
}

// AlgorithmParams holds algorithm-specific configuration.
// Different algorithms may use different subsets of these parameters.
type AlgorithmParams struct {
	// RequestWeight is used by weighted-blend algorithm (0.0 to 1.0)
	RequestWeight float64 `json:"requestWeight,omitempty"`

	// UsageWeight is used by weighted-blend algorithm (0.0 to 1.0)
	UsageWeight float64 `json:"usageWeight,omitempty"`

	// UsageWindowMinutes defines the time window for usage metrics
	UsageWindowMinutes int `json:"usageWindowMinutes,omitempty"`

	// Custom allows algorithm-specific parameters as key-value pairs
	Custom map[string]any `json:"custom,omitempty"`
}
