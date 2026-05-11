import { useState, useMemo, useCallback } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Search, Play, FileText, Download, Trash2, Loader2, Clock } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
  DataTable,
} from "@/components/ui/data-table";
import { queriesApi, type QueryHistoryItem } from "@/api/queries";
import { useToast } from "@/hooks/use-toast";
import {
  useAuthStore,
} from "@/stores/authStore";

import type { ColumnDef } from "@tanstack/react-table";

interface QueryHistoryTableProps {
  onRunQuery: (sql: string, databaseId: number) => void;
  onLoadSql: (sql: string) => void;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = Math.floor((ms % 60000) / 1000);
  return `${mins}m ${secs}s`;
}

function StatusBadge({ status }: { status: string }) {
  if (status === "success") {
    return <Badge variant="outline" className="text-green-600 bg-green-50 border-green-200">Success</Badge>;
  }
  if (status === "failed") {
    return <Badge variant="outline" className="text-red-600 bg-red-50 border-red-200">Failed</Badge>;
  }
  if (status === "running") {
    return (
      <Badge variant="outline" className="text-amber-600 bg-amber-50 border-amber-200">
        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        Running
      </Badge>
    );
  }
  if (status === "stopped") {
    return <Badge variant="outline" className="text-muted-foreground bg-muted/30">Cancelled</Badge>;
  }
  if (status === "pending" || status === "queued") {
    return (
      <Badge variant="outline" className="text-muted-foreground bg-muted/30">
        <Clock className="mr-1 h-3 w-3" />
        Queued
      </Badge>
    );
  }
  return <Badge variant="outline">{status}</Badge>;
}

export function QueryHistoryTable({ onRunQuery, onLoadSql }: QueryHistoryTableProps) {
  const { toast } = useToast();
  const [status, setStatus] = useState<string>("");
  const [sqlContains, setSqlContains] = useState("");
  const [page, setPage] = useState(1);
  const [clearDialogOpen, setClearDialogOpen] = useState(false);

  const user = useAuthStore(s => s.user);
  const isAdmin = Array.isArray(user?.roles) && user.roles.includes("Admin");

  const filters = useMemo(() => ({
    status: status || undefined,
    sql_contains: sqlContains || undefined,
    page,
    page_size: 20,
  }), [status, sqlContains, page]);

  const { data, isLoading } = useQuery({
    queryKey: ["query-history", filters],
    queryFn: () => queriesApi.getHistory(filters),
    refetchInterval: 5000,
  });

  const deleteMutation = useMutation({
    mutationFn: () => queriesApi.deleteHistory("30d"),
    onSuccess: (result) => {
      toast(`Deleted ${result.deleted} old query records`);
      setClearDialogOpen(false);
    },
    onError: () => {
      toast("Failed to clear history");
    },
  });

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 0;

  const columns = useMemo<ColumnDef<QueryHistoryItem>[]>(() => [
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      id: "sql",
      header: "SQL",
      accessorKey: "sql",
      cell: ({ row }) => {
        const sql = row.original.sql;
        const truncated = sql.length > 100 ? sql.slice(0, 100) + "..." : sql;
        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger className="text-left max-w-md truncate font-mono text-xs cursor-help">
                {truncated}
              </TooltipTrigger>
              <TooltipContent side="bottom" align="start" className="max-w-xl">
                <pre className="text-xs whitespace-pre-wrap break-all">{sql}</pre>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        );
      },
    },
    {
      id: "database_name",
      header: "Database",
      accessorKey: "database_name",
      cell: ({ row }) => (
        <span className="text-sm">{row.original.database_name || `DB #${row.original.database_id}`}</span>
      ),
    },
    {
      id: "duration_ms",
      header: "Duration",
      accessorKey: "duration_ms",
      cell: ({ row }) => <span className="text-sm text-muted-foreground">{formatDuration(row.original.duration_ms)}</span>,
    },
    {
      id: "rows",
      header: "Rows",
      accessorKey: "rows",
      cell: ({ row }) => <span className="text-sm">{row.original.rows.toLocaleString()}</span>,
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        const item = row.original;
        return (
          <div className="flex items-center gap-1">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => onRunQuery(item.sql, item.database_id)}
                  >
                    <Play className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Run Again</TooltipContent>
              </Tooltip>
            </TooltipProvider>

            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2"
                    onClick={() => onLoadSql(item.sql)}
                  >
                    <FileText className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Load SQL</TooltipContent>
              </Tooltip>
            </TooltipProvider>

            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2"
                    disabled={!item.results_key}
                    onClick={async () => {
                      try {
                        const result = await queriesApi.getResult(item.id);
                        // TODO: display result in a modal or result view
                        toast(`Result loaded - ${result.rows} rows`);
                      } catch {
                        toast("Result expired - rerun query");
                      }
                    }}
                  >
                    <Download className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {item.results_key ? "Download Result" : "Result expired - rerun query"}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        );
      },
    },
  ], [onRunQuery, onLoadSql, toast]);

  const handleSearch = useCallback((value: string) => {
    setSqlContains(value);
    setPage(1);
  }, []);

  const handleStatusFilter = useCallback((value: string) => {
    setStatus(value === "all" ? "" : value);
    setPage(1);
  }, []);

  return (
    <div className="space-y-3">
      {/* Filters */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search SQL..."
            value={sqlContains}
            onChange={e => handleSearch(e.target.value)}
            className="pl-8 h-9"
          />
        </div>
        <Select value={status || "all"} onValueChange={handleStatusFilter}>
          <SelectTrigger className="w-[140px] h-9">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="success">Success</SelectItem>
            <SelectItem value="failed">Failed</SelectItem>
            <SelectItem value="running">Running</SelectItem>
            <SelectItem value="stopped">Cancelled</SelectItem>
          </SelectContent>
        </Select>
        {isAdmin && (
          <AlertDialog open={clearDialogOpen} onOpenChange={setClearDialogOpen}>
            <AlertDialogTrigger asChild>
              <Button variant="destructive" size="sm" className="h-9 gap-1">
                <Trash2 className="h-3.5 w-3.5" />
                Clear History
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Clear Query History</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete all query records older than 30 days. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="text-center py-8 text-muted-foreground">Loading history...</div>
      ) : data && data.queries.length > 0 ? (
        <>
          <DataTable data={data.queries} columns={columns} />
          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <span>
                Page {data.page} of {totalPages} ({data.total.toLocaleString()} total)
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage(p => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      ) : (
        <div className="text-center py-8 text-muted-foreground">
          No query history found
        </div>
      )}
    </div>
  );
}
