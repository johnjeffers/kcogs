import type React from 'react';
import { useAppDispatch, useAppSelector } from '../../hooks/useAppDispatch';
import { fetchCosts, setAlgorithm } from '../../store/costSlice';

export const AlgorithmSelector: React.FC = () => {
  const dispatch = useAppDispatch();
  const { algorithms, algorithm } = useAppSelector((state) => state.costs);
  const description = algorithms.find((alg) => alg.id === algorithm.id)?.description;
  const hasChoice = algorithms.length > 1;

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    dispatch(setAlgorithm({ id: e.target.value, params: {} }));
    dispatch(fetchCosts());
  };

  return (
    <div className="flex items-center gap-2">
      <label htmlFor="algorithm" className="text-sm font-medium text-gray-700">
        Algorithm:
      </label>
      <div className="relative group">
        <svg className="w-4 h-4 text-gray-400 cursor-help" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <div className="absolute bottom-full left-0 mb-2 px-3 py-2 w-80 bg-gray-900 text-white text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10">
          {description}
          <div className="absolute top-full left-1 border-4 border-transparent border-t-gray-900"></div>
        </div>
      </div>
      <select
        id="algorithm"
        value={algorithm.id}
        onChange={handleChange}
        disabled={!hasChoice}
        className={`block w-48 rounded-md border-gray-300 shadow-sm sm:text-sm ${
          hasChoice ? 'bg-white text-gray-700 cursor-pointer' : 'bg-gray-100 text-gray-500 cursor-not-allowed'
        }`}
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
