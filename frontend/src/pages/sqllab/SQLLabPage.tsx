import { useEffect, useMemo } from "react";
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
import {
  CacheBadge,
  RLSBadge,
  QueryStatusBadge,
  RunButton,
} from "@/components/query/QueryBadges";
import { DataTable } from "@/components/ui/data-table";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import { queriesApi, type ExecuteQueryResponse } from "@/api/queries";
import { databasesApi } from "@/api/databases";

function calculateDurationMs(start: string, end: string): number {
  const startTime = new Date(start).getTime();
  const endTime = new Date(end).getTime();
  return endTime - startTime;
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
  } = useSqlLabStore();

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
          query: data.query,
        });
        setTabStatus(activeTabId, "success");
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

    executeMutation.mutate({
      database_id: activeTab.databaseId,
      sql: activeTab.sql,
    });
  };

  const handleForceRefresh = () => {
    if (!activeTab?.databaseId || !activeTab?.sql) return;

    executeMutation.mutate({
      database_id: activeTab.databaseId,
      sql: activeTab.sql,
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
    return activeTab.result.columns.map(col => ({
      accessorKey: col.name,
      header: col.name,
    }));
  }, [activeTab?.result?.columns]);

  const tableData = activeTab?.result?.data ?? [];

  return (
    <div className="container mx-auto py-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">SQL Lab</h1>
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

      <Tabs value={activeTabId ?? undefined} onValueChange={setActiveTab}>
        <TabsList>
          {tabs.map(tab => (
            <TabsTrigger
              key={tab.id}
              value={tab.id}
              className="relative"
            >
              <span className="mr-2">{tab.title}</span>
              {tabs.length > 1 && (
                <button
                  type="button"
                  onClick={e => {
                    e.stopPropagation();
                    removeTab(tab.id);
                  }}
                  className="ml-1 hover:text-red-500"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
              <QueryStatusBadge status={tab.status} />
            </TabsTrigger>
          ))}
        </TabsList>

        {tabs.map(tab => (
          <TabsContent key={tab.id} value={tab.id} className="space-y-4">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <RunButton
                  onClick={handleRun}
                  disabled={!tab.databaseId || !tab.sql}
                  isRunning={executeMutation.isPending}
                />
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

              <textarea
                value={tab.sql}
                onChange={e => {
                  if (activeTabId) {
                    updateTabSql(activeTabId, e.target.value);
                  }
                }}
                placeholder="SELECT * FROM ..."
                className="w-full h-48 p-4 font-mono text-sm bg-muted/30 border rounded-md resize-none"
                disabled={executeMutation.isPending}
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
            ) : executeMutation.isPending ? (
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

            {tab.result && (
              <div className="flex items-center gap-4 text-sm text-muted-foreground">
                <span>{tab.result.query.rows} rows</span>
              </div>
            )}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}