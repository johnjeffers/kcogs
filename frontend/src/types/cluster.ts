export interface ContextInfo {
  name: string;
  cluster: string;
  isCurrent: boolean;
}

export interface KubeconfigResponse {
  filename: string;
  contexts: ContextInfo[];
}

export interface ClusterInfo {
  name: string;
  context: string;
  status: string;
}

export interface ClustersResponse {
  clusters: ClusterInfo[];
}
