import { create } from "zustand";

interface QueryResultQueryMeta {
  id: string;
  client_id?: string;
  sql: string;
  executed_sql: string;
  start_time: string;
  start_running_time?: string;
  end_time: string;
  rows: number;
  limit: number;
  limiting_factor: number;
  status: string;
  progress?: string;
  results_key?: string;
  select_as_cta_used?: boolean;
}

interface QueryResult {
  data: Record<string, unknown>[];
  columns: { name: string; type: string }[];
  from_cache: boolean;
  results_truncated?: boolean;
  query: QueryResultQueryMeta;
}

interface SetAsyncResultPayload {
  data: Record<string, unknown>[];
  columns: { name: string; type: string }[];
  from_cache: boolean;
  results_truncated?: boolean;
  query: Partial<QueryResultQueryMeta>;
}

interface SqlLabTab {
  id: string;
  title: string;
  sql: string;
  databaseId: number | null;
  schema: string;
  catalog?: string;
  sqlEditorId?: string;
  result: QueryResult | null;
  status: "idle" | "running" | "success" | "error";
  error: string | null;
  asyncQueryId?: string;
  asyncStatus?: "pending" | "queued" | "running" | "done" | "failed" | "stopped";
  asyncQueue?: string;
  progress?: string;
}

interface SqlLabState {
  tabs: SqlLabTab[];
  activeTabId: string | null;
  databaseId: number | null;

  addTab: () => void;
  removeTab: (id: string) => void;
  setActiveTab: (id: string) => void;
  updateTabSql: (id: string, sql: string) => void;
  updateTabDatabase: (id: string, dbId: number | null) => void;
  setTabResult: (id: string, result: QueryResult) => void;
  setTabStatus: (id: string, status: SqlLabTab["status"]) => void;
  setTabError: (id: string, error: string | null) => void;
  setDatabaseId: (id: number) => void;
  setAsyncState: (id: string, queryId: string, status: SqlLabTab["asyncStatus"], queue?: string) => void;
  setAsyncResult: (id: string, result: SetAsyncResultPayload) => void;
  clearAsyncState: (id: string) => void;
}

let tabCounter = 0;

export const useSqlLabStore = create<SqlLabState>(set => ({
  tabs: [],
  activeTabId: null,
  databaseId: null,

  addTab: () => {
    const id = `tab-${++tabCounter}`;
    set(state => ({
      tabs: [
        ...state.tabs,
        {
          id,
          title: `Query ${tabCounter}`,
          sql: "",
          databaseId: null,
          schema: "public",
          result: null,
          status: "idle",
          error: null,
        },
      ],
      activeTabId: id,
    }));
  },

  removeTab: (id) => {
    set(state => {
      const newTabs = state.tabs.filter(t => t.id !== id);
      const newActiveId =
        state.activeTabId === id
          ? newTabs[0]?.id ?? null
          : state.activeTabId;
      return { tabs: newTabs, activeTabId: newActiveId };
    });
  },

  setActiveTab: (id) => set({ activeTabId: id }),

  updateTabSql: (id, sql) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, sql } : t
      ),
    }));
  },

  updateTabDatabase: (id, databaseId) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, databaseId } : t
      ),
    }));
  },

  setTabResult: (id, result) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? {
          ...t,
          result: {
            data: result.data,
            columns: result.columns,
            from_cache: result.from_cache,
            results_truncated: result.results_truncated,
            query: {
              id: result.query.id ?? "",
              client_id: result.query.client_id,
              sql: result.query.sql,
              executed_sql: result.query.executed_sql,
              start_time: result.query.start_time ?? new Date().toISOString(),
              start_running_time: result.query.start_running_time,
              end_time: result.query.end_time ?? new Date().toISOString(),
              rows: result.query.rows ?? 0,
              limit: result.query.limit ?? 0,
              limiting_factor: result.query.limiting_factor ?? 0,
              status: result.query.status ?? "success",
              progress: result.query.progress,
              results_key: result.query.results_key,
              select_as_cta_used: result.query.select_as_cta_used,
            },
          },
          status: "success" as const
        } : t
      ),
    }));
  },

  setTabStatus: (id, status) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, status } : t
      ),
    }));
  },

  setTabError: (id, error) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, error, status: "error" as const } : t
      ),
    }));
  },

  setDatabaseId: (id) => set({ databaseId: id }),

  setAsyncState: (id, queryId, status, queue) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, asyncQueryId: queryId, asyncStatus: status, asyncQueue: queue } : t
      ),
    }));
  },

  setAsyncResult: (id, result) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? {
          ...t,
          result: {
            data: result.data,
            columns: result.columns,
            from_cache: result.from_cache,
            results_truncated: result.results_truncated,
            query: {
              id: result.query.id ?? "",
              client_id: result.query.client_id,
              sql: result.query.sql,
              executed_sql: result.query.executed_sql ?? "",
              start_time: result.query.start_time ?? new Date().toISOString(),
              start_running_time: result.query.start_running_time,
              end_time: result.query.end_time ?? new Date().toISOString(),
              rows: result.query.rows ?? 0,
              limit: result.query.limit ?? 0,
              limiting_factor: result.query.limiting_factor ?? 0,
              status: result.query.status ?? "success",
              progress: result.query.progress,
              results_key: result.query.results_key,
              select_as_cta_used: result.query.select_as_cta_used,
            },
          },
          status: "success" as const
        } : t
      ),
    }));
  },

  clearAsyncState: (id) => {
    set(state => ({
      tabs: state.tabs.map(t =>
        t.id === id ? { ...t, asyncQueryId: undefined, asyncStatus: undefined, asyncQueue: undefined } : t
      ),
    }));
  },
}));