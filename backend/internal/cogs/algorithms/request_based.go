package algorithms

import (
	"context"
	"time"

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
	result := &cogs.CalculationResult{
		AlgorithmID:  a.ID(),
		Timestamp:    time.Now(),
		ClusterCosts: make(map[string]*cogs.ClusterCost),
		Params:       input.Params,
	}

	// Group nodes by cluster
	nodesByCluster := groupNodesByCluster(input.Nodes)

	// Group pods by node
	podsByNode := groupPodsByNode(input.Pods)

	for clusterName, nodes := range nodesByCluster {
		clusterCost := &cogs.ClusterCost{
			ClusterName:    clusterName,
			NamespaceCosts: make(map[string]*cogs.NamespaceCost),
			NodeCount:      len(nodes),
		}

		for _, node := range nodes {
			nodePods := podsByNode[node.Name]
			if len(nodePods) == 0 {
				continue
			}

			// Calculate total requests on this node
			totalCPURequests := sumCPURequests(nodePods)
			totalMemRequests := sumMemoryRequests(nodePods)

			// Allocate node cost to pods based on request proportion
			for _, pod := range nodePods {
				var cpuFraction, memFraction float64

				if totalCPURequests > 0 {
					cpuFraction = float64(pod.CPURequest) / float64(totalCPURequests)
				}
				if totalMemRequests > 0 {
					memFraction = float64(pod.MemoryRequest) / float64(totalMemRequests)
				}

				// Average of CPU and memory fractions
				fraction := (cpuFraction + memFraction) / 2.0
				podCost := cogs.CostValue(float64(node.HourlyCost) * fraction)

				// Aggregate into namespace
				ns := getOrCreateNamespace(clusterCost, pod.Namespace)
				ns.TotalCost += podCost

				// Aggregate into workload
				wl := getOrCreateWorkload(ns, pod.WorkloadName, pod.WorkloadKind)
				wl.TotalCost += podCost

				// Add pod cost
				wl.Pods = append(wl.Pods, &cogs.PodCost{
					Name:       pod.Name,
					Cost:       podCost,
					CPURequest: pod.CPURequest,
					MemRequest: pod.MemoryRequest,
				})

				clusterCost.PodCount++
			}

			clusterCost.TotalCost += node.HourlyCost
		}

		result.ClusterCosts[clusterName] = clusterCost
		result.TotalCost += clusterCost.TotalCost
	}

	return result, nil
}

// Helper functions

func groupNodesByCluster(nodes []cogs.NodeInfo) map[string][]cogs.NodeInfo {
	result := make(map[string][]cogs.NodeInfo)
	for _, node := range nodes {
		result[node.ClusterName] = append(result[node.ClusterName], node)
	}
	return result
}

func groupPodsByNode(pods []cogs.PodInfo) map[string][]cogs.PodInfo {
	result := make(map[string][]cogs.PodInfo)
	for _, pod := range pods {
		result[pod.NodeName] = append(result[pod.NodeName], pod)
	}
	return result
}

func sumCPURequests(pods []cogs.PodInfo) int64 {
	var total int64
	for _, pod := range pods {
		total += pod.CPURequest
	}
	return total
}

func sumMemoryRequests(pods []cogs.PodInfo) int64 {
	var total int64
	for _, pod := range pods {
		total += pod.MemoryRequest
	}
	return total
}

func getOrCreateNamespace(cluster *cogs.ClusterCost, ns string) *cogs.NamespaceCost {
	if cluster.NamespaceCosts[ns] == nil {
		cluster.NamespaceCosts[ns] = &cogs.NamespaceCost{
			Namespace:     ns,
			WorkloadCosts: make(map[string]*cogs.WorkloadCost),
		}
	}
	return cluster.NamespaceCosts[ns]
}

func getOrCreateWorkload(ns *cogs.NamespaceCost, name, kind string) *cogs.WorkloadCost {
	key := kind + "/" + name
	if ns.WorkloadCosts[key] == nil {
		ns.WorkloadCosts[key] = &cogs.WorkloadCost{
			Name: name,
			Kind: kind,
		}
	}
	return ns.WorkloadCosts[key]
}
