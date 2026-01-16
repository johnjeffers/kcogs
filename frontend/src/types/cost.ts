export interface CostResponse {
  timestamp: string;
  algorithm: AlgorithmMeta;
  totalCost: number;
  currency: string;
  clusters?: ClusterCost[];
  namespaces?: NamespaceCost[];
  workloads?: WorkloadCost[];
  nodes?: NodeCost[];
  pods?: PodCost[];
  filters: AppliedFilters;
}

export interface AlgorithmMeta {
  id: string;
  name: string;
  params?: Record<string, unknown>;
}

export interface ClusterCost {
  name: string;
  region: string;
  totalCost: number;
  nodeCount: number;
  podCount: number;
  namespaceCount: number;
}

export interface NamespaceCost {
  cluster: string;
  namespace: string;
  totalCost: number;
  podCount: number;
}

export interface WorkloadCost {
  cluster: string;
  namespace: string;
  name: string;
  kind: string;
  totalCost: number;
  podCount: number;
}

export interface PodCost {
  cluster: string;
  namespace: string;
  workload: string;
  workloadKind: string;
  name: string;
  node: string;
  cost: number;
  cpuRequest: string;
  memRequest: string;
}

export interface NodeCost {
  cluster: string;
  name: string;
  instanceType: string;
  region: string;
  hourlyCost: number;
  podCount: number;
  cpuCapacity: string;
  memCapacity: string;
  cpuCapacityRaw: number;
  memCapacityRaw: number;
}

export interface AppliedFilters {
  clusters?: string[];
  namespaces?: string[];
  workloads?: string[];
}

export interface AlgorithmDTO {
  id: string;
  name: string;
  description: string;
  defaultParams: Record<string, unknown>;
  isDefault: boolean;
}

export interface AlgorithmListResponse {
  algorithms: AlgorithmDTO[];
  default: string;
}

export interface CostFilters {
  clusters?: string[];
  namespaces?: string[];
  algorithm?: string;
}
