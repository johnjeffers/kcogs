import type React from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../../hooks/useAppDispatch';
import { fetchAlgorithms, fetchCosts } from '../../store/costSlice';
import { ClusterConfig } from '../clusters/ClusterConfig';
import { CostSummary } from './CostSummary';
import { CostTable } from './CostTable';

type TabType = 'clusters' | 'nodes' | 'namespaces' | 'workloads';

const tabs: { id: TabType; label: string }[] = [
  { id: 'clusters', label: 'Clusters' },
  { id: 'nodes', label: 'Nodes' },
  { id: 'namespaces', label: 'Namespaces' },
  { id: 'workloads', label: 'Workloads' },
];

export const CostDashboard: React.FC = () => {
  const dispatch = useAppDispatch();
  const { data, loading, error, algorithms } = useAppSelector((state) => state.costs);
  const [activeTab, setActiveTab] = useState<TabType>('clusters');
  const [filter, setFilter] = useState('');

  const refreshData = useCallback(() => {
    dispatch(fetchCosts());
  }, [dispatch]);

  useEffect(() => {
    dispatch(fetchAlgorithms());
    dispatch(fetchCosts());

    // Auto-refresh every 5 minutes
    const interval = setInterval(
      () => {
        dispatch(fetchCosts());
      },
      5 * 60 * 1000,
    );

    return () => clearInterval(interval);
  }, [dispatch]);

  // Filter data based on active tab and filter text
  // Supports negative filters with "!" prefix (e.g., "!fargate" hides matches)
  const filteredData = useMemo(() => {
    if (!data) return null;

    const matchesFilter = (searchableFields: string[]): boolean => {
      if (!filter.trim()) return true;

      const terms = filter
        .toLowerCase()
        .split(/\s+/)
        .filter((t) => t);
      const combined = searchableFields.join(' ').toLowerCase();

      for (const term of terms) {
        if (term.startsWith('!') && term.length > 1) {
          // Negative filter: exclude if matches
          const searchTerm = term.slice(1);
          if (combined.includes(searchTerm)) return false;
        } else {
          // Positive filter: include only if matches
          if (!combined.includes(term)) return false;
        }
      }
      return true;
    };

    return {
      clusters: data.clusters?.filter((c) => matchesFilter([c.name])),
      nodes: data.nodes?.filter((n) => matchesFilter([n.name, n.cluster, n.instanceType, n.region])),
      namespaces: data.namespaces?.filter((ns) => matchesFilter([ns.namespace, ns.cluster])),
      workloads: data.workloads?.filter((wl) => matchesFilter([wl.name, wl.namespace, wl.cluster, wl.kind])),
    };
  }, [data, filter]);

  const getTabCount = (tab: TabType): { filtered: number; total: number } => {
    if (!data) return { filtered: 0, total: 0 };
    switch (tab) {
      case 'clusters':
        return {
          filtered: filteredData?.clusters?.length || 0,
          total: data.clusters?.length || 0,
        };
      case 'nodes':
        return {
          filtered: filteredData?.nodes?.length || 0,
          total: data.nodes?.length || 0,
        };
      case 'namespaces':
        return {
          filtered: filteredData?.namespaces?.length || 0,
          total: data.namespaces?.length || 0,
        };
      case 'workloads':
        return {
          filtered: filteredData?.workloads?.length || 0,
          total: data.workloads?.length || 0,
        };
    }
  };

  const formatTabCount = (tab: TabType): string => {
    const { filtered, total } = getTabCount(tab);
    if (filtered === total) return String(total);
    return `${filtered}/${total}`;
  };

  // Calculate totals from all data (unfiltered)
  const totals = useMemo(() => {
    if (!data) return { cost: 0, count: 0 };
    const cost = data.totalCost;
    const count = (data.nodes?.length || 0) + (data.namespaces?.length || 0) + (data.workloads?.length || 0);
    return { cost, count };
  }, [data]);

  // Calculate selected summary based on active tab and filter
  const selectedData = useMemo(() => {
    if (!filteredData) return { cost: 0, count: 0 };

    const sumCost = <T extends { totalCost?: number; hourlyCost?: number }>(items: T[] | undefined) =>
      items?.reduce((sum, item) => sum + (item.totalCost ?? item.hourlyCost ?? 0), 0) || 0;

    const isClusterTab = activeTab === 'clusters';

    // For clusters tab, sum all filtered resource types
    if (isClusterTab) {
      const cost = sumCost(filteredData.nodes) + sumCost(filteredData.namespaces) + sumCost(filteredData.workloads);
      const count =
        (filteredData.nodes?.length || 0) +
        (filteredData.namespaces?.length || 0) +
        (filteredData.workloads?.length || 0);
      return { cost, count };
    }

    // For specific resource tabs, show only that resource type's data
    let items: { totalCost?: number; hourlyCost?: number }[] | undefined;
    switch (activeTab) {
      case 'nodes':
        items = filteredData.nodes;
        break;
      case 'namespaces':
        items = filteredData.namespaces;
        break;
      case 'workloads':
        items = filteredData.workloads;
        break;
    }

    return { cost: sumCost(items), count: items?.length || 0 };
  }, [filteredData, activeTab]);

  const exportToCSV = () => {
    if (!filteredData) return;

    let headers: string[] = [];
    let rows: string[][] = [];

    const dailyCost = (hourly: number) => hourly * 24;
    const monthlyCost = (hourly: number) => hourly * 730;

    switch (activeTab) {
      case 'clusters':
        headers = ['Cluster', 'Nodes', 'Pods', 'Namespaces', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.clusters || []).map((c) => [
          c.name,
          String(c.nodeCount),
          String(c.podCount),
          String(c.namespaceCount),
          c.totalCost.toFixed(4),
          dailyCost(c.totalCost).toFixed(2),
          monthlyCost(c.totalCost).toFixed(2),
        ]);
        break;
      case 'nodes':
        headers = [
          'Cluster',
          'Node',
          'Type',
          'Region',
          'CPU',
          'Memory',
          'Pods',
          'Hourly Cost',
          'Daily Cost',
          'Monthly Cost',
        ];
        rows = (filteredData.nodes || []).map((n) => [
          n.cluster,
          n.name,
          n.instanceType,
          n.region,
          n.cpuCapacity,
          n.memCapacity,
          String(n.podCount),
          n.hourlyCost.toFixed(4),
          dailyCost(n.hourlyCost).toFixed(2),
          monthlyCost(n.hourlyCost).toFixed(2),
        ]);
        break;
      case 'namespaces':
        headers = ['Cluster', 'Namespace', 'Pods', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.namespaces || []).map((ns) => [
          ns.cluster,
          ns.namespace,
          String(ns.podCount),
          ns.totalCost.toFixed(4),
          dailyCost(ns.totalCost).toFixed(2),
          monthlyCost(ns.totalCost).toFixed(2),
        ]);
        break;
      case 'workloads':
        headers = ['Cluster', 'Namespace', 'Workload', 'Kind', 'Pods', 'Hourly Cost', 'Daily Cost', 'Monthly Cost'];
        rows = (filteredData.workloads || []).map((wl) => [
          wl.cluster,
          wl.namespace,
          wl.name,
          wl.kind,
          String(wl.podCount),
          wl.totalCost.toFixed(4),
          dailyCost(wl.totalCost).toFixed(2),
          monthlyCost(wl.totalCost).toFixed(2),
        ]);
        break;
    }

    const escapeCSV = (value: string) => {
      if (value.includes(',') || value.includes('"') || value.includes('\n')) {
        return `"${value.replace(/"/g, '""')}"`;
      }
      return value;
    };

    const csvContent = [headers.map(escapeCSV).join(','), ...rows.map((row) => row.map(escapeCSV).join(','))].join(
      '\n',
    );

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `kcogs-${activeTab}-${new Date().toISOString().split('T')[0]}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      {/* Cluster Configuration */}
      <ClusterConfig
        onClusterChange={refreshData}
        onRefresh={refreshData}
        loading={loading}
        showAlgorithm={algorithms.length > 0}
      />

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4 mb-6">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error loading costs</h3>
              <p className="mt-1 text-sm text-red-700">{error}</p>
            </div>
          </div>
        </div>
      )}

      {/* Loading State */}
      {loading && !data && (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      )}

      {/* Cost Data */}
      {data && data.clusters && data.clusters.length > 0 && (
        <>
          <CostSummary
            selectedCost={selectedData.cost}
            totalCost={totals.cost}
            selectedCount={selectedData.count}
            totalCount={totals.count}
            currency={data.currency}
          />

          {/* Tabs and Filter */}
          <div className="bg-white shadow rounded-lg">
            <div className="border-b border-gray-200">
              <div className="flex items-center justify-between px-4">
                {/* Tab Buttons */}
                <nav className="flex -mb-px">
                  {tabs.map((tab) => (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`py-4 px-6 text-sm font-medium border-b-2 ${
                        activeTab === tab.id
                          ? 'border-blue-500 text-blue-600'
                          : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                      }`}
                    >
                      {tab.label}
                      <span
                        className={`ml-2 py-0.5 px-2 rounded-full text-xs ${
                          activeTab === tab.id ? 'bg-blue-100 text-blue-600' : 'bg-gray-100 text-gray-500'
                        }`}
                      >
                        {formatTabCount(tab.id)}
                      </span>
                    </button>
                  ))}
                </nav>

                {/* Filter Input and Export */}
                <div className="py-3 flex items-center gap-3">
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                        />
                      </svg>
                    </div>
                    <input
                      type="text"
                      placeholder="Filter..."
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      className="block w-64 pl-9 pr-3 py-2 border border-gray-300 rounded-md text-sm placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                    />
                    {filter && (
                      <button
                        onClick={() => setFilter('')}
                        className="absolute inset-y-0 right-0 pr-3 flex items-center"
                      >
                        <svg
                          className="h-4 w-4 text-gray-400 hover:text-gray-600"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                        >
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    )}
                  </div>
                  <button
                    onClick={exportToCSV}
                    className="px-3 py-2 text-sm bg-green-600 text-white rounded-md hover:bg-green-700 flex items-center gap-1"
                  >
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                      />
                    </svg>
                    Export CSV
                  </button>
                </div>
              </div>
            </div>

            {/* Tab Content */}
            <div>
              {activeTab === 'clusters' && <CostTable clusters={filteredData?.clusters} />}
              {activeTab === 'nodes' && <CostTable nodes={filteredData?.nodes} />}
              {activeTab === 'namespaces' && <CostTable namespaces={filteredData?.namespaces} />}
              {activeTab === 'workloads' && <CostTable workloads={filteredData?.workloads} />}
            </div>
          </div>
        </>
      )}

      {/* No Data Message */}
      {(!data || !data.clusters || data.clusters.length === 0) && !loading && (
        <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
          <p className="text-sm text-blue-700">
            No cost data available. Connect to a cluster using the configuration above to see cost data.
          </p>
        </div>
      )}
    </div>
  );
};
