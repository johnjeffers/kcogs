package algorithms

import (
	"context"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// CapacityBasedAlgorithm charges pods for the share of a node's capacity they
// reserve. Capacity nobody reserved is reported as idle instead of being spread
// across the pods.
type CapacityBasedAlgorithm struct{}

func (a *CapacityBasedAlgorithm) ID() cogs.AlgorithmID {
	return cogs.AlgorithmCapacityBased
}

func (a *CapacityBasedAlgorithm) Name() string {
	return "Capacity-Based Allocation"
}

func (a *CapacityBasedAlgorithm) Description() string {
	return `Charges each pod for the share of its node's schedulable CPU and memory
that it has requested, so a pod costs the same however full its node is.
Capacity no pod has requested appears as an idle line for each cluster.
Fargate pods pay their full billed size. Ideal for seeing how much of your
spend is unused.`
}

func (a *CapacityBasedAlgorithm) DefaultParams() cogs.AlgorithmParams {
	return cogs.AlgorithmParams{}
}

func (a *CapacityBasedAlgorithm) ValidateParams(params cogs.AlgorithmParams) error {
	return nil // No parameters for capacity-based
}

func (a *CapacityBasedAlgorithm) Calculate(
	ctx context.Context,
	input cogs.CalculationInput,
) (*cogs.CalculationResult, error) {
	return allocate(a.ID(), input, capacityShares), nil
}

// capacityShares charges each pod the average of its CPU and memory share of the
// node's allocatable capacity.
func capacityShares(node cogs.NodeInfo, nodePods []cogs.PodInfo) func(cogs.PodInfo) float64 {
	// A Fargate pod is the only tenant of its billed size, so none of it is idle.
	// Without allocatable figures there is nothing to measure a pod against.
	if node.IsFargate || node.CPUAlloc <= 0 || node.MemAlloc <= 0 {
		return requestShares(node, nodePods)
	}

	// If requests ever exceed allocatable, measure against them so the shares
	// can't add up to more than the whole node
	cpuBase := max(node.CPUAlloc, sumCPURequests(nodePods))
	memBase := max(node.MemAlloc, sumMemoryRequests(nodePods))

	return func(pod cogs.PodInfo) float64 {
		return (float64(pod.CPURequest)/float64(cpuBase) + float64(pod.MemoryRequest)/float64(memBase)) / 2.0
	}
}
