import type { PayloadAction } from '@reduxjs/toolkit';
import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import { costApi } from '../services/api';
import type { AlgorithmDTO, CostFilters, CostResponse } from '../types/cost';

interface AlgorithmConfig {
  id: string;
  params: Record<string, unknown>;
}

interface CostState {
  data: CostResponse | null;
  algorithms: AlgorithmDTO[];
  loading: boolean;
  error: string | null;
  algorithm: AlgorithmConfig;
  filters: CostFilters;
}

const initialState: CostState = {
  data: null,
  algorithms: [],
  loading: false,
  error: null,
  algorithm: {
    id: 'request-based',
    params: {},
  },
  filters: {},
};

export const fetchCosts = createAsyncThunk('costs/fetch', async (_, { getState }) => {
  const state = getState() as { costs: CostState };
  return await costApi.getCosts({
    ...state.costs.filters,
    algorithm: state.costs.algorithm.id,
  });
});

export const fetchAlgorithms = createAsyncThunk('costs/fetchAlgorithms', async () => {
  return await costApi.getAlgorithms();
});

const costSlice = createSlice({
  name: 'costs',
  initialState,
  reducers: {
    setAlgorithm: (state, action: PayloadAction<AlgorithmConfig>) => {
      state.algorithm = action.payload;
    },
    setFilters: (state, action: PayloadAction<CostFilters>) => {
      state.filters = action.payload;
    },
    clearError: (state) => {
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchCosts.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchCosts.fulfilled, (state, action) => {
        state.loading = false;
        state.data = action.payload;
      })
      .addCase(fetchCosts.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch costs';
      })
      .addCase(fetchAlgorithms.fulfilled, (state, action) => {
        state.algorithms = action.payload.algorithms;
        if (action.payload.default) {
          state.algorithm.id = action.payload.default;
        }
      });
  },
});

export const { setAlgorithm, setFilters, clearError } = costSlice.actions;
export default costSlice.reducer;
