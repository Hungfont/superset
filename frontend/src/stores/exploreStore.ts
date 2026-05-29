import { create } from "zustand";

interface ExploreState {
  datasourceId: string | null;
  vizType: string;
  params: string;
  queryContext: string;
  isDirty: boolean;
  savedParamsSnapshot: string | null;
  savedQueryContextSnapshot: string | null;
  setDatasourceId: (id: string) => void;
  setVizType: (type: string) => void;
  setParams: (p: string) => void;
  setQueryContext: (q: string) => void;
  markClean: () => void;
}

export const useExploreStore = create<ExploreState>((set) => ({
  datasourceId: null,
  vizType: "bar",
  params: "",
  queryContext: "",
  isDirty: false,
  savedParamsSnapshot: null,
  savedQueryContextSnapshot: null,

  setDatasourceId: (id) => set({ datasourceId: id }),
  setVizType: (type) => set({ vizType: type, isDirty: true }),
  setParams: (params) =>
    set((s) => ({
      params,
      isDirty:
        params !== s.savedParamsSnapshot ||
        s.queryContext !== s.savedQueryContextSnapshot,
    })),
  setQueryContext: (qc) =>
    set((s) => ({
      queryContext: qc,
      isDirty:
        qc !== s.savedQueryContextSnapshot ||
        s.params !== s.savedParamsSnapshot,
    })),
  markClean: () =>
    set((s) => ({
      isDirty: false,
      savedParamsSnapshot: s.params,
      savedQueryContextSnapshot: s.queryContext,
    })),
}));
