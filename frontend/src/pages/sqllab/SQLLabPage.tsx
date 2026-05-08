import { useEffect, useMemo, useRef, useCallback } from "react";
import { Plus, X } from "lucide-react";
import { useQuery, useMutation } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
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
} from "@/components/query/QueryBadges";
import { WsStatusBadge } from "@/components/query/WsStatusBadge";
import { DownloadButton } from "@/components/query/DownloadButton";
import { DataTable } from "@/components/ui/data-table";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import { useWsStore } from "@/stores/wsStore";
import { queriesApi, type ExecuteQueryResponse, type SubmitQueryResponse, type WsEvent } from "@/api/queries";
import { databasesApi } from "@/api/databases";

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
  const pollingTimeoutRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const {
    tabs,
    activeTabId,
    databaseId,
    addTab,
    removeTab,
    setActiveTab,
    updateTabSql,
    updateTabDatabase,
    setTabResult,
    setTabStatus,
    setTabError,
    setDatabaseId,
    setAsyncState,
  } = useSqlLabStore();

  const wsSubscribe = useWsStore(s => s.subscribe);
  const wsUnsubscribe = useWsStore(s => s.unsubscribe);
  const wsIsFallback = useWsStore(s => s.isFallbackToPolling);


  const activeTab = tabs.find(t => t.id === activeTabId);

  const { data: databasesData, isLoading: databasesLoading } = useQuery({
    queryKey: ["databases"],
    queryFn: () => databasesApi.getDatabases({}),
    enabled: databaseId === null,
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
          if (pollingTimeoutRef.current) {
            clearTimeout(pollingTimeoutRef.current);
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
        if (pollingTimeoutRef.current) {
          clearTimeout(pollingTimeoutRef.current);
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
    if (pollingTimeoutRef.current) {
      clearTimeout(pollingTimeoutRef.current);
    }

    const poll = () => {
      fetchQueryStatus(queryId);
    };

    poll();
    pollingTimeoutRef.current = setInterval(poll, POLLING_INTERVAL_MS);
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

        if (pollingTimeoutRef.current) {
          clearTimeout(pollingTimeoutRef.current);
        }
        wsUnsub(queryId);

        toast("Query complete - Results received via real-time update");
        showSystemNotification("Query Complete", "Your async query has finished processing.");
        store.clearAsyncState(tabId);
      } else if (data.type === "progress") {
        const mappedStatus: "pending" | "queued" | "running" | "done" | "failed" | "stopped" =
          data.progress === "running" ? "running" :
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
        if (pollingTimeoutRef.current) {
          clearTimeout(pollingTimeoutRef.current);
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
        if (pollingTimeoutRef.current) {
          clearTimeout(pollingTimeoutRef.current);
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
      }
      if (pollingTimeoutRef.current) {
        clearTimeout(pollingTimeoutRef.current);
      }
    };
  }, [activeTab?.asyncQueryId, wsUnsubscribe]);

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
    prevConnStatusRef.current = wsConnectionStatus;
  }, [wsConnectionStatus, toast]);

  const submitAsyncMutation = useMutation({
    mutationFn: queriesApi.submit,
    onSuccess: (data: SubmitQueryResponse) => {
      if (activeTabId) {
        setAsyncState(activeTabId, data.query_id, "queued", data.queue);

        toast("Query submitted", {
          description: "Results will appear when complete.",
        });

        startPolling(data.query_id);
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
    onSuccess: () => {
      if (activeTabId) {
        setAsyncState(activeTabId, activeTab?.asyncQueryId || "", "stopped", activeTab?.asyncQueue);
        if (activeTab?.asyncQueryId) {
          wsUnsubscribe(activeTab.asyncQueryId);
        }
        if (pollingTimeoutRef.current) {
          clearTimeout(pollingTimeoutRef.current);
        }
      }
    },
    onError: (error: Error) => {
      if (activeTabId) {
        setTabError(activeTabId, error.message);
      }
    },
  });

  const handleRun = () => {
    if (!activeTab?.databaseId || !activeTab?.sql) return;

    const useAsync = lastQueryDurationRef.current > AUTO_ASYNC_THRESHOLD_MS;

    if (useAsync) {
      handleRunAsync(true);
    } else {
      executeMutation.mutate({
        database_id: activeTab.databaseId,
        sql: activeTab.sql,
        catalog: activeTab.catalog,
        tab_name: activeTab.title,
        sql_editor_id: activeTab.sqlEditorId,
      });
    }
  };

  const handleRunAsync = (_autoDetected = false) => {
    if (!activeTab?.databaseId || !activeTab?.sql) return;

    if (activeTab.asyncQueryId) {
      setTabError(activeTabId!, "A query is already running in this tab");
      return;
    }

    submitAsyncMutation.mutate({
      database_id: activeTab.databaseId,
      sql: activeTab.sql,
      catalog: activeTab.catalog,
      tab_name: activeTab.title,
      sql_editor_id: activeTab.sqlEditorId,
    });
  };

  const handleCancel = () => {
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

  useEffect(() => {
    if (tabs.length === 0) {
      addTab();
    }
  }, [tabs.length, addTab]);

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

  return (
    <div className="container mx-auto py-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">SQL Lab {activeTab?.asyncQueryId && <WsStatusBadge queryId={activeTab.asyncQueryId} />}</h1>
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
          <Button onClick={addTab} size="sm" variant="outline">
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <Tabs value={activeTabId ?? ""} onValueChange={setActiveTab}>
        <TabsList>
          {tabs.map(tab => (
            <TabsTrigger
              key={tab.id}
              value={tab.id}
              className="relative"
            >
              <span className="mr-2">{tab.title}</span>
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
            </TabsTrigger>
          ))}
        </TabsList>

        {tabs.map(tab => (
          <TabsContent key={tab.id} value={tab.id} className="space-y-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                {!tab.asyncStatus && (
                  <>
                    <RunButton
                      onClick={handleRun}
                      disabled={!tab.databaseId || !tab.sql || isRunning || isAsyncRunning}
                      isRunning={isRunning}
                    />
                    <RunAsyncButton
                      onClick={() => handleRunAsync(false)}
                      disabled={!tab.databaseId || !tab.sql || isRunning || isAsyncRunning}
                      isRunning={isRunning}
                      isQueued={tab.asyncStatus === "pending" || tab.asyncStatus === "queued"}
                    />
                  </>
                )}
                {tab.asyncStatus && (
                  <>
                    <CancelButton
                      onClick={handleCancel}
                      disabled={tab.asyncStatus === "done" || tab.asyncStatus === "failed" || tab.asyncStatus === "stopped"}
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

              <textarea
                value={tab.sql}
                onChange={e => {
                  if (activeTabId) {
                    updateTabSql(activeTabId, e.target.value);
                  }
                }}
                placeholder="SELECT * FROM ..."
                className="w-full h-48 p-4 font-mono text-sm bg-muted/30 border rounded-md resize-none"
                disabled={isRunning || isAsyncRunning}
              />
            </div>

            {tab.error && (
              <Alert variant="destructive">
                <AlertDescription>{tab.error}</AlertDescription>
              </Alert>
            )}

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
        ))}
      </Tabs>
    </div>
  );
}