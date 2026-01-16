package cogs

import "time"

// CostValue represents a monetary cost (in USD)
type CostValue float64

// NodeInfo contains node information for cost calculation
type NodeInfo struct {
	Name         string
	ClusterName  string
	InstanceType string
	Region       string
	HourlyCost   CostValue
	CPUCapacity  int64 // millicores
	MemCapacity  int64 // bytes
	Labels       map[string]string
	IsFargate    bool // True if this is a Fargate virtual node
}

// PodInfo contains pod resource information
type PodInfo struct {
	Name          string
	Namespace     string
	ClusterName   string
	NodeName      string
	WorkloadName  string
	WorkloadKind  string // Deployment, StatefulSet, DaemonSet, etc.
	CPURequest    int64  // millicores
	MemoryRequest int64  // bytes
	CPUUsage      int64  // millicores (for usage-based algorithms)
	MemoryUsage   int64  // bytes (for usage-based algorithms)
	Labels        map[string]string
}

// ClusterCost represents aggregated costs for a cluster
type ClusterCost struct {
	ClusterName    string
	TotalCost      CostValue
	NamespaceCosts map[string]*NamespaceCost
	NodeCount      int
	PodCount       int
}

// NamespaceCost represents aggregated costs for a namespace
type NamespaceCost struct {
	Namespace     string
	TotalCost     CostValue
	WorkloadCosts map[string]*WorkloadCost
}

// WorkloadCost represents costs for a workload (Deployment, etc.)
type WorkloadCost struct {
	Name      string
	Kind      string
	TotalCost CostValue
	Pods      []*PodCost
}

// PodCost represents the cost of a single pod
type PodCost struct {
	Name       string
	Cost       CostValue
	CPURequest int64
	MemRequest int64
	CPUUsage   int64
	MemUsage   int64
}

// CalculationInput contains all data needed for cost calculation
type CalculationInput struct {
	Nodes     []NodeInfo
	Pods      []PodInfo
	Params    AlgorithmParams
	Timestamp time.Time
}

// CalculationResult contains the output of a cost calculation
type CalculationResult struct {
	AlgorithmID  AlgorithmID
	Timestamp    time.Time
	TotalCost    CostValue
	ClusterCosts map[string]*ClusterCost
	Params       AlgorithmParams
}

// AlgorithmInfo provides metadata about an algorithm
type AlgorithmInfo struct {
	ID            AlgorithmID     `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	DefaultParams AlgorithmParams `json:"defaultParams"`
	IsDefault     bool            `json:"isDefault"`
}
