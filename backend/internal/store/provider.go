package store

import (
	"context"
	"log/slog"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cluster"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/pricing"
)

// DataProvider interface for fetching cluster data
type DataProvider interface {
	GetNodes(ctx context.Context, clusters []string) ([]cogs.NodeInfo, error)
	GetPods(ctx context.Context, clusters []string, namespaces []string) ([]cogs.PodInfo, error)
	// RecalculateFargateCosts updates Fargate node costs based on actual pod requests
	RecalculateFargateCosts(ctx context.Context, nodes []cogs.NodeInfo, pods []cogs.PodInfo) []cogs.NodeInfo
}

// ClusterDataProvider implements DataProvider using the cluster manager
type ClusterDataProvider struct {
	manager        *cluster.Manager
	pricingProvider pricing.Provider
}

// NewClusterDataProvider creates a new cluster-backed data provider
func NewClusterDataProvider(manager *cluster.Manager, pricingProvider pricing.Provider) *ClusterDataProvider {
	return &ClusterDataProvider{
		manager:        manager,
		pricingProvider: pricingProvider,
	}
}

// GetNodes fetches nodes from connected clusters and enriches with pricing
func (p *ClusterDataProvider) GetNodes(ctx context.Context, clusters []string) ([]cogs.NodeInfo, error) {
	allNodes, err := p.manager.GetAllNodes(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich nodes with pricing data
	if p.pricingProvider != nil {
		// Cache Fargate pricing by region to avoid repeated API calls
		fargatePriceCache := make(map[string]*pricing.FargatePricing)

		for i := range allNodes {
			node := &allNodes[i]
			if node.Region == "" {
				continue
			}

			if node.IsFargate {
				// Get Fargate pricing for this region
				fargatePricing, ok := fargatePriceCache[node.Region]
				if !ok {
					var err error
					fargatePricing, err = p.pricingProvider.GetFargatePricing(ctx, node.Region)
					if err != nil {
						slog.Warn("failed to get Fargate pricing",
							"node", node.Name,
							"region", node.Region,
							"error", err)
						continue
					}
					fargatePriceCache[node.Region] = fargatePricing
				}

				// Calculate Fargate cost based on CPU and memory capacity
				// CPU: millicores -> vCPU (divide by 1000)
				// Memory: bytes -> GB (divide by 1024^3)
				cpuVCPU := float64(node.CPUCapacity) / 1000.0
				memGB := float64(node.MemCapacity) / (1024.0 * 1024.0 * 1024.0)

				cpuCost := cogs.CostValue(cpuVCPU) * fargatePricing.CPUPerHour
				memCost := cogs.CostValue(memGB) * fargatePricing.MemoryPerHour
				node.HourlyCost = cpuCost + memCost
			} else if node.InstanceType != "" {
				// EC2 node - get instance pricing
				price, err := p.pricingProvider.GetEC2Price(ctx, node.Region, node.InstanceType)
				if err != nil {
					slog.Warn("failed to get pricing for node",
						"node", node.Name,
						"instanceType", node.InstanceType,
						"region", node.Region,
						"error", err)
				} else {
					node.HourlyCost = price
				}
			}
		}
	}

	if len(clusters) == 0 {
		return allNodes, nil
	}

	clusterSet := make(map[string]bool)
	for _, c := range clusters {
		clusterSet[c] = true
	}

	var filtered []cogs.NodeInfo
	for _, n := range allNodes {
		if clusterSet[n.ClusterName] {
			filtered = append(filtered, n)
		}
	}
	return filtered, nil
}

// GetPods fetches pods from connected clusters
func (p *ClusterDataProvider) GetPods(ctx context.Context, clusters []string, namespaces []string) ([]cogs.PodInfo, error) {
	allPods, err := p.manager.GetAllPods(ctx, namespaces)
	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return allPods, nil
	}

	clusterSet := make(map[string]bool)
	for _, c := range clusters {
		clusterSet[c] = true
	}

	var filtered []cogs.PodInfo
	for _, pod := range allPods {
		if clusterSet[pod.ClusterName] {
			filtered = append(filtered, pod)
		}
	}
	return filtered, nil
}

