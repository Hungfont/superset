import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from "@tanstack/react-table";
import { Pencil, Plus, Shield, Trash2, Users, Table, Search, Loader2 } from "lucide-react";
import { toast as sonnerToast } from "sonner";

import { rlsFiltersApi, type RLSFilter, type CreateRLSFilterRequest } from "@/api/rlsFilters";
import { rolesApi, type Role } from "@/api/roles";
import { datasetsApi, type DatasetListResponse, type DatasetWithCounts } from "@/api/datasets";
import { useRLSStore } from "@/stores/rlsStore";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { DEFAULT_FORM_VALUES, type RLSFilterFormValues, rlsFilterSchema } from "@/lib/validations/rls";

const columnHelper = createColumnHelper<RLSFilter>();

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

function getRoleBadgeVariant(roleName: string): "default" | "secondary" | "outline" {
  const lower = roleName.toLowerCase();
  if (lower === "admin") return "default";
  if (lower === "alpha") return "secondary";
  return "outline";
}

function SortableHeader({ label }: { label: string }) {
  return (
    <span className="flex items-center gap-1 select-none">
      {label}
    </span>
  );
}

export default function RLSFiltersPage() {
  const queryClient = useQueryClient();
  const {
    dialogOpen, editingFilter, deleteFilterId,
    setDialogOpen, setDeleteFilterId, openCreate, openEdit, reset: resetStore,
  } = useRLSStore();

  const [searchQ, setSearchQ] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [roleIdFilter, setRoleIdFilter] = useState<string>("all");
  const [page, setPage] = useState(1);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [activeTab, setActiveTab] = useState("filters");
  const pageSize = 20;

  const [selectedRoles, setSelectedRoles] = useState<number[]>([]);
  const [selectedTables, setSelectedTables] = useState<number[]>([]);
  const [deleteReady, setDeleteReady] = useState(false);
  const deleteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [rowFlashId, setRowFlashId] = useState<number | null>(null);

  const form = useForm<RLSFilterFormValues>({
    resolver: zodResolver(rlsFilterSchema),
    defaultValues: DEFAULT_FORM_VALUES,
  });

  const { data: filtersData, isLoading: filtersLoading } = useQuery({
    queryKey: ["rls-filters", { page, pageSize, q: searchQ, filter_type: filterType, role_id: roleIdFilter }],
    queryFn: () => rlsFiltersApi.getFilters({
      page,
      page_size: pageSize,
      q: searchQ || undefined,
      filter_type: filterType === "all" ? undefined : filterType,
      role_id: roleIdFilter !== "all" ? Number(roleIdFilter) : undefined,
    }),
    staleTime: 30000,
  });

  const { data: rolesData } = useQuery<Role[]>({
    queryKey: ["roles"],
    queryFn: () => rolesApi.getRoles(),
    staleTime: 60000,
  });

  const { data: datasetsData } = useQuery<DatasetListResponse>({
    queryKey: ["datasets", { page: 1, page_size: 100 }],
    queryFn: () => datasetsApi.getDatasets({ page: 1, page_size: 100 }),
    staleTime: 60000,
  });

  const datasetsByDB = useMemo(() => {
    const items = datasetsData?.items || [];
    const groups = new Map<string, DatasetWithCounts[]>();
    for (const ds of items) {
      const dbName = ds.database_name || "Unknown";
      const existing = groups.get(dbName) || [];
      existing.push(ds);
      groups.set(dbName, existing);
    }
    return groups;
  }, [datasetsData]);

  const createMutation = useMutation({
    mutationFn: (data: CreateRLSFilterRequest) => rlsFiltersApi.createFilter(data),
    onSuccess: (result) => {
      sonnerToast("Filter created");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      resetStore();
      form.reset();
      setRowFlashId(result.id);
      setTimeout(() => setRowFlashId(null), 600);
    },
    onError: (error: Error) => {
      sonnerToast(error.message, { style: { backgroundColor: "var(--destructive)", color: "var(--destructive-foreground)" } });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number } & CreateRLSFilterRequest) => rlsFiltersApi.updateFilter(id, data),
    onSuccess: (result) => {
      sonnerToast("Filter updated");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      resetStore();
      form.reset();
      setRowFlashId(result.id);
      setTimeout(() => setRowFlashId(null), 600);
    },
    onError: (error: Error) => {
      sonnerToast(error.message, { style: { backgroundColor: "var(--destructive)", color: "var(--destructive-foreground)" } });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => rlsFiltersApi.deleteFilter(id),
    onSuccess: () => {
      sonnerToast("Filter deleted");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      resetStore();
    },
    onError: (error: Error) => {
      sonnerToast(error.message, { style: { backgroundColor: "var(--destructive)", color: "var(--destructive-foreground)" } });
    },
  });

  useEffect(() => {
    if (deleteFilterId !== null) {
      setDeleteReady(false);
      deleteTimerRef.current = setTimeout(() => setDeleteReady(true), 1500);
    } else {
      setDeleteReady(false);
    }
    return () => {
      if (deleteTimerRef.current) clearTimeout(deleteTimerRef.current);
    };
  }, [deleteFilterId]);

  const allGroupKeysEmpty = useMemo(() => {
    const data = filtersData?.data || [];
    return data.length === 0 || data.every((f) => !f.group_key);
  }, [filtersData]);

  const columns = useMemo(() => [
    columnHelper.accessor("name", {
      header: () => <SortableHeader label="Name" />,
      cell: ({ row }) => (
        <span
          className="cursor-pointer font-medium hover:underline"
          onClick={() => handleEdit(row.original)}
        >
          {row.original.name}
        </span>
      ),
    }),
    columnHelper.accessor("filter_type", {
      header: () => <SortableHeader label="Type" />,
      cell: ({ row }) => (
        <Badge
          variant="outline"
          className={row.original.filter_type === "Regular"
            ? "border-blue-300 text-blue-700"
            : "border-amber-300 text-amber-700"}
        >
          {row.original.filter_type}
        </Badge>
      ),
    }),
    columnHelper.accessor("clause", {
      header: () => <SortableHeader label="Clause" />,
      enableSorting: false,
      cell: ({ row }) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger className="max-w-[200px] truncate block font-mono text-sm">
              {row.original.clause.slice(0, 60)}{row.original.clause.length > 60 ? "..." : ""}
            </TooltipTrigger>
            <TooltipContent>
              <pre className="max-w-[400px] whitespace-pre-wrap font-mono text-xs">
                {row.original.clause}
              </pre>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ),
    }),
    columnHelper.accessor("group_key", {
      header: "Group",
      cell: ({ row }) => row.original.group_key ? (
        <Badge variant="outline" className="font-mono text-xs">
          {row.original.group_key}
        </Badge>
      ) : null,
    }),
    columnHelper.accessor("roles", {
      header: "Roles",
      enableSorting: false,
      cell: ({ row }) => {
        const roles = row.original.roles || [];
        if (roles.length === 0) return <span className="text-muted-foreground text-sm">—</span>;
        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <div className="flex gap-1">
                  {roles.slice(0, 3).map((r) => (
                    <Badge key={r.id} variant={getRoleBadgeVariant(r.name)} className="text-xs">
                      {r.name}
                    </Badge>
                  ))}
                  {roles.length > 3 && (
                    <Badge variant="outline" className="text-xs">
                      +{roles.length - 3}
                    </Badge>
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                {roles.map((r) => r.name).join(", ")}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        );
      },
    }),
    columnHelper.accessor("tables", {
      header: "Tables",
      enableSorting: false,
      cell: ({ row }) => {
        const tables = row.original.tables || [];
        if (tables.length === 0) return <span className="text-muted-foreground text-sm">—</span>;
        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <div className="flex gap-1">
                  {tables.slice(0, 3).map((t) => (
                    <Badge key={t.datasource_id} variant="outline" className="text-xs">
                      {t.table_name}
                    </Badge>
                  ))}
                  {tables.length > 3 && (
                    <Badge variant="outline" className="text-xs">
                      +{tables.length - 3}
                    </Badge>
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                {tables.map((t) => `${t.table_name} (${t.database_name})`).join(", ")}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        );
      },
    }),
    columnHelper.accessor("created_by", {
      header: "Created By",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.created_by || "—"}
        </span>
      ),
    }),
    columnHelper.accessor("created_on", {
      header: () => <SortableHeader label="Created" />,
      cell: ({ row }) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger className="text-sm text-muted-foreground">
              {relativeTime(row.original.created_on)}
            </TooltipTrigger>
            <TooltipContent>
              {new Date(row.original.created_on).toLocaleString()}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ),
    }),
    columnHelper.accessor("changed_on", {
      header: () => <SortableHeader label="Modified" />,
      cell: ({ row }) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger className="text-sm text-muted-foreground">
              {relativeTime(row.original.changed_on)}
            </TooltipTrigger>
            <TooltipContent>
              {new Date(row.original.changed_on).toLocaleString()}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      ),
    }),
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => {
        const name = row.original.name;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" aria-label={`Filter actions for ${name}`}>
                Actions
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleEdit(row.original)}>
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleEdit(row.original)}>
                <Users className="mr-2 h-4 w-4" />
                Manage Roles
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleEdit(row.original)}>
                <Table className="mr-2 h-4 w-4" />
                Manage Datasets
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => setDeleteFilterId(row.original.id)}
                className="text-red-600"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    }),
  ], []);

  const table = useReactTable({
    data: filtersData?.data || [],
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  useEffect(() => {
    const col = table.getColumn("group_key");
    if (col) col.toggleVisibility(!allGroupKeysEmpty);
  }, [table, allGroupKeysEmpty]);

  const totalPages = filtersData?.pages || 1;

  const handleEdit = useCallback((filter: RLSFilter) => {
    openEdit(filter);
    form.reset({
      name: filter.name,
      filter_type: filter.filter_type,
      clause: filter.clause,
      group_key: filter.group_key,
      description: filter.description,
      role_ids: filter.roles?.map((r) => r.id) || [],
      table_ids: filter.tables?.map((t) => t.datasource_id) || [],
    });
    setSelectedRoles(filter.roles?.map((r) => r.id) || []);
    setSelectedTables(filter.tables?.map((t) => t.datasource_id) || []);
  }, [form, openEdit]);

  const handleCreate = useCallback(() => {
    openCreate();
    form.reset({
      name: "",
      filter_type: "Regular",
      clause: "",
      group_key: "",
      description: "",
      role_ids: [],
      table_ids: [],
    });
    setSelectedRoles([]);
    setSelectedTables([]);
  }, [form, openCreate]);

  const onSubmit = (data: RLSFilterFormValues) => {
    if (editingFilter) {
      updateMutation.mutate({ id: editingFilter.id, ...data });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleDelete = () => {
    if (deleteFilterId && deleteReady) {
      deleteMutation.mutate(deleteFilterId);
    }
  };

  const isFormLoading = createMutation.isPending || updateMutation.isPending;
  const deletingFilterName = deleteFilterId
    ? (filtersData?.data || []).find((f) => f.id === deleteFilterId)?.name || ""
    : "";

  const filtersTab = (
    <>
      <div className="flex gap-4 mb-6">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search filters..."
            value={searchQ}
            onChange={(e) => { setSearchQ(e.target.value); setPage(1); }}
            className="pl-10"
          />
        </div>
        <Select value={filterType} onValueChange={(v) => { setFilterType(v); setPage(1); }}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="Regular">Regular</SelectItem>
            <SelectItem value="Base">Base</SelectItem>
          </SelectContent>
        </Select>
        <Select value={roleIdFilter} onValueChange={(v) => { setRoleIdFilter(v); setPage(1); }}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="All Roles" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Roles</SelectItem>
            {rolesData?.map((role) => (
              <SelectItem key={role.id} value={String(role.id)}>
                {role.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {filtersLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !filtersData?.data || filtersData.data.length === 0 ? (
        <div className="text-center py-12">
          <Shield className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
          <h3 className="text-lg font-medium mb-2">No RLS filters configured</h3>
          <p className="text-muted-foreground mb-4">
            Create your first filter to restrict data access by role
          </p>
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            Add Filter
          </Button>
        </div>
      ) : (
        <>
          <div className="border rounded-md">
            <table className="w-full">
              <thead className="bg-muted/50">
                {table.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => {
                      const canSort = header.column.getCanSort();
                      return (
                        <th
                          key={header.id}
                          className={`px-4 py-3 text-left text-sm font-medium ${canSort ? "cursor-pointer select-none" : ""}`}
                          onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                        >
                          {header.isPlaceholder
                            ? null
                            : flexRender(header.column.columnDef.header, header.getContext())}
                        </th>
                      );
                    })}
                  </tr>
                ))}
              </thead>
              <tbody>
                {table.getRowModel().rows.map((row) => (
                  <tr
                    key={row.id}
                    className={`border-t hover:bg-muted/30 transition-colors ${
                      rowFlashId === row.original.id
                        ? "animate-in fade-in slide-in-from-left-2 duration-500"
                        : ""
                    }`}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-4 py-3">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {totalPages > 1 && (
            <div className="mt-4 flex justify-center">
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      aria-disabled={page <= 1}
                      className={page <= 1 ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                  </PaginationItem>
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
                    <PaginationItem key={p}>
                      <PaginationLink
                        isActive={p === page}
                        onClick={() => setPage(p)}
                      >
                        {p}
                      </PaginationLink>
                    </PaginationItem>
                  ))}
                  <PaginationItem>
                    <PaginationNext
                      onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                      aria-disabled={page >= totalPages}
                      className={page >= totalPages ? "pointer-events-none opacity-50" : "cursor-pointer"}
                    />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </>
      )}
    </>
  );

  return (
    <div className="container mx-auto py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="h-6 w-6" />
            Row Level Security
            <Badge variant="secondary" className="ml-2">Admin</Badge>
          </h1>
          <p className="text-muted-foreground">
            Restrict data access by role using SQL filter clauses
          </p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          Add Filter
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="mb-6">
          <TabsTrigger value="filters">Filters</TabsTrigger>
          <TabsTrigger value="audit">Audit Log</TabsTrigger>
        </TabsList>

        <TabsContent value="filters">
          {filtersTab}
        </TabsContent>

        <TabsContent value="audit">
          <div className="text-center py-12">
            <Shield className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">Audit Log</h3>
            <p className="text-muted-foreground">
              Filter creation, update, and deletion events will appear here.
            </p>
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={dialogOpen} onOpenChange={(open) => {
        if (!open) {
          resetStore();
          form.reset(DEFAULT_FORM_VALUES);
        } else {
          setDialogOpen(open);
        }
      }}>
        <DialogContent className="max-w-2xl max-h-[90vh] p-0">
          <DialogHeader className="px-6 pt-6">
            <DialogTitle>
              {editingFilter ? `Edit RLS Filter: ${editingFilter.name}` : "Create RLS Filter"}
            </DialogTitle>
            <DialogDescription>
              Define a SQL WHERE clause to filter data access for selected roles
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="max-h-[calc(90vh-140px)] px-6">
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pb-6">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Name</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g. tenant_isolation" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="filter_type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Filter Type</FormLabel>
                      <Select
                        onValueChange={field.onChange}
                        defaultValue={field.value}
                        value={field.value}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Regular">Regular — AND appended to WHERE</SelectItem>
                          <SelectItem value="Base">Base — Replaces WHERE entirely</SelectItem>
                        </SelectContent>
                      </Select>
                      {field.value === "Base" && (
                        <Alert variant="destructive" className="mt-2 border-amber-300 bg-amber-50 text-amber-800 dark:bg-amber-950 dark:text-amber-200">
                          <AlertDescription>
                            Base filters replace the entire WHERE clause. Use only when defining the base dataset scope for all users.
                          </AlertDescription>
                        </Alert>
                      )}
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="clause"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Clause</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="e.g. org_id = {{current_user_id}}"
                          className="font-mono min-h-[120px]"
                          {...field}
                        />
                      </FormControl>
                      <div className="flex gap-2 mt-2">
                        <Badge
                          variant="outline"
                          className="font-mono text-xs cursor-pointer hover:bg-muted transition-colors"
                          onClick={() => {
                            const current = field.value || "";
                            field.onChange(current + "{{current_user_id}}");
                          }}
                        >
                          {"{{current_user_id}}"}
                        </Badge>
                        <Badge
                          variant="outline"
                          className="font-mono text-xs cursor-pointer hover:bg-muted transition-colors"
                          onClick={() => {
                            const current = field.value || "";
                            field.onChange(current + "{{current_username}}");
                          }}
                        >
                          {"{{current_username}}"}
                        </Badge>
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="group_key"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Group Key (optional)</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g. org_group" {...field} />
                      </FormControl>
                      <p className="text-sm text-muted-foreground">
                        Filters with the same group key are OR'd together. Leave empty to AND with all others.
                      </p>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Description (optional)</FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="Optional description..."
                          className="min-h-[60px]"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="role_ids"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Roles</FormLabel>
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="outline" className="w-full justify-start">
                            <Users className="mr-2 h-4 w-4" />
                            {selectedRoles.length === 0
                              ? "Select roles..."
                              : `${selectedRoles.length} of ${rolesData?.length || 0} role(s) selected`}
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-[360px] p-0" align="start">
                          <Command>
                            <CommandInput placeholder="Search roles..." />
                            <CommandList>
                              <CommandEmpty>No roles found.</CommandEmpty>
                              <CommandGroup>
                                {rolesData?.map((role) => {
                                  const isSelected = selectedRoles.includes(role.id);
                                  return (
                                    <CommandItem
                                      key={role.id}
                                      onSelect={() => {
                                        if (isSelected) {
                                          const updated = selectedRoles.filter((id) => id !== role.id);
                                          setSelectedRoles(updated);
                                          field.onChange(updated);
                                        } else {
                                          const updated = [...selectedRoles, role.id];
                                          setSelectedRoles(updated);
                                          field.onChange(updated);
                                        }
                                      }}
                                    >
                                      <Checkbox checked={isSelected} className="mr-2" />
                                      <span>{role.name}</span>
                                      <Badge variant={getRoleBadgeVariant(role.name)} className="text-xs ml-auto">
                                        {role.name === "Admin" ? "Admin" : role.name === "Alpha" ? "Alpha" : "Gamma"}
                                      </Badge>
                                    </CommandItem>
                                  );
                                })}
                              </CommandGroup>
                            </CommandList>
                          </Command>
                        </PopoverContent>
                      </Popover>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="table_ids"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Datasets</FormLabel>
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="outline" className="w-full justify-start">
                            <Table className="mr-2 h-4 w-4" />
                            {selectedTables.length === 0
                              ? "Select datasets..."
                              : `${selectedTables.length} dataset(s) selected`}
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-[360px] p-0" align="start">
                          <Command>
                            <CommandInput placeholder="Search datasets..." />
                            <CommandList>
                              <CommandEmpty>No datasets found.</CommandEmpty>
                              {Array.from(datasetsByDB.entries()).map(([dbName, dss]) => (
                                <CommandGroup key={dbName} heading={dbName}>
                                  {dss.map((ds) => {
                                    const isSelected = selectedTables.includes(ds.id);
                                    return (
                                      <CommandItem
                                        key={ds.id}
                                        onSelect={() => {
                                          if (isSelected) {
                                            const updated = selectedTables.filter((id) => id !== ds.id);
                                            setSelectedTables(updated);
                                            field.onChange(updated);
                                          } else {
                                            const updated = [...selectedTables, ds.id];
                                            setSelectedTables(updated);
                                            field.onChange(updated);
                                          }
                                        }}
                                      >
                                        <Checkbox checked={isSelected} className="mr-2" />
                                        <div className="flex flex-col">
                                          <span>{ds.table_name}</span>
                                          <span className="text-xs text-muted-foreground">{ds.schema}</span>
                                        </div>
                                      </CommandItem>
                                    );
                                  })}
                                </CommandGroup>
                              ))}
                            </CommandList>
                          </Command>
                        </PopoverContent>
                      </Popover>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter className="pt-2">
                  <Button type="button" variant="outline" onClick={() => { resetStore(); form.reset(DEFAULT_FORM_VALUES); }}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isFormLoading}>
                    {isFormLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    {editingFilter ? "Save Changes" : "Create Filter"}
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </ScrollArea>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteFilterId !== null} onOpenChange={() => {
        setDeleteFilterId(null);
        setDeleteReady(false);
      }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete RLS Filter?</AlertDialogTitle>
            <AlertDialogDescription>
              Deleting &apos;{deletingFilterName}&apos; will immediately remove data restrictions for all users assigned to this filter&apos;s roles.
              This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-red-600 hover:bg-red-700"
              disabled={deleteMutation.isPending || !deleteReady}
            >
              {deleteMutation.isPending ? "Deleting..." : !deleteReady ? "Delete Filter (wait...)" : "Delete Filter"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
