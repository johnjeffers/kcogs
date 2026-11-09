package algorithms

import (
	"time"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// idleTolerance is the smallest leftover node cost, in dollars per hour, that
// counts as idle. Anything less is floating-point noise from adding up shares.
const idleTolerance = 1e-9

// shareFunc prepares one node's allocation and returns a function giving each
// pod's fraction of that node's cost. Any fraction left unclaimed is idle.
type shareFunc func(node cogs.NodeInfo, nodePods []cogs.PodInfo) func(cogs.PodInfo) float64

// allocate splits every node's cost across its pods and rolls the results up by
// workload, namespace, and cluster.
func allocate(id cogs.AlgorithmID, input cogs.CalculationInput, shares shareFunc) *cogs.CalculationResult {
	result := &cogs.CalculationResult{
		AlgorithmID:  id,
		Timestamp:    time.Now(),
		ClusterCosts: make(map[string]*cogs.ClusterCost),
		Params:       input.Params,
	}

	nodesByCluster := groupNodesByCluster(input.Nodes)
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
			share := shares(node, nodePods)

			var allocated cogs.CostValue
			for _, pod := range nodePods {
				podCost := cogs.CostValue(float64(node.HourlyCost) * share(pod))
				allocated += podCost

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

			if idle := node.HourlyCost - allocated; float64(idle) > idleTolerance {
				clusterCost.IdleCost += idle
			}
		}

		result.ClusterCosts[clusterName] = clusterCost
		result.TotalCost += clusterCost.TotalCost
	}

	return result
}

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
