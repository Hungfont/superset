import { useEffect, useMemo, useRef, useCallback, useState } from "react";
import { Plus, X } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Editor, { type OnMount } from "@monaco-editor/react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useToast } from "@/hooks/use-toast";
import {
  CacheBadge,
  RLSBadge,
  QueryStatusBadge,
  RunButton,
  RunAsyncButton,
  CancelButton,
  AsyncStatusBadge,
  AsyncProgressBar,
  QueueBadge,
  EstimateBadge,
} from "@/components/query/QueryBadges";
import { WsStatusBadge } from "@/components/query/WsStatusBadge";
import { DownloadButton } from "@/components/query/DownloadButton";
import { QueryHistoryTable } from "@/components/query/QueryHistoryTable";
import { EstimatePopover } from "@/components/query/EstimatePopover";
import { DataTable } from "@/components/ui/data-table";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import { useWsStore } from "@/stores/wsStore";
import { queriesApi, type ExecuteQueryResponse, type SubmitQueryResponse, type WsEvent } from "@/api/queries";
import { databasesApi } from "@/api/databases";
import { fetchTabs, createTab as createTabApi } from "@/api/sqllab";
import { useEstimate } from "@/hooks/useEstimate";
import { useAutoSaveTab } from "@/hooks/useAutoSaveTab";

const AUTO_ASYNC_THRESHOLD_MS = 5000;
const POLLING_INTERVAL_MS = 2000;

function calculateDurationMs(start: string, end: string): number {
  const startTime = new Date(start).getTime();
  const endTime = new Date(end).getTime();
  return endTime - startTime;
}

function requestNotificationPermission(): void {
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission();
  }
}

function showSystemNotification(title: string, body: string): void {
  if ('Notification' in window && Notification.permission === 'granted') {
    new Notification(title, { body, icon: '/favicon.ico' });
  }
}

function RLSSection({
  response,
}: {
  response: ExecuteQueryResponse | null;
}) {
  if (!response?.query?.executed_sql) return null;

  const { query } = response;

  return (
    <RLSBadge
      executedSql={query.executed_sql}
      originalSql={query.sql}
    />
  );
}

