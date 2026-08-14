import type React from 'react';
import { useMemo, useState } from 'react';
import type { ClusterCost, NamespaceCost, NodeCost, WorkloadCost } from '../../types/cost';

interface CostTableProps {
  clusters?: ClusterCost[];
  namespaces?: NamespaceCost[];
  workloads?: WorkloadCost[];
  nodes?: NodeCost[];
}

type SortDirection = 'asc' | 'desc';

interface SortConfig {
  key: string;
  direction: SortDirection;
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 'All'] as const;
type PageSize = (typeof PAGE_SIZE_OPTIONS)[number];

interface PaginationProps {
  currentPage: number;
  totalItems: number;
  pageSize: PageSize;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: PageSize) => void;
}

const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  totalItems,
  pageSize,
  onPageChange,
  onPageSizeChange,
}) => {
  const effectivePageSize = pageSize === 'All' ? totalItems : pageSize;
  const totalPages = Math.ceil(totalItems / effectivePageSize);
  const isAllSelected = pageSize === 'All';

  const startItem = isAllSelected ? 1 : (currentPage - 1) * effectivePageSize + 1;
  const endItem = isAllSelected ? totalItems : Math.min(currentPage * effectivePageSize, totalItems);

  return (
    <div className="flex items-center justify-between px-6 py-3 bg-gray-50 border-t border-gray-200">
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label htmlFor="pageSize" className="text-sm text-gray-500">
            Rows per page:
          </label>
          <select
            id="pageSize"
            value={pageSize}
            onChange={(e) => onPageSizeChange(e.target.value === 'All' ? 'All' : (Number(e.target.value) as PageSize))}
            className="text-sm border border-gray-300 rounded px-2 py-1 bg-white"
          >
            {PAGE_SIZE_OPTIONS.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
        <div className="text-sm text-gray-500">
          Showing {startItem} to {endItem} of {totalItems} rows
        </div>
      </div>
      {!isAllSelected && totalPages > 1 && (
        <div className="flex items-center gap-2">
          <button
            onClick={() => onPageChange(1)}
            disabled={currentPage === 1}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            «
          </button>
          <button
            onClick={() => onPageChange(currentPage - 1)}
            disabled={currentPage === 1}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            ‹
          </button>
          <span className="px-3 py-1 text-sm text-gray-700">
            Page {currentPage} of {totalPages}
          </span>
          <button
            onClick={() => onPageChange(currentPage + 1)}
            disabled={currentPage === totalPages}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            ›
          </button>
          <button
            onClick={() => onPageChange(totalPages)}
            disabled={currentPage === totalPages}
            className="px-2 py-1 text-base text-gray-600 hover:bg-gray-200 rounded disabled:opacity-50 disabled:cursor-not-allowed"
          >
            »
          </button>
        </div>
      )}
    </div>
  );
};

function paginate<T>(data: T[], page: number, pageSize: PageSize): T[] {
  if (pageSize === 'All') return data;
  const start = (page - 1) * pageSize;
  return data.slice(start, start + pageSize);
}

const SortIcon: React.FC<{ active: boolean; direction: SortDirection }> = ({ active, direction }) => {
  if (!active) return null;
  return <span className="ml-1 inline-block text-blue-600">{direction === 'asc' ? '▲' : '▼'}</span>;
};

const SortableHeader: React.FC<{
  label: string;
  sortKey: string;
  currentSort: SortConfig;
  onSort: (key: string) => void;
  align?: 'left' | 'right';
  rowSpan?: number;
}> = ({ label, sortKey, currentSort, onSort, align = 'left', rowSpan }) => {
  const isActive = currentSort.key === sortKey;
  return (
    <th
      onClick={() => onSort(sortKey)}
      rowSpan={rowSpan}
      className={`px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none whitespace-nowrap ${
        align === 'right' ? 'text-right' : 'text-left'
      }`}
    >
      {label}
      <SortIcon active={isActive} direction={isActive ? currentSort.direction : 'asc'} />
    </th>
  );
};

const CostGroupHeader: React.FC<{
  sortKey: string;
  currentSort: SortConfig;
  onSort: (key: string) => void;
}> = ({ sortKey, currentSort, onSort }) => {
  const isActive = currentSort.key === sortKey;
  return (
    <th
      colSpan={3}
      onClick={() => onSort(sortKey)}
      className="px-6 py-2 text-center text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none whitespace-nowrap border-b border-gray-200"
    >
      Cost
      <SortIcon active={isActive} direction={isActive ? currentSort.direction : 'asc'} />
    </th>
  );
};

const CostSubHeaders: React.FC = () => (
  <>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Hourly</th>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Daily</th>
    <th className="px-6 py-2 text-right text-xs font-medium text-gray-400 tracking-wider">Monthly</th>
  </>
);

function sortData<T>(data: T[], sortConfig: SortConfig): T[] {
  return [...data].sort((a, b) => {
    const aVal = (a as Record<string, unknown>)[sortConfig.key];
    const bVal = (b as Record<string, unknown>)[sortConfig.key];

    let comparison = 0;
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      comparison = aVal.localeCompare(bVal);
    } else if (typeof aVal === 'number' && typeof bVal === 'number') {
      comparison = aVal - bVal;
    }

    return sortConfig.direction === 'asc' ? comparison : -comparison;
  });
}

