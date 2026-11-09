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
			// A node is billed whether or not anything is scheduled on it
			clusterCost.TotalCost += node.HourlyCost

			nodePods := podsByNode[nodeKey(node.ClusterName, node.Name)]
			if len(nodePods) == 0 {
				continue
			}

			// Calculate total requests on this node
			totalCPURequests := sumCPURequests(nodePods)
			totalMemRequests := sumMemoryRequests(nodePods)

			// Allocate node cost to pods based on request proportion
			for _, pod := range nodePods {
				fraction := podFraction(pod, totalCPURequests, totalMemRequests, len(nodePods))
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

// nodeKey identifies a node across clusters, since node names are only unique
// within one cluster.
func nodeKey(clusterName, nodeName string) string {
	return clusterName + "/" + nodeName
}

func groupPodsByNode(pods []cogs.PodInfo) map[string][]cogs.PodInfo {
	result := make(map[string][]cogs.PodInfo)
	for _, pod := range pods {
		key := nodeKey(pod.ClusterName, pod.NodeName)
		result[key] = append(result[key], pod)
	}
	return result
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
