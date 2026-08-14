import type React from 'react';
import { useEffect, useRef, useState } from 'react';
import { clusterApi } from '../../services/api';
import type { ClusterInfo, ContextInfo } from '../../types/cluster';
import { AlgorithmSelector } from '../filters/AlgorithmSelector';

interface ClusterConfigProps {
  onClusterChange?: () => void;
  onRefresh?: () => void;
  loading?: boolean;
  showAlgorithm?: boolean;
}

export const ClusterConfig: React.FC<ClusterConfigProps> = ({
  onClusterChange,
  onRefresh,
  loading: externalLoading,
  showAlgorithm,
}) => {
  const [contexts, setContexts] = useState<ContextInfo[]>([]);
  const [connectedClusters, setConnectedClusters] = useState<ClusterInfo[]>([]);
  const [clusterErrors, setClusterErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);
  const [togglingCluster, setTogglingCluster] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [collapsed, setCollapsed] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const [kubeconfigRes, clustersRes] = await Promise.all([clusterApi.getKubeconfig(), clusterApi.getClusters()]);
      setContexts(kubeconfigRes.contexts || []);
      setConnectedClusters(clustersRes.clusters || []);
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files || files.length === 0) return;

    setLoading(true);
    setError(null);
    const newErrors: Record<string, string> = {};
    try {
      // Upload all selected files (they will be merged on the backend)
      const res = await clusterApi.uploadKubeconfig(Array.from(files));
      setContexts(res.contexts || []);

      // Get currently connected cluster names
      const alreadyConnected = new Set(connectedClusters.map((c) => c.context));

      // Auto-connect to new contexts only
      const newContexts = (res.contexts || []).filter((ctx) => !alreadyConnected.has(ctx.name));
      for (const ctx of newContexts) {
        try {
          await clusterApi.addCluster(ctx.name, ctx.name);
        } catch (connectErr: any) {
          const errMsg = connectErr.response?.data?.details || connectErr.response?.data?.error || 'Connection failed';
          newErrors[ctx.name] = errMsg;
          console.warn(`Failed to connect to ${ctx.name}:`, connectErr);
        }
      }

      setClusterErrors((prev) => ({ ...prev, ...newErrors }));

      // Reload connected clusters
      const clustersRes = await clusterApi.getClusters();
      setConnectedClusters(clustersRes.clusters || []);
      if (newContexts.length > 0) {
        onClusterChange?.();
      }
    } catch (err: any) {
      console.error('Upload error:', err);
      const errorMsg =
        err.response?.data?.error || err.response?.data?.message || err.message || 'Failed to load kubeconfig';
      setError(`${errorMsg}${err.response?.data?.details ? ': ' + err.response.data.details : ''}`);
    } finally {
      setLoading(false);
      // Reset the file input so the same file(s) can be selected again
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleBrowseClick = () => {
    fileInputRef.current?.click();
  };

  const isConnected = (contextName: string) => {
    return connectedClusters.some((c) => c.context === contextName);
  };

  const toggleCluster = async (contextName: string) => {
    setTogglingCluster(contextName);
    setError(null);

    if (isConnected(contextName)) {
      // Disconnect
      try {
        await clusterApi.removeCluster(contextName);
        setClusterErrors((prev) => {
          const updated = { ...prev };
          delete updated[contextName];
          return updated;
        });
        const res = await clusterApi.getClusters();
        setConnectedClusters(res.clusters || []);
        onClusterChange?.();
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to disconnect cluster');
      }
    } else {
      // Connect
      try {
        await clusterApi.addCluster(contextName, contextName);
        setClusterErrors((prev) => {
          const updated = { ...prev };
          delete updated[contextName];
          return updated;
        });
        const res = await clusterApi.getClusters();
        setConnectedClusters(res.clusters || []);
        onClusterChange?.();
      } catch (err: any) {
        const errMsg = err.response?.data?.details || err.response?.data?.error || 'Connection failed';
        setClusterErrors((prev) => ({ ...prev, [contextName]: errMsg }));
      }
    }

    setTogglingCluster(null);
  };

  const getClusterStatus = (contextName: string): 'connected' | 'disconnected' | 'error' => {
    if (clusterErrors[contextName]) return 'error';
    if (isConnected(contextName)) return 'connected';
    return 'disconnected';
  };

  const removeContext = async (contextName: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent toggle from firing
    setTogglingCluster(contextName);
    setError(null);

    try {
      const res = await clusterApi.removeContext(contextName);
      setContexts(res.contexts || []);
      setClusterErrors((prev) => {
        const updated = { ...prev };
        delete updated[contextName];
        return updated;
      });
      // Reload connected clusters
      const clustersRes = await clusterApi.getClusters();
      setConnectedClusters(clustersRes.clusters || []);
      onClusterChange?.();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to remove context');
    } finally {
      setTogglingCluster(null);
    }
  };

  return (
    <div className="bg-white shadow rounded-lg mb-6">
      <div className="px-6 py-4 flex items-center justify-between">
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="flex items-center gap-3 hover:bg-gray-50 -ml-2 px-2 py-1 rounded transition-colors"
        >
          <svg
            className={`w-5 h-5 text-gray-500 transition-transform ${collapsed ? '-rotate-90' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
          <h2 className="text-lg font-medium text-gray-900">Cluster Configuration</h2>
          {collapsed && connectedClusters.length > 0 && (
            <span className="text-sm text-gray-500">({connectedClusters.length} connected)</span>
          )}
        </button>
      </div>

      {!collapsed && (
        <div className="px-6 pb-6">
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-md p-3 mb-4">
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}

          <input ref={fileInputRef} type="file" multiple onChange={handleFileSelect} className="hidden" />

          {/* Clusters */}
          {contexts.length > 0 ? (
            <div className="flex items-start justify-between gap-4">
              <div className="flex flex-wrap gap-2 flex-1">
                {contexts.map((ctx) => {
                  const status = getClusterStatus(ctx.name);
                  const isToggling = togglingCluster === ctx.name;
                  const errorMsg = clusterErrors[ctx.name];

                  const statusColors = {
                    connected: 'bg-green-100 text-green-800 hover:bg-green-200',
                    disconnected: 'bg-gray-100 text-gray-600 hover:bg-gray-200',
                    error: 'bg-red-100 text-red-800 hover:bg-red-200',
                  };

                  const dotColors = {
                    connected: 'bg-green-500',
                    disconnected: 'bg-gray-400',
                    error: 'bg-red-500',
                  };

                  return (
                    <div
                      key={ctx.name}
                      className={`group inline-flex items-center pl-3 pr-1.5 py-1.5 rounded-full text-sm font-medium transition-colors ${statusColors[status]}`}
                    >
                      <button
                        onClick={() => toggleCluster(ctx.name)}
                        disabled={loading || isToggling}
                        title={errorMsg || (status === 'connected' ? 'Click to disconnect' : 'Click to connect')}
                        className="inline-flex items-center cursor-pointer disabled:opacity-50 disabled:cursor-wait"
                      >
                        <span
                          className={`w-2 h-2 rounded-full mr-2 ${isToggling ? 'animate-pulse bg-yellow-500' : dotColors[status]}`}
                        ></span>
                        {ctx.name}
                      </button>
                      <button
                        onClick={(e) => removeContext(ctx.name, e)}
                        disabled={loading || isToggling}
                        title="Remove from list"
                        className="ml-1.5 p-0.5 rounded-full opacity-0 group-hover:opacity-100 hover:bg-black/10 transition-opacity disabled:opacity-50"
                      >
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  );
                })}
              </div>
              <button
                onClick={handleBrowseClick}
                disabled={loading}
                className="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:opacity-50 whitespace-nowrap"
              >
                {loading ? 'Loading...' : 'Add More...'}
              </button>
            </div>
          ) : (
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-500">Select kubeconfig files to connect to clusters.</p>
              <button
                onClick={handleBrowseClick}
                disabled={loading}
                className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
              >
                {loading ? 'Loading...' : 'Browse...'}
              </button>
            </div>
          )}

          {/* Algorithm Selector and Refresh */}
          {(showAlgorithm || onRefresh) && (
            <div className="mt-4 pt-4 border-t border-gray-200 flex items-center justify-between">
              {showAlgorithm ? <AlgorithmSelector /> : <div />}
              {onRefresh && (
                <button
                  onClick={onRefresh}
                  disabled={externalLoading || connectedClusters.length === 0}
                  className="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded hover:bg-gray-200 disabled:opacity-50"
                >
                  {externalLoading ? 'Refreshing...' : 'Refresh'}
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
