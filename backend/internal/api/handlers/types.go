package handlers

import (
	"time"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// CostResponse is the standard response for cost queries
type CostResponse struct {
	Timestamp  time.Time          `json:"timestamp"`
	Algorithm  AlgorithmMeta      `json:"algorithm"`
	TotalCost  float64            `json:"totalCost"`
	Currency   string             `json:"currency"`
	Clusters   []ClusterCostDTO   `json:"clusters,omitempty"`
	Namespaces []NamespaceCostDTO `json:"namespaces,omitempty"`
	Workloads  []WorkloadCostDTO  `json:"workloads,omitempty"`
	Nodes      []NodeCostDTO      `json:"nodes,omitempty"`
	Pods       []PodCostDTO       `json:"pods,omitempty"`
	Filters    AppliedFilters     `json:"filters"`
}

// AlgorithmMeta holds metadata about the algorithm used
type AlgorithmMeta struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Params map[string]any `json:"params,omitempty"`
}

// ClusterCostDTO represents cluster cost data for API responses
type ClusterCostDTO struct {
	Name           string  `json:"name"`
	Region         string  `json:"region"`
	TotalCost      float64 `json:"totalCost"`
	NodeCount      int     `json:"nodeCount"`
	PodCount       int     `json:"podCount"`
	NamespaceCount int     `json:"namespaceCount"`
}

// NamespaceCostDTO represents namespace cost data for API responses
type NamespaceCostDTO struct {
	Cluster   string  `json:"cluster"`
	Namespace string  `json:"namespace"`
	TotalCost float64 `json:"totalCost"`
	PodCount  int     `json:"podCount"`
}

// WorkloadCostDTO represents workload cost data for API responses
type WorkloadCostDTO struct {
	Cluster   string  `json:"cluster"`
	Namespace string  `json:"namespace"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	TotalCost float64 `json:"totalCost"`
	PodCount  int     `json:"podCount"`
}

// PodCostDTO represents pod cost data for API responses
type PodCostDTO struct {
	Cluster      string  `json:"cluster"`
	Namespace    string  `json:"namespace"`
	Workload     string  `json:"workload"`
	WorkloadKind string  `json:"workloadKind"`
	Name         string  `json:"name"`
	Node         string  `json:"node"`
	Cost         float64 `json:"cost"`
	CPURequest   string  `json:"cpuRequest"`
	MemRequest   string  `json:"memRequest"`
}

// NodeCostDTO represents node cost data for API responses
type NodeCostDTO struct {
	Cluster         string  `json:"cluster"`
	Name            string  `json:"name"`
	InstanceType    string  `json:"instanceType"`
	Region          string  `json:"region"`
	HourlyCost      float64 `json:"hourlyCost"`
	PodCount        int     `json:"podCount"`
	CPUCapacity     string  `json:"cpuCapacity"`
	MemCapacity     string  `json:"memCapacity"`
	CPUCapacityRaw  int64   `json:"cpuCapacityRaw"`  // millicores for sorting
	MemCapacityRaw  int64   `json:"memCapacityRaw"`  // bytes for sorting
}

// AppliedFilters shows which filters were applied
type AppliedFilters struct {
	Clusters   []string `json:"clusters,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Workloads  []string `json:"workloads,omitempty"`
}

// ClusterResponse for cluster management endpoints
type ClusterResponse struct {
	Name       string    `json:"name"`
	Region     string    `json:"region"`
	Status     string    `json:"status"`
	NodeCount  int       `json:"nodeCount"`
	PodCount   int       `json:"podCount"`
	LastSynced time.Time `json:"lastSynced"`
	Error      string    `json:"error,omitempty"`
}

// AlgorithmListResponse for listing algorithms
type AlgorithmListResponse struct {
	Algorithms []AlgorithmDTO `json:"algorithms"`
	Default    string         `json:"default"`
}

// AlgorithmDTO represents algorithm info for API responses
type AlgorithmDTO struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	DefaultParams map[string]any `json:"defaultParams"`
	IsDefault     bool           `json:"isDefault"`
}

// HealthResponse for health check endpoint
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ErrorResponse for error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// ToAlgorithmDTO converts internal AlgorithmInfo to API DTO
func ToAlgorithmDTO(info cogs.AlgorithmInfo) AlgorithmDTO {
	params := make(map[string]any)
	if info.DefaultParams.RequestWeight != 0 {
		params["requestWeight"] = info.DefaultParams.RequestWeight
	}
	if info.DefaultParams.UsageWeight != 0 {
		params["usageWeight"] = info.DefaultParams.UsageWeight
	}
	if info.DefaultParams.UsageWindowMinutes != 0 {
		params["usageWindowMinutes"] = info.DefaultParams.UsageWindowMinutes
	}
	if info.DefaultParams.Custom != nil {
		for k, v := range info.DefaultParams.Custom {
			params[k] = v
		}
	}

	return AlgorithmDTO{
		ID:            string(info.ID),
		Name:          info.Name,
		Description:   info.Description,
		DefaultParams: params,
		IsDefault:     info.IsDefault,
	}
}
