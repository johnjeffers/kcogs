import axios from 'axios';
import type { CostResponse, AlgorithmListResponse, CostFilters } from '../types/cost';
import type { KubeconfigResponse, ClustersResponse } from '../types/cluster';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 5 * 60 * 1000, // 5 minutes for large cluster queries
});

export const costApi = {
  async getCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.clusters?.length) {
      params.set('cluster', filters.clusters.join(','));
    }
    if (filters.namespaces?.length) {
      params.set('namespace', filters.namespaces.join(','));
    }
    if (filters.algorithm) {
      params.set('algorithm', filters.algorithm);
    }
    const response = await api.get<CostResponse>(`/costs?${params.toString()}`);
    return response.data;
  },

  async getClusterCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.clusters?.length) {
      params.set('cluster', filters.clusters.join(','));
    }
    if (filters.algorithm) {
      params.set('algorithm', filters.algorithm);
    }
    const response = await api.get<CostResponse>(`/costs/clusters?${params.toString()}`);
    return response.data;
  },

  async getNamespaceCosts(filters: CostFilters = {}): Promise<CostResponse> {
    const params = new URLSearchParams();
    if (filters.clusters?.length) {
      params.set('cluster', filters.clusters.join(','));
    }
    if (filters.namespaces?.length) {
      params.set('namespace', filters.namespaces.join(','));
    }
    if (filters.algorithm) {
      params.set('algorithm', filters.algorithm);
    }
    const response = await api.get<CostResponse>(`/costs/namespaces?${params.toString()}`);
    return response.data;
  },

  async getAlgorithms(): Promise<AlgorithmListResponse> {
    const response = await api.get<AlgorithmListResponse>('/algorithms');
    return response.data;
  },
};

export const clusterApi = {
  async getKubeconfig(): Promise<KubeconfigResponse> {
    const response = await api.get<KubeconfigResponse>('/kubeconfig');
    return response.data;
  },

  async uploadKubeconfig(files: File | File[]): Promise<KubeconfigResponse> {
    const formData = new FormData();
    const fileArray = Array.isArray(files) ? files : [files];
    for (const file of fileArray) {
      formData.append('kubeconfig', file);
    }
    const response = await api.post<KubeconfigResponse>('/kubeconfig', formData);
    return response.data;
  },

  async getClusters(): Promise<ClustersResponse> {
    const response = await api.get<ClustersResponse>('/clusters');
    return response.data;
  },

  async addCluster(name: string, context: string): Promise<void> {
    await api.post('/clusters', { name, context });
  },

  async removeCluster(name: string): Promise<void> {
    await api.delete(`/clusters/${name}`);
  },

  async removeContext(name: string): Promise<KubeconfigResponse> {
    const response = await api.delete<KubeconfigResponse>(`/contexts/${name}`);
    return response.data;
  },
};
