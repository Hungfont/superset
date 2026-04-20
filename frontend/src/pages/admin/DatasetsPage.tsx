import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Database, MoreHorizontal, Plus, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { databasesApi } from "@/api/databases";
import { datasetsApi } from "@/api/datasets";
import type { DatasetWithCounts } from "@/api/datasets";
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

const columnHelper = createColumnHelper<DatasetWithCounts>();
const typeOptions = ["all", "physical", "virtual"];

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

export default function DatasetsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { success, error } = useToast();

  const [searchQ, setSearchQ] = useState("");
  const [dbFilter, setDbFilter] = useState<number | undefined>(undefined);
  const [schemaFilter, setSchemaFilter] = useState<string | undefined>(undefined);
  const [selectedType, setSelectedType] = useState("all");
  const [page, setPage] = useState(1);
  const [pendingDelete, setPendingDelete] = useState<DatasetWithCounts | null>(null);

  const databasesQuery = useQuery({
    queryKey: ["databases", { page: 1, pageSize: 100 }],
    queryFn: () => databasesApi.getDatabases({ page: 1, pageSize: 100 }),
  });

  const schemasQuery = useQuery({
    queryKey: ["db-schemas", dbFilter],
    queryFn: () => (dbFilter ? databasesApi.getSchemas(dbFilter) : Promise.resolve([])),
    enabled: !!dbFilter,
  });

  const filters = useMemo(
    (): Parameters<typeof datasetsApi.getDatasets>[0] => ({
      q: searchQ.trim() || undefined,
      database_id: dbFilter,
      schema: schemaFilter,
      type: selectedType === "all" ? undefined : selectedType as "physical" | "virtual",
      page,
      page_size: 10,
    }),
    [page, searchQ, dbFilter, schemaFilter, selectedType],
  );

  const datasetsQuery = useQuery({
    queryKey: ["datasets", filters],
    queryFn: () => datasetsApi.getDatasets(filters),
  });

  const deleteMutation = useMutation({
    mutationFn: ({ id, force }: { id: number; force: boolean }) => datasetsApi.deleteDataset(id, force),
    onSuccess: () => {
      success("Dataset deleted");
      setPendingDelete(null);
      setPage(1);
      queryClient.invalidateQueries({ queryKey: ["datasets"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to delete dataset");
    },
  });

  const rows = datasetsQuery.data?.items ?? [];
  const pagination = datasetsQuery.data;

  const columns = useMemo(
    () => [
      columnHelper.accessor("table_name", {
        header: "Name",
        cell: ({ row }) => <span className="font-medium">{renderTruncatedText(row.original.table_name, 22, "max-w-[200px]")}</span>,
      }),
      columnHelper.accessor("type", {
        header: "Type",
        cell: ({ row }) => {
          const isPhysical = row.original.type === "physical";
          return (
            <Badge variant={isPhysical ? "default" : "secondary"}>
              {isPhysical ? "Physical" : "Virtual"}
            </Badge>
          );
        },
      }),
      columnHelper.accessor("database_name", {
        header: "Database",
        cell: ({ row }) => <Badge variant="outline">{row.original.database_name || "—"}</Badge>,
      }),
      columnHelper.accessor("schema", {
        header: "Schema",
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.schema || "—"}</span>,
      }),
      columnHelper.accessor("owner_name", {
        header: "Owner",
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.owner_name || "—"}</span>,
      }),
      columnHelper.accessor("column_count", {
        header: "Columns",
        cell: ({ row }) => (
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline" className="cursor-default">
                {row.original.column_count}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>{row.original.column_count} columns</TooltipContent>
          </Tooltip>
        ),
      }),
      columnHelper.accessor("metric_count", {
        header: "Metrics",
        cell: ({ row }) => (
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="outline" className="cursor-default">
                {row.original.metric_count}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>{row.original.metric_count} metrics</TooltipContent>
          </Tooltip>
        ),
      }),
      columnHelper.accessor("changed_on", {
        header: "Modified",
        cell: ({ row }) => {
          const date = new Date(row.original.changed_on);
          return <span className="text-muted-foreground text-sm">{date.toLocaleDateString()}</span>;
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const dataset = row.original;
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="icon" aria-label={`Actions for ${dataset.table_name}`}>
                  <MoreHorizontal />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => navigate(`/admin/settings/datasets/${dataset.id}/edit`)}>
                  Edit
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => navigate(`/explore?datasource_id=${dataset.id}`)}>
                  Explore
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setPendingDelete(dataset)}>
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      }),
    ],
    [navigate],
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
            <h1 className="text-2xl font-semibold">Datasets</h1>
            <p className="text-sm text-muted-foreground">Manage physical and virtual datasets.</p>
          </div>

          <Button onClick={() => navigate("/admin/settings/datasets/new")}>
            <Plus data-icon="inline-start" />
            Add Dataset
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
              placeholder="Search by dataset name"
              aria-label="Search by dataset name"
            />
          </div>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline">
                Database: {dbFilter ? (databasesQuery.data?.items.find(d => d.id === dbFilter)?.database_name ?? "...") : "All"}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuRadioGroup value={String(dbFilter ?? "")} onValueChange={(v) => { setPage(1); setDbFilter(v ? Number(v) : undefined); setSchemaFilter(undefined); }}>
                <DropdownMenuRadioItem value="">All Databases</DropdownMenuRadioItem>
                {databasesQuery.data?.items.map((db) => (
                  <DropdownMenuRadioItem key={db.id} value={String(db.id)}>{db.database_name}</DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" disabled={!dbFilter}>
                Schema: {schemaFilter ?? "All"}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuRadioGroup value={schemaFilter ?? ""} onValueChange={(v) => { setPage(1); setSchemaFilter(v || undefined); }}>
                <DropdownMenuRadioItem value="">All Schemas</DropdownMenuRadioItem>
                {(schemasQuery.data ?? []).map((s) => (
                  <DropdownMenuRadioItem key={s} value={s}>{s}</DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline">Type: {selectedType}</Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuRadioGroup
                value={selectedType}
                onValueChange={(value) => {
                  setPage(1);
                  setSelectedType(value);
                }}
              >
                {typeOptions.map((option) => (
                  <DropdownMenuRadioItem key={option} value={option}>
                    {option}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {datasetsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : null}

        {!datasetsQuery.isLoading && !hasRows ? (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Database className="size-4" />
                No datasets yet
              </CardTitle>
              <CardDescription>No datasets have been created yet.</CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={() => navigate("/admin/settings/datasets/new")}>Add Dataset</Button>
            </CardContent>
          </Card>
        ) : null}

        {!datasetsQuery.isLoading && hasRows ? (
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
                {table.getRowModel().rows.map((row) => (
                  <tr key={row.id} className="border-t align-middle">
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-3 py-2">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
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
              <AlertDialogTitle>Delete {pendingDelete?.table_name}?</AlertDialogTitle>
              <AlertDialogDescription>
                This will permanently delete the dataset and its associated columns and metrics.
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
                  deleteMutation.mutate({ id: pendingDelete.id, force: false });
                }}
              >
                Delete Dataset
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </TooltipProvider>
  );
}