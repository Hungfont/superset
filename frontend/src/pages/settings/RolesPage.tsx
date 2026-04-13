import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Link, useNavigate } from "react-router-dom";
import { MoreHorizontal, Plus } from "lucide-react";
import { z } from "zod";

import { rolesApi, type Role } from "@/api/roles";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useToast } from "@/hooks/use-toast";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const roleNameSchema = z.string().min(1, "Name is required").max(64, "Max 64 chars");

interface RoleFormState {
  name: string;
  error: string | null;
}

export default function RolesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { toast } = useToast();

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isUsersSheetOpen, setIsUsersSheetOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [formState, setFormState] = useState<RoleFormState>({ name: "", error: null });

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: rolesApi.getRoles,
  });

  const createRoleMutation = useMutation({
    mutationFn: rolesApi.createRole,
    onMutate: async (payload) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previousRoles = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      const optimisticRole: Role = {
        id: -Date.now(),
        name: payload.name,
        user_count: 0,
        permission_count: 0,
        built_in: false,
      };
      queryClient.setQueryData<Role[]>(["roles"], [...previousRoles, optimisticRole]);
      return { previousRoles };
    },
    onError: (error, _payload, context) => {
      if (context?.previousRoles) {
        queryClient.setQueryData(["roles"], context.previousRoles);
      }
      toast({ title: "Create failed", description: error.message, variant: "destructive" });
    },
    onSuccess: () => {
      setIsCreateOpen(false);
      setFormState({ name: "", error: null });
      toast({ title: "Role created" });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, name }: { id: number; name: string }) => rolesApi.updateRole(id, { name }),
    onSuccess: () => {
      setIsCreateOpen(false);
      setSelectedRole(null);
      setFormState({ name: "", error: null });
      toast({ title: "Role updated" });
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
    onError: (error) => {
      toast({ title: "Update failed", description: error.message, variant: "destructive" });
    },
  });

  const deleteRoleMutation = useMutation({
    mutationFn: rolesApi.deleteRole,
    onMutate: async (roleId) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previousRoles = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(
        ["roles"],
        previousRoles.filter((role) => role.id !== roleId),
      );
      return { previousRoles };
    },
    onError: (error, _id, context) => {
      if (context?.previousRoles) {
        queryClient.setQueryData(["roles"], context.previousRoles);
      }
      toast({ title: "Delete failed", description: error.message, variant: "destructive" });
    },
    onSuccess: () => {
      setIsDeleteOpen(false);
      setSelectedRole(null);
      toast({ title: "Role deleted" });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const roles = rolesQuery.data ?? [];

  const columns = useMemo<ColumnDef<Role>[]>(
    () => [
      { accessorKey: "name", header: "Name" },
      {
        accessorKey: "user_count",
        header: "Users",
        cell: ({ row }) => (
          <Button
            variant="ghost"
            className="px-0"
            onClick={() => {
              setSelectedRole(row.original);
              setIsUsersSheetOpen(true);
            }}
          >
            <Badge variant="secondary">{row.original.user_count}</Badge>
          </Button>
        ),
      },
      {
        accessorKey: "permission_count",
        header: "Permissions",
        cell: ({ row }) => (
          <Button
            variant="ghost"
            className="px-0"
            onClick={() => navigate(`/settings/roles/${row.original.id}/permissions`)}
          >
            <Badge>{row.original.permission_count}</Badge>
          </Button>
        ),
      },
      {
        id: "actions",
        header: "Actions",
        cell: ({ row }) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label={`Actions for ${row.original.name}`}>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() => {
                  setSelectedRole(row.original);
                  setFormState({ name: row.original.name, error: null });
                  setIsCreateOpen(true);
                }}
              >
                Edit
              </DropdownMenuItem>
              {row.original.built_in ? (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div>
                        <DropdownMenuItem disabled>Delete</DropdownMenuItem>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>Built-in roles cannot be deleted</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              ) : (
                <DropdownMenuItem
                  onClick={() => {
                    setSelectedRole(row.original);
                    setIsDeleteOpen(true);
                  }}
                >
                  Delete
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        ),
      },
    ],
    [navigate],
  );

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const isEditMode = selectedRole !== null && isCreateOpen;

  const handleOpenCreate = () => {
    setSelectedRole(null);
    setFormState({ name: "", error: null });
    setIsCreateOpen(true);
  };

  const handleSubmitRole = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const parsed = roleNameSchema.safeParse(formState.name.trim());
    if (!parsed.success) {
      setFormState((prev) => ({
        ...prev,
        error: parsed.error.issues[0]?.message ?? "Invalid name",
      }));
      return;
    }

    if (isEditMode && selectedRole) {
      updateRoleMutation.mutate({ id: selectedRole.id, name: parsed.data });
      return;
    }

    createRoleMutation.mutate({ name: parsed.data });
  };

  return (
    <main className="mx-auto w-full max-w-5xl p-6">
      <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Role Management</h1>
          <p className="text-sm text-muted-foreground">Create and manage RBAC roles for your workspace.</p>
        </div>
        <Button onClick={handleOpenCreate}>
          <Plus className="mr-2 h-4 w-4" />
          New Role
        </Button>
      </header>

      <div className="overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/40">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th key={header.id} className="px-4 py-3 text-left font-medium">
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {rolesQuery.isLoading ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">
                  Loading roles...
                </td>
              </tr>
            ) : table.getRowModel().rows.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">
                  No roles found.
                </td>
              </tr>
            ) : (
              table.getRowModel().rows.map((row) => (
                <tr key={row.id} tabIndex={0} className="border-t focus-within:bg-muted/30 hover:bg-muted/20">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-4 py-3">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent aria-labelledby="role-dialog-title">
          <DialogHeader>
            <DialogTitle id="role-dialog-title">{isEditMode ? "Edit Role" : "Create Role"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmitRole} className="mt-4 flex flex-col gap-3">
            <Input
              autoFocus
              placeholder="Role name"
              value={formState.name}
              onChange={(event) => setFormState({ name: event.target.value, error: null })}
              aria-invalid={formState.error !== null}
            />
            {formState.error ? <p className="text-sm text-destructive">{formState.error}</p> : null}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createRoleMutation.isPending || updateRoleMutation.isPending}>
                {isEditMode ? "Save" : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete role <strong>{selectedRole?.name}</strong>?</AlertDialogTitle>
            <AlertDialogDescription>
              This cannot be undone.
              {selectedRole && selectedRole.user_count > 0 ? (
                <span className="block mt-2 text-destructive">
                  This role still has {selectedRole.user_count} assigned users and will return a conflict.
                </span>
              ) : null}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedRole) {
                  deleteRoleMutation.mutate(selectedRole.id);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Sheet open={isUsersSheetOpen} onOpenChange={setIsUsersSheetOpen}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Users in {selectedRole?.name}</SheetTitle>
            <SheetDescription>
              User list API for roles is not available yet. This panel is ready for integration.
            </SheetDescription>
          </SheetHeader>
        </SheetContent>
      </Sheet>

      <div className="mt-6 text-sm text-muted-foreground">
        Need permission details? Open the matrix from each row, or visit <Link className="underline" to="/settings/roles/1/permissions">Role permissions</Link>.
      </div>
    </main>
  );
}