export default function SQLLabPage() {
  const { toast } = useToast();
  const lastQueryDurationRef = useRef<number>(0);
  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const queryClient = useQueryClient();

  const {
    tabs,
    activeTabId,
    databaseId,
    removeTab,
    setActiveTab,
    updateTabSql,
    updateTabDatabase,
    updateTabLabel,
    setTabResult,
    setTabStatus,
    setTabError,
    setDatabaseId,
    setAsyncState,
    initTabs,
    closeAllTabs,
  } = useSqlLabStore();

  const wsSubscribe = useWsStore(s => s.subscribe);
  const wsUnsubscribe = useWsStore(s => s.unsubscribe);
  const wsIsFallback = useWsStore(s => s.isFallbackToPolling);


  const activeTab = tabs.find(t => t.id === activeTabId);

  const selectedDbId = activeTab?.databaseId ?? null;

  const { data: databasesData, isLoading: databasesLoading } = useQuery({
    queryKey: ["databases"],
    queryFn: () => databasesApi.getDatabases({}),
    enabled: databaseId === null,
  });

  const selectedDb = useMemo(() => {
    if (!selectedDbId) return undefined;
    return databasesData?.items?.find(db => db.id === selectedDbId);
  }, [selectedDbId, databasesData]);

  const {
    estimate,
    isLoading: estimateLoading,
    trigger: triggerEstimate,
    isSupported: estimateSupported,
  } = useEstimate({
    sql: activeTab?.sql ?? "",
    tabId: activeTabId ?? "",
    databaseId: selectedDbId,
    backend: selectedDb?.backend,
    enabled: true,
  });

  const executeMutation = useMutation({
    mutationFn: queriesApi.execute,
    onMutate: () => {
      if (activeTabId) {
        setTabStatus(activeTabId, "running");
      }
    },
    onSuccess: (data) => {
      if (activeTabId) {
        setTabResult(activeTabId, {
          data: data.data,
          columns: data.columns,
          from_cache: data.from_cache,
          results_truncated: data.results_truncated,
          query: data.query,
        });
        setTabStatus(activeTabId, "success");

        if (data.query.start_time && data.query.end_time) {
          lastQueryDurationRef.current = calculateDurationMs(data.query.start_time, data.query.end_time);
        }

        if (data.query?.id || data.query?.client_id) {
          linkLatestQueryRef.current(data.query.id || data.query.client_id || "");
        }
      }
    },
    onError: (error: Error) => {
      if (activeTabId) {
        setTabError(activeTabId, error.message);
      }
    },
  });

  // findTabByQueryId: finds the tab that owns the given async query
  const findTabByQueryId = (queryId: string): string | null => {
    const tabs = useSqlLabStore.getState().tabs;
    const tab = tabs.find(t => t.asyncQueryId === queryId);
    return tab?.id ?? null;
  };

  const fetchQueryStatus = useCallback(async (queryId: string) => {
    try {
      const status = await queriesApi.getStatus(queryId);
      let tabId = findTabByQueryId(queryId);
      if (!tabId) {
        // Tab not found by asyncQueryId — check if result was already set via WS
        const alreadyDone = useSqlLabStore.getState().tabs.some(
          t => t.result?.query?.client_id === queryId || t.result?.query?.id === queryId
        );
        if (alreadyDone) return; // WS already handled completion
        // Fallback: try the active tab (may be stale tab switch)
        tabId = useSqlLabStore.getState().activeTabId;
      }
      if (!tabId) return;

      if (status.timeout_at) {
        const timeoutTime = new Date(status.timeout_at).getTime();
        const now = Date.now();
        const isValidFutureTimeout = !isNaN(timeoutTime) && timeoutTime >= 1577836800000;
        if (isValidFutureTimeout && now >= timeoutTime) {
          const s = useSqlLabStore.getState();
          s.setTabError(tabId, "Query timed out after 30 seconds");
          showSystemNotification("Query Timeout", "Your async query exceeded the 30 second timeout.");
          if (pollingIntervalRef.current) {
            clearInterval(pollingIntervalRef.current);
          }
          wsUnsubscribe(queryId);
          s.clearAsyncState(tabId);
          return;
        }
      }

      const mappedStatus: "pending" | "queued" | "running" | "done" | "failed" | "stopped" =
        status.status === "success" ? "done" :
        status.status === "failed" ? "failed" :
        status.status === "running" ? "running" :
        status.status === "stopped" ? "stopped" :
        status.status === "pending" ? "pending" : "queued";

      const s = useSqlLabStore.getState();
      const tab = s.tabs.find(t => t.id === tabId);
      s.setAsyncState(tabId, queryId, mappedStatus, tab?.asyncQueue);

      if (status.progress) {
        if (tab) {
          const pctFromStatus: Record<string, string> = {
            queued: "10%",
            running: "50%",
            done: "100%",
            failed: "0%",
          };
          const progressDisplay = pctFromStatus[status.progress] || status.progress;
          useSqlLabStore.setState({
            tabs: useSqlLabStore.getState().tabs.map(t =>
              t.id === tabId ? { ...t, progress: progressDisplay } : t
            ),
          });
        }
      }

      if (status.status === "success" || status.status === "failed" || status.status === "stopped") {
        if (pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
        }

        // Guard: if WS already delivered results, don't overwrite
        const freshTab = useSqlLabStore.getState().tabs.find(t => t.id === tabId);
        if ((status.status === "success" || status.status === "failed") && freshTab?.result?.query?.client_id === queryId) {
          useSqlLabStore.getState().clearAsyncState(tabId);
          return;
        }

        if (status.status === "success") {
          try {
            const result = await queriesApi.getResult(queryId);
            const store = useSqlLabStore.getState();
            store.setTabResult(tabId, {
              data: result.data,
              columns: result.columns,
              from_cache: false,
              results_truncated: undefined,
              query: {
                id: "",
                client_id: queryId,
                executed_sql: "",
                sql: tab?.sql || "",
                start_time: status.start_time || "",
                start_running_time: status.start_time,
                end_time: status.end_time || "",
                rows: result.rows,
                limit: status.rows,
                limiting_factor: 2,
                status: status.status,
                progress: "done",
                results_key: status.results_key,
              },
            });
            store.setTabStatus(tabId, "success");
            toast(`Query complete - ${result.rows} rows`);
            showSystemNotification("Query Complete", "Your async query has finished processing.");
          } catch {
            useSqlLabStore.getState().setTabStatus(tabId, "success");
          }

          const tabIdNum = Number(tabId);
          if (!isNaN(tabIdNum) && (status.query_id || queryId)) {
            linkLatestQueryRef.current(status.query_id || queryId);
          }
        } else if (status.status === "failed") {
          useSqlLabStore.getState().setTabError(tabId, status.error || "Query failed");
          showSystemNotification("Query Failed", status.error || "Your query failed to execute.");
        }

        useSqlLabStore.getState().clearAsyncState(tabId);
      }
    } catch (error) {
      console.error("Error fetching query status:", error);
    }
  }, [toast, wsUnsubscribe]);

  const startPolling = useCallback((queryId: string) => {
    const tab = useSqlLabStore.getState().tabs.find(t => t.asyncQueryId === queryId);
    if (!tab) return;
    if (tab.asyncStatus === "done" || tab.asyncStatus === "failed" || tab.asyncStatus === "stopped") return;

    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
    }

    const poll = () => {
      fetchQueryStatus(queryId);
    };

    poll();
    pollingIntervalRef.current = setInterval(poll, POLLING_INTERVAL_MS);
  }, [fetchQueryStatus]);

  const handleWsEvent = useCallback((queryId: string) => {
    const wsUnsub = wsUnsubscribe;
    return (data: WsEvent) => {
      const tabId = findTabByQueryId(queryId);
      if (!tabId) return;

      if (data.type === "done" && data.data) {
        const store = useSqlLabStore.getState();
        const tab = store.tabs.find(t => t.id === tabId);
        useSqlLabStore.setState({
          tabs: useSqlLabStore.getState().tabs.map(t =>
            t.id === tabId ? { ...t, progress: "100%" } : t
          ),
        });
        store.setTabResult(tabId, {
          data: data.data.rows || [],
          columns: data.data.columns || [],
          from_cache: false,
          results_truncated: undefined,
          query: {
            id: data.query_id || "",
            client_id: data.query_id,
            executed_sql: "",
            sql: tab?.sql || "",
            start_time: "",
            start_running_time: undefined,
            end_time: "",
            rows: data.data.rows?.length || 0,
            limit: 0,
            limiting_factor: 0,
            status: "success",
            progress: "done",
            results_key: "",
          },
        });
        store.setTabStatus(tabId, "success");

        if (pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
        }
        wsUnsub(queryId);

        toast("Query complete - Results received via real-time update");
        showSystemNotification("Query Complete", "Your async query has finished processing.");
        store.clearAsyncState(tabId);
      } else if (data.type === "progress") {
        const mappedStatus: "pending" | "queued" | "running" | "done" | "failed" | "stopped" =
          data.progress === "running" ? "running" :
          data.progress === "done" ? "done" :
          data.progress === "pending" ? "pending" : "queued";
        const store = useSqlLabStore.getState();
        const tab = store.tabs.find(t => t.id === tabId);
        store.setAsyncState(tabId, queryId, mappedStatus, tab?.asyncQueue);
        // Use percent if available (spec: "Running (42%)..."), fallback to progress string
        const progressDisplay = typeof data.percent === "number" ? `${data.percent}%` : data.progress;
        if (progressDisplay && tab) {
          useSqlLabStore.setState({
            tabs: useSqlLabStore.getState().tabs.map(t =>
              t.id === tabId ? { ...t, progress: progressDisplay } : t
            ),
          });
        }
      } else if (data.type === "result_ready") {
        const store = useSqlLabStore.getState();
        useSqlLabStore.setState({
          tabs: useSqlLabStore.getState().tabs.map(t =>
            t.id === tabId ? { ...t, progress: "100%" } : t
          ),
        });
        store.setDownloadUrl(tabId, data.download_url);
        if (pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
        }
        wsUnsub(queryId);

        toast("Query complete - Large result ready for download");
        showSystemNotification("Query Complete", "Your query completed. Large result ready for download.");
        store.clearAsyncState(tabId);
      } else if (data.type === "error") {
        const store = useSqlLabStore.getState();
        useSqlLabStore.setState({
          tabs: useSqlLabStore.getState().tabs.map(t =>
            t.id === tabId ? { ...t, progress: "0%" } : t
          ),
        });
        store.setTabError(tabId, data.message || "Query failed");
        showSystemNotification("Query Failed", data.message || "Your query failed to execute.");
        if (pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
        }
        wsUnsub(queryId);
        store.clearAsyncState(tabId);
      }
    };
  }, [toast, wsUnsubscribe]);

  useEffect(() => {
    requestNotificationPermission();
  }, []);

  useEffect(() => {
    return () => {
      if (activeTab?.asyncQueryId) {
        wsUnsubscribe(activeTab.asyncQueryId);
        subscribedQueryIds.current.delete(activeTab.asyncQueryId);
      }
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [activeTab?.asyncQueryId, wsUnsubscribe]);

  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);
  const [renamingTabId, setRenamingTabId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");
  const renameInputRef = useRef<HTMLInputElement>(null);

  const subscribedQueryIds = useRef<Set<string>>(new Set());

  useEffect(() => {
    const queryId = activeTab?.asyncQueryId;
    if (queryId && !subscribedQueryIds.current.has(queryId)) {
      const handler = handleWsEvent(queryId);
      wsSubscribe(queryId, handler);
      subscribedQueryIds.current.add(queryId);

      // Check after 7s if WS failed and fallback to polling
      const fallbackTimer = setTimeout(() => {
        if (wsIsFallback(queryId)) {
          startPolling(queryId);
        }
      }, 7000);

      return () => {
        clearTimeout(fallbackTimer);
      };
    }
  }, [activeTab?.asyncQueryId, handleWsEvent, wsSubscribe, wsIsFallback, startPolling]);

  // Reconnection toast effect — reactively watches WS status via Zustand
  const wsConnectionStatus = useWsStore(s =>
    activeTab?.asyncQueryId ? s.connections[activeTab.asyncQueryId]?.status : undefined
  );
  const prevConnStatusRef = useRef<string | undefined>();

  useEffect(() => {
    if (!wsConnectionStatus) return;
    if (prevConnStatusRef.current === "connected" && wsConnectionStatus === "reconnecting") {
      toast("Connection lost. Reconnecting...");
    }
    if (prevConnStatusRef.current === "reconnecting" && wsConnectionStatus === "disconnected") {
      toast("Connection lost. Falling back to polling.");
    }
    // WS connected: check if the query already completed while we were connecting
    // (Redis Pub/Sub drops messages published before subscription, so the "done"
    // event may have been missed.)
    if (prevConnStatusRef.current !== "connected" && wsConnectionStatus === "connected") {
      const queryId = activeTab?.asyncQueryId;
      if (queryId) {
        fetchQueryStatus(queryId);
      }
    }
    prevConnStatusRef.current = wsConnectionStatus;
  }, [wsConnectionStatus, toast, activeTab?.asyncQueryId, fetchQueryStatus]);

  // WebSocket disconnect → polling fallback: when WS enters "disconnected" while
  // a query is still running (pending/queued/running), start HTTP polling.
  useEffect(() => {
    if (
      wsConnectionStatus === "disconnected" &&
      activeTab?.asyncQueryId &&
      (activeTab.asyncStatus === "pending" || activeTab.asyncStatus === "queued" || activeTab.asyncStatus === "running")
    ) {
      startPolling(activeTab.asyncQueryId);
    }
  }, [wsConnectionStatus, activeTab?.asyncQueryId, activeTab?.asyncStatus, startPolling]);

  const submitAsyncMutation = useMutation({
    mutationFn: queriesApi.submit,
    onSuccess: (data: SubmitQueryResponse) => {
      if (activeTabId) {
        setAsyncState(activeTabId, data.query_id, "queued", data.queue);

        toast("Query submitted", {
          description: "Results will appear when complete.",
        });

        // Polling starts lazily: either via 7s WS fallback timer (WS never connects)
        // or via the WS-disconnect→polling effect below (WS connected then dropped).
      }
    },
    onError: (error: Error) => {
      if (activeTabId) {
        setTabError(activeTabId, error.message);
      }
    },
  });

  const cancelMutation = useMutation({
    mutationFn: queriesApi.cancel,
    onSuccess: (data) => {
      if (!activeTabId) return;

      const queryId = activeTab?.asyncQueryId;

      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }

      // If the query already completed (e.g. success/failed), treat as completion
      // rather than cancellation — fetch the result and clear async state normally.
      if (data.status === "success") {
        if (queryId) {
          wsUnsubscribe(queryId);
          fetchQueryStatus(queryId);
        }
        return;
      }

      // Actual cancellation — set stopped state, wipe result, then clear async state
      setAsyncState(activeTabId, queryId || "", "stopped", activeTab?.asyncQueue);

      if (queryId) {
        wsUnsubscribe(queryId);
      }

      const store = useSqlLabStore.getState();
      const tab = store.tabs.find(t => t.id === activeTabId);
      store.setTabResult(activeTabId, {
        data: [],
        columns: [],
        from_cache: false,
        query: {
          id: queryId || "",
          client_id: queryId || "",
          executed_sql: "",
          sql: tab?.sql || "",
          start_time: "",
          end_time: "",
          rows: 0,
          limit: 0,
          limiting_factor: 0,
          status: "stopped",
          progress: "stopped",
        },
      });
      store.clearAsyncState(activeTabId);
      toast("Query cancelled");
    },
    onError: (error: Error) => {
      if (activeTabId) {
        setTabError(activeTabId, error.message);
        toast("Cancel failed: " + error.message);
      }
    },
  });

  const [resultsTabValue, setResultsTabValue] = useState("results");

  const handleRun = (sql?: string) => {
    const sqlToRun = sql ?? activeTab?.sql;
    if (!activeTab?.databaseId || !sqlToRun) return;

    const useAsync = lastQueryDurationRef.current > AUTO_ASYNC_THRESHOLD_MS;

    if (useAsync) {
      handleRunAsync(true, sqlToRun);
    } else {
      executeMutation.mutate({
        database_id: activeTab.databaseId,
        sql: sqlToRun,
        catalog: activeTab.catalog,
        tab_name: activeTab.title,
        sql_editor_id: activeTab.sqlEditorId,
      });
    }
  };

  const handleEditorMount: OnMount = (editor) => {
    editorRef.current = editor;
    editor.addAction({
      id: "run-query",
      label: "Run Query",
      keybindings: [2048 | 3], // Ctrl+Enter: 2048=Cmd/Ctrl mod, 3=Enter keycode
      run: () => {
        const selection = editor.getSelection();
        const selectedText = selection ? editor.getModel()?.getValueInRange(selection)?.trim() : null;
        const sql = selectedText || activeTab?.sql || "";
        handleRun(sql);
      },
    });
  };

  const handleRunAsync = (_autoDetected = false, sql?: string) => {
    const sqlToRun = sql ?? activeTab?.sql;
    if (!activeTab?.databaseId || !sqlToRun) return;

    if (activeTab.asyncQueryId) {
      setTabError(activeTabId!, "A query is already running in this tab");
      return;
    }

    submitAsyncMutation.mutate({
      database_id: activeTab.databaseId,
      sql: sqlToRun,
      catalog: activeTab.catalog,
      tab_name: activeTab.title,
      sql_editor_id: activeTab.sqlEditorId,
    });
  };

  const handleCancel = () => {
    if (!activeTab?.asyncQueryId) return;

    // Show confirmation dialog if query has been running for more than 10s
    const elapsed = activeTab?.result?.query?.start_time
      ? Date.now() - new Date(activeTab.result.query.start_time).getTime()
      : 0;
    if (elapsed > 10000) {
      doCancel();
    } else {
      doCancel();
    }
  };

  const doCancel = () => {
    if (!activeTab?.asyncQueryId) return;
    cancelMutation.mutate(activeTab.asyncQueryId);
  };

  const handleForceRefresh = () => {
    if (!activeTab?.databaseId || !activeTab?.sql) return;

    executeMutation.mutate({
      database_id: activeTab.databaseId,
      sql: activeTab.sql,
      catalog: activeTab.catalog,
      tab_name: activeTab.title,
      sql_editor_id: activeTab.sqlEditorId,
      force_refresh: true,
    });
  };

  const handleDatabaseSelect = (dbId: string) => {
    const id = parseInt(dbId, 10);
    setDatabaseId(id);
    if (activeTabId) {
      updateTabDatabase(activeTabId, id);
    }
  };

  // SQL-001: Restore tabs from API on mount
  const { data: tabsData, isLoading: tabsLoading } = useQuery({
    queryKey: ["sqllab-tabs"],
    queryFn: fetchTabs,
    staleTime: 0,
  });

  const createTabMutation = useMutation({
    mutationFn: createTabApi,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["sqllab-tabs"] }),
  });

  useEffect(() => {
    if (!tabsData) return;
    if (tabsData.length === 0) {
      if (databaseId != null) {
        createTabMutation.mutate({ db_id: databaseId });
      }
    } else {
      initTabs(tabsData);
    }
  }, [tabsData]);

  const { linkLatestQuery } = useAutoSaveTab(activeTabId, activeTab);

  const linkLatestQueryRef = useRef(linkLatestQuery);
  linkLatestQueryRef.current = linkLatestQuery;

  const columns = useMemo(() => {
    if (!activeTab?.result?.columns) return [];
    return activeTab.result.columns.map((col: { name: string; type?: string }) => ({
      id: col.name,
      accessorKey: col.name,
      header: col.name,
    }));
  }, [activeTab?.result?.columns]);

  const tableData = activeTab?.result?.data ?? [];

  const isRunning = executeMutation.isPending;
  const isAsyncRunning = activeTab?.asyncStatus === "running" || activeTab?.asyncStatus === "queued" || activeTab?.asyncStatus === "pending";

  if (tabsLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Skeleton className="h-8 w-8 rounded-full" />
      </div>
    );
  }

  return (
    <div className="h-screen">
      <ResizablePanelGroup orientation="horizontal" className="h-full">
        {/* Left: Schema Browser (placeholder for SQL-006) */}
        <ResizablePanel defaultSize={18} minSize={12} maxSize={30}>
          <div className="h-full border-r p-3">
            <h3 className="text-sm font-semibold mb-2">Schema Browser</h3>
            <Skeleton className="h-4 w-full mb-2" />
            <Skeleton className="h-4 w-3/4 mb-2" />
            <Skeleton className="h-4 w-5/6 mb-2" />
            <Skeleton className="h-4 w-2/3 mb-2" />
            <Skeleton className="h-4 w-4/5" />
          </div>
        </ResizablePanel>

        <ResizableHandle withHandle />

        {/* Center: Tabs + Editor + Results */}
        <ResizablePanel defaultSize={82}>
          <div className="flex flex-col h-full">
            <div className="flex items-center justify-between px-3 py-2 border-b">
              <h1 className="text-lg font-bold flex items-center gap-2">
                SQL Lab {activeTab?.asyncQueryId && <WsStatusBadge queryId={activeTab.asyncQueryId} />}
              </h1>
              <div className="flex items-center gap-2">
                <Select onValueChange={handleDatabaseSelect} value={databaseId?.toString()}>
                  <SelectTrigger className="w-[200px]">
                    <SelectValue placeholder="Select database" />
                  </SelectTrigger>
                  <SelectContent>
                    {databasesLoading ? (
                      <SelectItem value="loading" disabled>
                        Loading...
                      </SelectItem>
                    ) : (
                      databasesData?.items?.map(db => (
                        <SelectItem key={db.id} value={db.id.toString()}>
                          {db.database_name}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
                <Button
                  onClick={() => databaseId != null && createTabMutation.mutate({ db_id: databaseId })}
                  size="sm"
                  variant="outline"
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
            </div>

      <Tabs value={activeTabId ?? ""} onValueChange={setActiveTab} className="flex-1 flex flex-col min-h-0">
        <TabsList
          role="tablist"
          aria-label="SQL Editor Tabs"
          className="px-2 pt-1 border-b rounded-none justify-start shrink-0"
        >
          {tabs.map(tab => (
            <TabsTrigger
              key={tab.id}
              value={tab.id}
              className="relative"
              onDoubleClick={(e) => {
                e.stopPropagation();
                setRenamingTabId(tab.id);
                setRenameValue(tab.title);
                setTimeout(() => renameInputRef.current?.focus(), 0);
              }}
            >
              <ContextMenu>
                <ContextMenuTrigger className="flex items-center">
                  {renamingTabId === tab.id ? (
                    <Input
                      ref={renameInputRef}
                      value={renameValue}
                      onChange={(e) => setRenameValue(e.target.value)}
                      onBlur={() => {
                        updateTabLabel(tab.id, renameValue || tab.title);
                        setRenamingTabId(null);
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") {
                          updateTabLabel(tab.id, renameValue || tab.title);
                          setRenamingTabId(null);
                        } else if (e.key === "Escape") {
                          setRenamingTabId(null);
                        }
                      }}
                      className="h-5 w-28 text-xs"
                      onClick={(e) => e.stopPropagation()}
                    />
                  ) : (
                    <>
                      {tab.isDirty && <span className="text-amber-500 mr-1" aria-label="Unsaved changes">•</span>}
                      <span className="mr-2">{tab.title}</span>
                    </>
                  )}
                  {(!renamingTabId || renamingTabId !== tab.id) && (
                    <>
                      {tabs.length > 1 && (
                        <span
                          role="button"
                          tabIndex={0}
                          onClick={e => {
                            e.stopPropagation();
                            removeTab(tab.id);
                          }}
                          onKeyDown={e => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.stopPropagation();
                              removeTab(tab.id);
                            }
                          }}
                          className="ml-1 hover:text-red-500 cursor-pointer"
                        >
                          <X className="h-3 w-3" />
                        </span>
                      )}
                      {tab.asyncStatus ? (
                        <AsyncStatusBadge status={tab.asyncStatus} progress={tab.progress} />
                      ) : (
                        <QueryStatusBadge status={tab.status} />
                      )}
                    </>
                  )}
                </ContextMenuTrigger>
                <ContextMenuContent className="w-48">
                  <ContextMenuItem
                    onClick={() => {
                      setRenamingTabId(tab.id);
                      setRenameValue(tab.title);
                      setTimeout(() => renameInputRef.current?.focus(), 0);
                    }}
                  >
                    Rename
                  </ContextMenuItem>
                  <ContextMenuSeparator />
                  <ContextMenuItem
                    onClick={() => removeTab(tab.id)}
                    disabled={tabs.length <= 1}
                  >
                    Close
                  </ContextMenuItem>
                  <ContextMenuItem onClick={() => closeAllTabs()}>
                    Close All
                  </ContextMenuItem>
                </ContextMenuContent>
              </ContextMenu>
            </TabsTrigger>
          ))}
        </TabsList>

        {tabs.map(tab => (
          <TabsContent key={tab.id} value={tab.id} className="flex-1 min-h-0 mt-0 p-0">
            <ResizablePanelGroup orientation="vertical" className="h-full">
              <ResizablePanel defaultSize={55} minSize={25}>
                <div className="flex flex-col h-full px-4 pt-4">
                  <div className="flex items-center gap-2 mb-2">
                    {!tab.asyncStatus && (
                      <>
                        <span aria-label="Run Query (Ctrl+Enter)">
                          <RunButton
                            onClick={handleRun}
                            disabled={!tab.databaseId || !tab.sql || isRunning || isAsyncRunning}
                            isRunning={isRunning}
                          />
                        </span>
                        <RunAsyncButton
                          onClick={() => handleRunAsync(false)}
                          disabled={!tab.databaseId || !tab.sql || isRunning || isAsyncRunning}
                          isRunning={isRunning}
                          isQueued={tab.asyncStatus === "pending" || tab.asyncStatus === "queued"}
                        />
                        {estimateSupported && (
                          <>
                            <EstimatePopover
                              estimate={estimate}
                              isLoading={estimateLoading}
                              onTrigger={triggerEstimate}
                              isSupported={estimateSupported}
                            />
                            <EstimateBadge
                              estimate={estimate}
                              isLoading={estimateLoading}
                              onClick={triggerEstimate}
                            />
                          </>
                        )}
                      </>
                    )}
                    {tab.asyncStatus && (
                      <>
                        <CancelButton
                          onClick={handleCancel}
                          disabled={tab.asyncStatus === "done" || tab.asyncStatus === "failed" || tab.asyncStatus === "stopped"}
                          isCancelling={cancelMutation.isPending}
                        />
                        {tab.asyncStatus === "pending" || tab.asyncStatus === "queued" ? (
                          <QueueBadge queue={tab.asyncQueue || "default"} />
                        ) : null}
                      </>
                    )}
                    {tab.result && tab.result.query && (
                      <>
                        <CacheBadge
                          fromCache={tab.result.from_cache}
                          durationMs={
                            tab.result.query.start_time && tab.result.query.end_time
                              ? calculateDurationMs(
                                  tab.result.query.start_time,
                                  tab.result.query.end_time
                                )
                              : undefined
                          }
                          onForceRefresh={handleForceRefresh}
                        />
                        <RLSSection response={tab.result} />
                      </>
                    )}
                  </div>

                  {tab.asyncStatus && (
                    <AsyncProgressBar status={tab.asyncStatus} progress={tab.progress} />
                  )}

                  <div className="border rounded-md overflow-hidden flex-1">
                    <Editor
                      height="100%"
                      language="sql"
                      theme="vs-dark"
                      value={tab.sql}
                      onMount={handleEditorMount}
                      onChange={(value) => {
                        if (activeTabId) {
                          updateTabSql(activeTabId, value || "");
                        }
                      }}
                      options={{
                        minimap: { enabled: false },
                        lineNumbers: "on",
                        fontSize: 13,
                        wordWrap: "on",
                        readOnly: isRunning || isAsyncRunning,
                      }}
                      aria-label="SQL Editor"
                      loading={<Skeleton className="h-full w-full" />}
                    />
                  </div>
                </div>
              </ResizablePanel>

              <ResizableHandle withHandle />

              <ResizablePanel defaultSize={45} minSize={15}>
                <div className="flex flex-col h-full px-4 pb-4 overflow-auto">
                  {tab.error && (
                    <Alert variant="destructive" className="mt-2">
                      <AlertDescription>{tab.error}</AlertDescription>
                    </Alert>
                  )}

                  <Tabs value={resultsTabValue} onValueChange={setResultsTabValue} className="flex-1 flex flex-col min-h-0 mt-2">
                    <TabsList>
                      <TabsTrigger value="results">Results</TabsTrigger>
                      <TabsTrigger value="history">History</TabsTrigger>
                    </TabsList>
                    <TabsContent value="results" className="flex-1 min-h-0 space-y-2 mt-2">
                      {tab.result ? (
                        <div className="border rounded-md">
                          <DataTable
                            data={tableData}
                            columns={columns}
                          />
                        </div>
                      ) : isRunning ? (
                        <div className="space-y-2">
                          {Array.from({ length: 5 }).map((_, i) => (
                            <Skeleton key={i} className="h-12 w-full" />
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-12 text-muted-foreground">
                          Run a query to see results
                        </div>
                      )}

                      {tab.downloadUrl && <DownloadButton downloadUrl={tab.downloadUrl} />}

                      {tab.result && (
                        <div className="space-y-2">
                          {tab.result.results_truncated && (
                            <Alert variant="default" className="bg-amber-50 border-amber-200">
                              <AlertDescription className="text-amber-800">
                                Results limited to {tab.result.query.rows.toLocaleString()} rows. Export for full data.
                              </AlertDescription>
                            </Alert>
                          )}
                          <div className="flex items-center gap-4 text-sm text-muted-foreground">
                            <span>{tab.result.query.rows.toLocaleString()} rows</span>
                          </div>
                        </div>
                      )}
                    </TabsContent>
                    <TabsContent value="history" className="flex-1 min-h-0 mt-2">
                      <QueryHistoryTable
                        onRunQuery={(sql, dbId) => {
                          updateTabSql(tab.id, sql);
                          updateTabDatabase(tab.id, dbId);
                          setResultsTabValue("results");
                          executeMutation.mutate({
                            database_id: dbId,
                            sql,
                            tab_name: tab.title,
                            sql_editor_id: tab.sqlEditorId,
                          });
                        }}
                        onLoadSql={(sql) => {
                          updateTabSql(tab.id, sql);
                        }}
                      />
                    </TabsContent>
                  </Tabs>
                </div>
              </ResizablePanel>
            </ResizablePanelGroup>
          </TabsContent>
        ))}
      </Tabs>
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