// RecalculateFargateCosts updates Fargate node costs based on actual pod requests
func (p *ClusterDataProvider) RecalculateFargateCosts(ctx context.Context, nodes []cogs.NodeInfo, pods []cogs.PodInfo) []cogs.NodeInfo {
	if p.pricingProvider == nil {
		return nodes
	}

	// Build map of pod resources by node
	type nodeResources struct {
		cpuMillicores int64
		memoryBytes   int64
	}
	podResourcesByNode := make(map[string]*nodeResources)

	for _, pod := range pods {
		key := pod.ClusterName + "/" + pod.NodeName
		if _, ok := podResourcesByNode[key]; !ok {
			podResourcesByNode[key] = &nodeResources{}
		}
		podResourcesByNode[key].cpuMillicores += pod.CPURequest
		podResourcesByNode[key].memoryBytes += pod.MemoryRequest
	}

	// Cache Fargate pricing by region
	fargatePriceCache := make(map[string]*pricing.FargatePricing)

	// Update Fargate node costs
	result := make([]cogs.NodeInfo, len(nodes))
	copy(result, nodes)

	for i := range result {
		node := &result[i]
		if !node.IsFargate || node.Region == "" {
			continue
		}

		// Get pod resources for this node
		key := node.ClusterName + "/" + node.Name
		resources, ok := podResourcesByNode[key]
		if !ok {
			// No pods on this node
			node.HourlyCost = 0
			node.CPUCapacity = 0
			node.MemCapacity = 0
			continue
		}

		// Get Fargate pricing for this region
		fargatePricing, ok := fargatePriceCache[node.Region]
		if !ok {
			var err error
			fargatePricing, err = p.pricingProvider.GetFargatePricing(ctx, node.Region)
			if err != nil {
				slog.Warn("failed to get Fargate pricing",
					"node", node.Name,
					"region", node.Region,
					"error", err)
				continue
			}
			fargatePriceCache[node.Region] = fargatePricing
		}

		// Update node capacity to reflect actual pod requests
		node.CPUCapacity = resources.cpuMillicores
		node.MemCapacity = resources.memoryBytes

		// Calculate cost based on actual pod requests
		cpuVCPU := float64(resources.cpuMillicores) / 1000.0
		memGB := float64(resources.memoryBytes) / (1024.0 * 1024.0 * 1024.0)

		cpuCost := cogs.CostValue(cpuVCPU) * fargatePricing.CPUPerHour
		memCost := cogs.CostValue(memGB) * fargatePricing.MemoryPerHour
		node.HourlyCost = cpuCost + memCost
	}

	return result
}

// MemoryStore is an in-memory implementation of DataProvider
type MemoryStore struct {
	nodes []cogs.NodeInfo
	pods  []cogs.PodInfo
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// SetNodes sets the current node data
func (s *MemoryStore) SetNodes(nodes []cogs.NodeInfo) {
	s.nodes = nodes
}

// SetPods sets the current pod data
func (s *MemoryStore) SetPods(pods []cogs.PodInfo) {
	s.pods = pods
}

// GetNodes returns nodes, optionally filtered by cluster names
func (s *MemoryStore) GetNodes(ctx context.Context, clusters []string) ([]cogs.NodeInfo, error) {
	if len(clusters) == 0 {
		return s.nodes, nil
	}

	clusterSet := make(map[string]bool)
	for _, c := range clusters {
		clusterSet[c] = true
	}

	var result []cogs.NodeInfo
	for _, n := range s.nodes {
		if clusterSet[n.ClusterName] {
			result = append(result, n)
		}
	}
	return result, nil
}

// GetPods returns pods, optionally filtered by cluster and namespace
func (s *MemoryStore) GetPods(ctx context.Context, clusters []string, namespaces []string) ([]cogs.PodInfo, error) {
	var result []cogs.PodInfo

	clusterSet := make(map[string]bool)
	for _, c := range clusters {
		clusterSet[c] = true
	}

	nsSet := make(map[string]bool)
	for _, ns := range namespaces {
		nsSet[ns] = true
	}

	for _, p := range s.pods {
		if len(clusters) > 0 && !clusterSet[p.ClusterName] {
			continue
		}
		if len(namespaces) > 0 && !nsSet[p.Namespace] {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

// RecalculateFargateCosts is a no-op for MemoryStore
func (s *MemoryStore) RecalculateFargateCosts(ctx context.Context, nodes []cogs.NodeInfo, pods []cogs.PodInfo) []cogs.NodeInfo {
	return nodes
}
