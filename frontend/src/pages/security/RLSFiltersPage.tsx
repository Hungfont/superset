import { useState, useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Pencil, Plus, Shield, Trash2, Users, Table, Search } from "lucide-react";
import { toast as sonnerToast } from "sonner";
import { z } from "zod";

import { rlsFiltersApi, type RLSFilter, type CreateRLSFilterRequest } from "@/api/rlsFilters";
import { rolesApi } from "@/api/roles";
import { datasetsApi, type DatasetListResponse } from "@/api/datasets";

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
import { RLSFilterFormValues, rlsFilterSchema } from "@/lib/validations/rls";


const columnHelper = createColumnHelper<RLSFilter>();

export default function RLSFiltersPage() {
  const queryClient = useQueryClient();
  const [searchQ, setSearchQ] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [page] = useState(1);
  const pageSize = 20;

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingFilter, setEditingFilter] = useState<RLSFilter | null>(null);
  const [deleteFilterId, setDeleteFilterId] = useState<number | null>(null);
  const [selectedRoles, setSelectedRoles] = useState<number[]>([]);
  const [selectedTables, setSelectedTables] = useState<number[]>([]);

  const form = useForm<RLSFilterFormValues>({
    resolver: zodResolver(rlsFilterSchema),
    defaultValues: {
      name: "",
      filter_type: "Regular",
      clause: "",
      group_key: "",
      description: "",
      role_ids: [],
      table_ids: [],
    },
  });

  const { data: filtersData, isLoading: filtersLoading } = useQuery({
    queryKey: ["rls-filters", { page, pageSize, q: searchQ, filter_type: filterType }],
    queryFn: () => rlsFiltersApi.getFilters({ page, page_size: pageSize, q: searchQ, filter_type: filterType === "all" ? undefined : filterType }),
    staleTime: 30000,
  });

  const { data: rolesData } = useQuery({
    queryKey: ["roles"],
    queryFn: () => rolesApi.getRoles(),
    staleTime: 60000,
  });

  const { data: datasetsData } = useQuery<DatasetListResponse>({
    queryKey: ["datasets", { page: 1, page_size: 100 }],
    queryFn: () => datasetsApi.getDatasets({ page: 1, page_size: 100 }),
    staleTime: 60000,
  });

  const createMutation = useMutation({
    mutationFn: (data: CreateRLSFilterRequest) => rlsFiltersApi.createFilter(data),
    onSuccess: () => {
      sonnerToast("Filter created successfully");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      setDialogOpen(false);
      form.reset();
    },
    onError: (error: Error) => {
      sonnerToast(error.message);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number } & CreateRLSFilterRequest) => rlsFiltersApi.updateFilter(id, data),
    onSuccess: () => {
      sonnerToast("Filter updated successfully");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      setDialogOpen(false);
      setEditingFilter(null);
      form.reset();
    },
    onError: (error: Error) => {
      sonnerToast(error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => rlsFiltersApi.deleteFilter(id),
    onSuccess: () => {
      sonnerToast("Filter deleted successfully");
      queryClient.invalidateQueries({ queryKey: ["rls-filters"] });
      setDeleteFilterId(null);
    },
    onError: (error: Error) => {
      sonnerToast(error.message);
    },
  });

  const columns = useMemo(() => [
    columnHelper.accessor("name", {
      header: "Name",
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
      header: "Type",
      cell: ({ row }) => (
        <Badge variant={row.original.filter_type === "Regular" ? "outline" : "secondary"}>
          {row.original.filter_type}
        </Badge>
      ),
    }),
    columnHelper.accessor("clause", {
      header: "Clause",
      cell: ({ row }) => (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger className="max-w-[200px] truncate block font-mono text-sm">
              {row.original.clause.slice(0, 60)}...
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
      cell: ({ row }) => {
        const roles = row.original.roles || [];
        if (roles.length === 0) return null;
        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <div className="flex gap-1">
                  {roles.slice(0, 3).map((r) => (
                    <Badge key={r.id} variant="outline" className="text-xs">
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
      cell: ({ row }) => {
        const tables = row.original.tables || [];
        if (tables.length === 0) return null;
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
    columnHelper.display({
      id: "actions",
      header: "Actions",
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm">
              Actions
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => handleEdit(row.original)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit
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
      ),
    }),
  ], []);

  const table = useReactTable({
    data: filtersData?.data || [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const handleEdit = (filter: RLSFilter) => {
    setEditingFilter(filter);
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
    setDialogOpen(true);
  };

  const handleCreate = () => {
    setEditingFilter(null);
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
    setDialogOpen(true);
  };

  const onSubmit = (data: RLSFilterFormValues) => {
    if (editingFilter) {
      updateMutation.mutate({ id: editingFilter.id, ...data });
    } else {
      createMutation.mutate(data);
    }
  };

  const handleDelete = () => {
    if (deleteFilterId) {
      deleteMutation.mutate(deleteFilterId);
    }
  };

  const isFormLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="container mx-auto py-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Shield className="h-6 w-6" />
            Row Level Security
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

      <div className="flex gap-4 mb-6">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search filters..."
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select value={filterType} onValueChange={setFilterType}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Filter type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="Regular">Regular</SelectItem>
            <SelectItem value="Base">Base</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {filtersLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : filtersData?.data?.length === 0 ? (
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
        <div className="border rounded-md">
          <table className="w-full">
            <thead className="bg-muted/50">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th key={header.id} className="px-4 py-3 text-left text-sm font-medium">
                      {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="border-t">
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
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editingFilter ? `Edit RLS Filter: ${editingFilter.name}` : "Create RLS Filter"}
            </DialogTitle>
            <DialogDescription>
              Define a SQL WHERE clause to filter data access for selected roles
            </DialogDescription>
          </DialogHeader>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
                      <p className="text-sm text-amber-600 mt-1">
                        Base filters replace the entire WHERE clause. Use only when defining the base dataset scope for all users.
                      </p>
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
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          const current = field.value || "";
                          field.onChange(current + "{{current_user_id}}");
                        }}
                      >
                        {"{{current_user_id}}"}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          const current = field.value || "";
                          field.onChange(current + "{{current_username}}");
                        }}
                      >
                        {"{{current_username}}"}
                      </Button>
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
                            : `${selectedRoles.length} role(s) selected`}
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-[300px] p-0">
                        <div className="max-h-[300px] overflow-y-auto p-2">
                          {rolesData?.map((role) => (
                            <div key={role.id} className="flex items-center gap-2 p-2 hover:bg-muted rounded">
                              <Checkbox
                                checked={selectedRoles.includes(role.id)}
                                onCheckedChange={(checked) => {
                                  if (checked) {
                                    const updated = [...selectedRoles, role.id];
                                    setSelectedRoles(updated);
                                    field.onChange(updated);
                                  } else {
                                    const updated = selectedRoles.filter((id) => id !== role.id);
                                    setSelectedRoles(updated);
                                    field.onChange(updated);
                                  }
                                }}
                              />
                              <span>{role.name}</span>
                            </div>
                          ))}
                        </div>
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
                      <PopoverContent className="w-[300px] p-0">
                        <div className="max-h-[300px] overflow-y-auto p-2">
                          {datasetsData?.items?.map((ds) => (
                            <div key={ds.id} className="flex items-center gap-2 p-2 hover:bg-muted rounded">
                              <Checkbox
                                checked={selectedTables.includes(ds.id)}
                                onCheckedChange={(checked) => {
                                  if (checked) {
                                    const updated = [...selectedTables, ds.id];
                                    setSelectedTables(updated);
                                    field.onChange(updated);
                                  } else {
                                    const updated = selectedTables.filter((id) => id !== ds.id);
                                    setSelectedTables(updated);
                                    field.onChange(updated);
                                  }
                                }}
                              />
                              <span>{ds.table_name}</span>
                              <span className="text-muted-foreground text-sm">({ds.schema})</span>
                            </div>
                          ))}
                        </div>
                      </PopoverContent>
                    </Popover>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isFormLoading}>
                  {isFormLoading ? "Saving..." : editingFilter ? "Save Changes" : "Create Filter"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteFilterId !== null} onOpenChange={() => setDeleteFilterId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete RLS Filter?</AlertDialogTitle>
            <AlertDialogDescription>
              Deleting this filter will immediately remove data restrictions for all users assigned to this filter&apos;s roles.
              This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-red-600 hover:bg-red-700"
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete Filter"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}