export const CostTable: React.FC<CostTableProps> = ({ clusters, namespaces, workloads, nodes }) => {
  const [clusterSort, setClusterSort] = useState<SortConfig>({
    key: 'name',
    direction: 'asc',
  });
  const [nodeSort, setNodeSort] = useState<SortConfig>({
    key: 'name',
    direction: 'asc',
  });
  const [namespaceSort, setNamespaceSort] = useState<SortConfig>({
    key: 'namespace',
    direction: 'asc',
  });
  const [workloadSort, setWorkloadSort] = useState<SortConfig>({
    key: 'namespace',
    direction: 'asc',
  });

  const [clusterPage, setClusterPage] = useState(1);
  const [nodePage, setNodePage] = useState(1);
  const [namespacePage, setNamespacePage] = useState(1);
  const [workloadPage, setWorkloadPage] = useState(1);

  const [pageSize, setPageSize] = useState<PageSize>(10);

  const handlePageSizeChange = (newSize: PageSize, resetPage: () => void) => {
    setPageSize(newSize);
    resetPage();
  };

  const handleSort = (
    setter: React.Dispatch<React.SetStateAction<SortConfig>>,
    current: SortConfig,
    key: string,
    resetPage: () => void,
  ) => {
    setter({
      key,
      direction: current.key === key && current.direction === 'asc' ? 'desc' : 'asc',
    });
    resetPage();
  };

  const formatCost = (cost: number, decimals: number = 4) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(cost);
  };

  const dailyCost = (hourly: number) => hourly * 24;
  const monthlyCost = (hourly: number) => hourly * 730;

  const sortedClusters = useMemo(() => {
    if (!clusters) return [];
    return sortData(clusters, clusterSort);
  }, [clusters, clusterSort]);

  const sortedNodes = useMemo(() => {
    if (!nodes) return [];
    return sortData(nodes, nodeSort);
  }, [nodes, nodeSort]);

  const sortedNamespaces = useMemo(() => {
    if (!namespaces) return [];
    return sortData(namespaces, namespaceSort);
  }, [namespaces, namespaceSort]);

  const sortedWorkloads = useMemo(() => {
    if (!workloads) return [];
    return sortData(workloads, workloadSort);
  }, [workloads, workloadSort]);

  if (clusters && clusters.length > 0) {
    const paginatedClusters = paginate(sortedClusters, clusterPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader
                label="Cluster"
                sortKey="name"
                currentSort={clusterSort}
                onSort={(k) => handleSort(setClusterSort, clusterSort, k, () => setClusterPage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Nodes"
                sortKey="nodeCount"
                currentSort={clusterSort}
                onSort={(k) => handleSort(setClusterSort, clusterSort, k, () => setClusterPage(1))}
                rowSpan={2}
                align="right"
              />
              <SortableHeader
                label="Namespaces"
                sortKey="namespaceCount"
                currentSort={clusterSort}
                onSort={(k) => handleSort(setClusterSort, clusterSort, k, () => setClusterPage(1))}
                rowSpan={2}
                align="right"
              />
              <SortableHeader
                label="Pods"
                sortKey="podCount"
                currentSort={clusterSort}
                onSort={(k) => handleSort(setClusterSort, clusterSort, k, () => setClusterPage(1))}
                rowSpan={2}
                align="right"
              />
              <CostGroupHeader
                sortKey="totalCost"
                currentSort={clusterSort}
                onSort={(k) => handleSort(setClusterSort, clusterSort, k, () => setClusterPage(1))}
              />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedClusters.map((cluster) => (
              <tr key={cluster.name}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{cluster.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{cluster.nodeCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">
                  {cluster.namespaceCount}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{cluster.podCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(cluster.totalCost)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(dailyCost(cluster.totalCost), 2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(monthlyCost(cluster.totalCost), 2)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination
          currentPage={clusterPage}
          totalItems={sortedClusters.length}
          pageSize={pageSize}
          onPageChange={setClusterPage}
          onPageSizeChange={(size) => handlePageSizeChange(size, () => setClusterPage(1))}
        />
      </div>
    );
  }

  if (nodes && nodes.length > 0) {
    const paginatedNodes = paginate(sortedNodes, nodePage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader
                label="Cluster"
                sortKey="cluster"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Node"
                sortKey="name"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Type"
                sortKey="instanceType"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Region"
                sortKey="region"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="CPU"
                sortKey="cpuCapacityRaw"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
                align="right"
              />
              <SortableHeader
                label="Memory"
                sortKey="memCapacityRaw"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
                align="right"
              />
              <SortableHeader
                label="Pods"
                sortKey="podCount"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
                rowSpan={2}
                align="right"
              />
              <CostGroupHeader
                sortKey="hourlyCost"
                currentSort={nodeSort}
                onSort={(k) => handleSort(setNodeSort, nodeSort, k, () => setNodePage(1))}
              />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedNodes.map((node) => (
              <tr key={`${node.cluster}-${node.name}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{node.cluster}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{node.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{node.instanceType}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{node.region}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{node.cpuCapacity}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{node.memCapacity}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{node.podCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(node.hourlyCost)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(dailyCost(node.hourlyCost), 2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(monthlyCost(node.hourlyCost), 2)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination
          currentPage={nodePage}
          totalItems={sortedNodes.length}
          pageSize={pageSize}
          onPageChange={setNodePage}
          onPageSizeChange={(size) => handlePageSizeChange(size, () => setNodePage(1))}
        />
      </div>
    );
  }

  if (namespaces && namespaces.length > 0) {
    const paginatedNamespaces = paginate(sortedNamespaces, namespacePage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader
                label="Cluster"
                sortKey="cluster"
                currentSort={namespaceSort}
                onSort={(k) => handleSort(setNamespaceSort, namespaceSort, k, () => setNamespacePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Namespace"
                sortKey="namespace"
                currentSort={namespaceSort}
                onSort={(k) => handleSort(setNamespaceSort, namespaceSort, k, () => setNamespacePage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Pods"
                sortKey="podCount"
                currentSort={namespaceSort}
                onSort={(k) => handleSort(setNamespaceSort, namespaceSort, k, () => setNamespacePage(1))}
                rowSpan={2}
                align="right"
              />
              <CostGroupHeader
                sortKey="totalCost"
                currentSort={namespaceSort}
                onSort={(k) => handleSort(setNamespaceSort, namespaceSort, k, () => setNamespacePage(1))}
              />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedNamespaces.map((ns) => (
              <tr key={`${ns.cluster}-${ns.namespace}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ns.cluster}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{ns.namespace}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{ns.podCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(ns.totalCost)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(dailyCost(ns.totalCost), 2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(monthlyCost(ns.totalCost), 2)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination
          currentPage={namespacePage}
          totalItems={sortedNamespaces.length}
          pageSize={pageSize}
          onPageChange={setNamespacePage}
          onPageSizeChange={(size) => handlePageSizeChange(size, () => setNamespacePage(1))}
        />
      </div>
    );
  }

  if (workloads && workloads.length > 0) {
    const paginatedWorkloads = paginate(sortedWorkloads, workloadPage, pageSize);
    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <SortableHeader
                label="Cluster"
                sortKey="cluster"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Namespace"
                sortKey="namespace"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Workload"
                sortKey="name"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Kind"
                sortKey="kind"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
                rowSpan={2}
              />
              <SortableHeader
                label="Pods"
                sortKey="podCount"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
                rowSpan={2}
                align="right"
              />
              <CostGroupHeader
                sortKey="totalCost"
                currentSort={workloadSort}
                onSort={(k) => handleSort(setWorkloadSort, workloadSort, k, () => setWorkloadPage(1))}
              />
            </tr>
            <tr>
              <CostSubHeaders />
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {paginatedWorkloads.map((wl) => (
              <tr key={`${wl.cluster}-${wl.namespace}-${wl.kind}-${wl.name}`}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{wl.cluster}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{wl.namespace}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{wl.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{wl.kind}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-right">{wl.podCount}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(wl.totalCost)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(dailyCost(wl.totalCost), 2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-right">
                  {formatCost(monthlyCost(wl.totalCost), 2)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <Pagination
          currentPage={workloadPage}
          totalItems={sortedWorkloads.length}
          pageSize={pageSize}
          onPageChange={setWorkloadPage}
          onPageSizeChange={(size) => handlePageSizeChange(size, () => setWorkloadPage(1))}
        />
      </div>
    );
  }

  return (
    <div className="p-6">
      <p className="text-gray-500 text-center">No data available</p>
    </div>
  );
};
