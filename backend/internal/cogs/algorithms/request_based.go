package algorithms

import (
	"context"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// RequestBasedAlgorithm allocates costs based on resource requests.
// Pods pay for what they request, regardless of actual usage.
type RequestBasedAlgorithm struct{}

func (a *RequestBasedAlgorithm) ID() cogs.AlgorithmID {
	return cogs.AlgorithmRequestBased
}

func (a *RequestBasedAlgorithm) Name() string {
	return "Request-Based Allocation"
}

func (a *RequestBasedAlgorithm) Description() string {
	return `Allocates node costs to pods based on their resource requests.
Each pod pays for the resources it has requested, regardless of actual usage.
This reflects the guaranteed capacity reserved for each workload and matches
how Kubernetes schedules pods. Ideal for chargeback scenarios where teams
should pay for reserved capacity.`
}

func (a *RequestBasedAlgorithm) DefaultParams() cogs.AlgorithmParams {
	return cogs.AlgorithmParams{}
}

func (a *RequestBasedAlgorithm) ValidateParams(params cogs.AlgorithmParams) error {
	return nil // No parameters for request-based
}

func (a *RequestBasedAlgorithm) Calculate(
	ctx context.Context,
	input cogs.CalculationInput,
) (*cogs.CalculationResult, error) {
	return allocate(a.ID(), input, requestShares), nil
}

// requestShares splits a node's whole cost among the pods on it, in proportion
// to what each one requested.
func requestShares(_ cogs.NodeInfo, nodePods []cogs.PodInfo) func(cogs.PodInfo) float64 {
	totalCPU := sumCPURequests(nodePods)
	totalMem := sumMemoryRequests(nodePods)

	return func(pod cogs.PodInfo) float64 {
		return podFraction(pod, totalCPU, totalMem, len(nodePods))
	}
}

// podFraction returns the share of a node's cost owed by one pod: the average of
// its CPU and memory shares. A resource nobody on the node requested is left out
// of the average, so the shares of all pods on a node always add up to 1.
func podFraction(pod cogs.PodInfo, totalCPU, totalMem int64, podCount int) float64 {
	switch {
	case totalCPU > 0 && totalMem > 0:
		return (float64(pod.CPURequest)/float64(totalCPU) + float64(pod.MemoryRequest)/float64(totalMem)) / 2.0
	case totalCPU > 0:
		return float64(pod.CPURequest) / float64(totalCPU)
	case totalMem > 0:
		return float64(pod.MemoryRequest) / float64(totalMem)
	default:
		return 1.0 / float64(podCount)
	}
}
