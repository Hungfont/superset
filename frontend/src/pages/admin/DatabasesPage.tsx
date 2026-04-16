import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Database, Eye, EyeOff, MoreHorizontal, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { databasesApi, type DatabaseDetail } from "@/api/databases";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useToast } from "@/hooks/use-toast";

const columnHelper = createColumnHelper<DatabaseDetail>();
const backendOptions = ["all", "postgresql", "mysql", "bigquery", "snowflake"];

function renderTruncatedText(value: string, maxLength = 24, widthClass = "max-w-[220px]") {
  const trimmedValue = value.trim();
  const displayValue = trimmedValue.length > maxLength ? `${trimmedValue.slice(0, maxLength)}...` : trimmedValue;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`${widthClass} block truncate text-left`}>{displayValue}</span>
      </TooltipTrigger>
      <TooltipContent>{trimmedValue}</TooltipContent>
    </Tooltip>
  );
}

export default function DatabasesPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { success, error } = useToast();

  const [searchQ, setSearchQ] = useState("");
  const [selectedBackend, setSelectedBackend] = useState("all");
  const [page, setPage] = useState(1);
  const [pendingDelete, setPendingDelete] = useState<DatabaseDetail | null>(null);
  const [visibleURIById, setVisibleURIById] = useState<Record<number, boolean>>({});

  const filters = useMemo(
    () => ({
      q: searchQ.trim(),
      backend: selectedBackend === "all" ? "" : selectedBackend,
      page,
      pageSize: 10,
    }),
    [page, searchQ, selectedBackend],
  );

  const databasesQuery = useQuery({
    queryKey: ["databases", filters],
    queryFn: () => databasesApi.getDatabases(filters),
  });

  const deleteMutation = useMutation({
    mutationFn: databasesApi.deleteDatabase,
    onSuccess: () => {
      success("Database deleted");
      setPendingDelete(null);
      queryClient.invalidateQueries({ queryKey: ["databases"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to delete database");
    },
  });

  const testMutation = useMutation({
    mutationFn: databasesApi.testConnectionById,
    onSuccess: (result) => {
      if (result.success) {
        success(`Connection successful (${result.latency_ms ?? 0}ms)`);
        return;
      }
      error(result.error || "Connection failed");
    },
    onError: (err) => {
      error((err as Error).message || "Connection test failed");
    },
  });

  const rows = databasesQuery.data?.items ?? [];
  const pagination = databasesQuery.data?.pagination;

  const columns = useMemo(
    () => [
      columnHelper.accessor("database_name", {
        header: "Name",
        cell: ({ row }) => <span className="font-medium">{renderTruncatedText(row.original.database_name, 22, "max-w-[200px]")}</span>,
      }),
      columnHelper.accessor("backend", {
        header: "Backend",
        cell: ({ row }) => <Badge variant="secondary">{row.original.backend}</Badge>,
      }),
      columnHelper.accessor("expose_in_sqllab", {
        header: "SQL Lab",
        cell: ({ row }) => <Badge variant={row.original.expose_in_sqllab ? "secondary" : "outline"}>{row.original.expose_in_sqllab ? "On" : "Off"}</Badge>,
      }),
      columnHelper.accessor("allow_run_async", {
        header: "Async",
        cell: ({ row }) => <Badge variant={row.original.allow_run_async ? "secondary" : "outline"}>{row.original.allow_run_async ? "On" : "Off"}</Badge>,
      }),
      columnHelper.display({
        id: "status",
        header: "Status",
        cell: ({ row }) => <Badge variant="outline">{row.original.dataset_count && row.original.dataset_count > 0 ? "In Use" : "Connected"}</Badge>,
      }),
      columnHelper.display({
        id: "uri",
        header: "URI",
        cell: ({ row }) => {
          const database = row.original;
          const uri = database.sqlalchemy_uri.trim();
          const isVisible = visibleURIById[database.id] === true;
          const displayValue = isVisible ? uri : "*****";

          const valueNode = (
            <span className="max-w-[220px] block truncate text-left text-xs text-muted-foreground">
              {displayValue}
            </span>
          );

          return (
            <div className="flex items-center gap-1">
              {isVisible ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    {valueNode}
                  </TooltipTrigger>
                  <TooltipContent>{uri}</TooltipContent>
                </Tooltip>
              ) : (
                valueNode
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={isVisible ? "Hide URI" : "Show URI"}
                onClick={() => {
                  setVisibleURIById((prev) => ({
                    ...prev,
                    [database.id]: !isVisible,
                  }));
                }}
              >
                {isVisible ? <EyeOff /> : <Eye />}
              </Button>
            </div>
          );
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const database = row.original;
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label={`Actions for ${database.database_name}`}>
                  <MoreHorizontal />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => navigate(`/admin/settings/databases/${database.id}`)}>
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => testMutation.mutate(database.id)}>
                  Test Connection
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setPendingDelete(database)}>
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      }),
    ],
    [navigate, testMutation, visibleURIById],
  );

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const hasRows = rows.length > 0;

  return (
    <TooltipProvider>
      <div className="flex flex-col gap-4">
        <header className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold">Database Connections</h1>
            <p className="text-sm text-muted-foreground">Manage external databases for SQL Lab and datasets.</p>
          </div>

          <Button onClick={() => navigate("/admin/settings/databases/new")}>
            <Plus data-icon="inline-start" />
            Connect a Database
          </Button>
        </header>

        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-[280px] flex-1">
            <Search className="pointer-events-none absolute left-2 top-2.5 size-4 text-muted-foreground" />
            <Input
              value={searchQ}
              onChange={(event) => {
                setPage(1);
                setSearchQ(event.target.value);
              }}
              className="pl-8"
              placeholder="Search by database name"
              aria-label="Search by database name"
            />
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline">Backend: {selectedBackend}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuRadioGroup
                value={selectedBackend}
                onValueChange={(value) => {
                  setPage(1);
                  setSelectedBackend(value);
                }}
              >
                {backendOptions.map((option) => (
                  <DropdownMenuRadioItem key={option} value={option}>
                    {option}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {databasesQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : null}

        {!databasesQuery.isLoading && !hasRows ? (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Database className="size-4" />
                No databases yet
              </CardTitle>
              <CardDescription>No databases connected yet.</CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={() => navigate("/admin/settings/databases/new")}>Connect a Database</Button>
            </CardContent>
          </Card>
        ) : null}

        {!databasesQuery.isLoading && hasRows ? (
          <div className="rounded-md border">
            <table className="w-full table-fixed text-sm">
              <thead className="bg-muted/50 text-left">
                {table.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <th key={header.id} className="px-3 py-2 font-medium">
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody>
                {table.getRowModel().rows.map((row) => {
                  const database = row.original;
                  return (
                    <tr key={row.id} className="border-t align-middle">
                      {row.getVisibleCells().map((cell) => (
                        <td key={cell.id} className="px-3 py-2">
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </td>
                      ))}
                      <td className="px-3 py-2">
                        <Button
                          variant="ghost"
                          onClick={() => navigate(`/admin/settings/databases/${database.id}`)}
                          aria-label={`Open details for ${database.database_name}`}
                        >
                          Open
                        </Button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : null}

        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" disabled={page <= 1} onClick={() => setPage((prev) => Math.max(prev - 1, 1))}>
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {pagination?.page ?? page}
          </span>
          <Button
            variant="outline"
            disabled={pagination ? pagination.page * pagination.page_size >= pagination.total : true}
            onClick={() => setPage((prev) => prev + 1)}
          >
            Next
          </Button>
        </div>

        <AlertDialog open={pendingDelete !== null} onOpenChange={(open) => !open && setPendingDelete(null)}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {pendingDelete?.database_name}?</AlertDialogTitle>
              <AlertDialogDescription>
                This will disconnect all datasets using this database.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={(event) => {
                  event.preventDefault();
                  if (!pendingDelete) {
                    return;
                  }
                  deleteMutation.mutate(pendingDelete.id);
                }}
              >
                Delete Database
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </TooltipProvider>
  );
}
