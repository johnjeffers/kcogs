import React from 'react';
import { useAppSelector } from '../../hooks/useAppDispatch';

export const AlgorithmSelector: React.FC = () => {
  const { algorithms, algorithm } = useAppSelector((state) => state.costs);

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="algorithm" className="text-sm font-medium text-gray-700">
        Algorithm:
      </label>
      <div className="relative group">
        <svg
          className="w-4 h-4 text-gray-400 cursor-help"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-3 py-2 bg-gray-900 text-white text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-10">
          Additional algorithms may be added in the future.
          <br />
          Currently only Request-Based Allocation is available.
          <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-gray-900"></div>
        </div>
      </div>
      <select
        id="algorithm"
        value={algorithm.id}
        disabled
        className="block w-48 rounded-md border-gray-300 shadow-sm sm:text-sm bg-gray-100 text-gray-500 cursor-not-allowed"
      >
        {algorithms.map((alg) => (
          <option key={alg.id} value={alg.id}>
            {alg.name}
          </option>
        ))}
      </select>
    </div>
  );
};
