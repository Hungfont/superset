import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Pencil, Plus, ShieldAlert, Trash2, Users } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { z } from "zod";

import { rolesApi, type Role } from "@/api/roles";
import { useToast } from "@/hooks/use-toast";
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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";

const roleNameSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(64, "Max 64 chars"),
});

type RoleNameValues = z.infer<typeof roleNameSchema>;

function tempRole(name: string): Role {
  return {
    id: -Date.now(),
    name,
    user_count: 0,
    permission_count: 0,
    built_in: false,
  };
}

export default function RolesPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { success, error } = useToast();

  const [isUpsertOpen, setIsUpsertOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [usersSheetRole, setUsersSheetRole] = useState<Role | null>(null);

  const form = useForm<RoleNameValues>({
    resolver: zodResolver(roleNameSchema),
    defaultValues: { name: "" },
  });

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: rolesApi.getRoles,
  });

  const createMutation = useMutation({
    mutationFn: rolesApi.createRole,
    onMutate: async (payload) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previous = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(["roles"], [...previous, tempRole(payload.name)]);
      return { previous };
    },
    onError: (err, _payload, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["roles"], context.previous);
      }
      error((err as Error).message || "Failed to create role");
    },
    onSuccess: () => {
      success("Role created");
      setIsUpsertOpen(false);
      form.reset({ name: "" });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ roleId, payload }: { roleId: number; payload: { name: string } }) =>
      rolesApi.updateRole(roleId, payload),
    onMutate: async ({ roleId, payload }) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previous = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(
        ["roles"],
        previous.map((role) => (role.id === roleId ? { ...role, name: payload.name } : role)),
      );
      return { previous };
    },
    onError: (err, _payload, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["roles"], context.previous);
      }
      error((err as Error).message || "Failed to update role");
    },
    onSuccess: () => {
      success("Role updated");
      setIsUpsertOpen(false);
      setSelectedRole(null);
      form.reset({ name: "" });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: rolesApi.deleteRole,
    onMutate: async (roleId) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previous = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(["roles"], previous.filter((role) => role.id !== roleId));
      return { previous };
    },
    onError: (err, _payload, context) => {
      if (context?.previous) {
        queryClient.setQueryData(["roles"], context.previous);
      }
      error((err as Error).message || "Failed to delete role");
    },
    onSuccess: () => {
      success("Role deleted");
      setIsDeleteOpen(false);
      setSelectedRole(null);
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const isDeleteBlocked = !!selectedRole && (selectedRole.built_in || selectedRole.user_count > 0);

  function openCreateDialog() {
    setSelectedRole(null);
    form.reset({ name: "" });
    setIsUpsertOpen(true);
  }

  function openEditDialog(role: Role) {
    setSelectedRole(role);
    form.reset({ name: role.name });
    setIsUpsertOpen(true);
  }

  function openDeleteDialog(role: Role) {
    setSelectedRole(role);
    setIsDeleteOpen(true);
  }

  function submitRole(values: RoleNameValues) {
    if (selectedRole) {
      updateMutation.mutate({ roleId: selectedRole.id, payload: { name: values.name } });
      return;
    }
    createMutation.mutate({ name: values.name });
  }

  const roles = rolesQuery.data ?? [];
  const columnHelper = createColumnHelper<Role>();

  const columns = useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Name",
        cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      }),
      columnHelper.display({
        id: "users",
        header: "Users",
        cell: ({ row }) => {
          const role = row.original;
          return (
            <Button
              variant="ghost"
              className="h-auto p-0"
              onClick={() => setUsersSheetRole(role)}
              aria-label={`View users for ${role.name}`}
            >
              <Badge variant="secondary">{role.user_count} users</Badge>
            </Button>
          );
        },
      }),
      columnHelper.display({
        id: "permissions",
        header: "Permissions",
        cell: ({ row }) => {
          const role = row.original;
          return (
            <Button
              variant="ghost"
              className="h-auto p-0"
              onClick={() => navigate(`/admin/settings/roles/${role.id}/permissions`)}
              aria-label={`Open permission matrix for ${role.name}`}
            >
              <Badge variant="outline">{role.permission_count} perms</Badge>
            </Button>
          );
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const role = row.original;
          return (
            <div className="flex items-center justify-center gap-2">
              {role.built_in && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span tabIndex={0}>
                      <ShieldAlert className="size-4 text-muted-foreground" aria-hidden="true" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>Built-in roles cannot be deleted</TooltipContent>
                </Tooltip>
              )}

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => openEditDialog(role)}
                    aria-label={`Edit ${role.name}`}
                  >
                    <Pencil className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Edit role</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    disabled={role.built_in}
                    onClick={() => openDeleteDialog(role)}
                    aria-label={`Delete ${role.name}`}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="left">
                  {role.built_in ? "Cannot delete built-in roles" : "Delete role"}
                </TooltipContent>
              </Tooltip>
            </div>
          );
        },
      }),
    ],
    [columnHelper, navigate],
  );

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <TooltipProvider>
      <div className="flex flex-col gap-4">
        <header className="flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold">Role Management</h1>
            <p className="text-sm text-muted-foreground">Admin CRUD for roles and RBAC counts.</p>
          </div>
          <Button onClick={openCreateDialog}>
            <Plus />
            New Role
          </Button>
        </header>

        {rolesQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : null}

        {rolesQuery.isError ? (
          <p className="text-sm text-destructive">{(rolesQuery.error as Error).message}</p>
        ) : null}

        {!rolesQuery.isLoading && !rolesQuery.isError ? (
          <div className="rounded-md border">
            <table className="w-full text-sm">
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
                  <tr key={row.id} tabIndex={0} className="border-t focus:outline-none focus:ring-2 focus:ring-ring">
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="px-3 py-2 align-middle">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}

        <Dialog
          open={isUpsertOpen}
          onOpenChange={(open) => {
            setIsUpsertOpen(open);
            if (!open) {
              setSelectedRole(null);
              form.reset({ name: "" });
            }
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{selectedRole ? "Edit role" : "Create role"}</DialogTitle>
              <DialogDescription>
                Provide a role name up to 64 characters.
              </DialogDescription>
            </DialogHeader>

            <Form {...form}>
              <form onSubmit={form.handleSubmit(submitRole)} className="flex flex-col gap-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Role name</FormLabel>
                      <FormControl>
                        <Input autoFocus placeholder="Role name" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setIsUpsertOpen(false)}
                    disabled={isSaving}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isSaving}>
                    Save Role
                  </Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        </Dialog>

        <AlertDialog
          open={isDeleteOpen}
          onOpenChange={(open) => {
            setIsDeleteOpen(open);
            if (!open) {
              setSelectedRole(null);
            }
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                Delete role <strong>{selectedRole?.name ?? ""}</strong>?
              </AlertDialogTitle>
              <AlertDialogDescription>
                This cannot be undone.
              </AlertDialogDescription>
              {selectedRole && selectedRole.user_count > 0 ? (
                <AlertDialogDescription className="text-destructive">
                  This role has assigned users and cannot be deleted.
                </AlertDialogDescription>
              ) : null}
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                disabled={isDeleteBlocked || deleteMutation.isPending}
                onClick={(event) => {
                  event.preventDefault();
                  if (!selectedRole || isDeleteBlocked) {
                    return;
                  }
                  deleteMutation.mutate(selectedRole.id);
                }}
              >
                Delete
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        <Sheet open={usersSheetRole !== null} onOpenChange={(open) => !open && setUsersSheetRole(null)}>
          <SheetContent side="right">
            <SheetHeader>
              <SheetTitle className="flex items-center gap-2">
                <Users className="size-4" />
                {usersSheetRole?.name} users
              </SheetTitle>
              <SheetDescription>
                User list details are provided by AUTH-010; this view currently shows aggregate counts only.
              </SheetDescription>
            </SheetHeader>
            <div className="mt-4">
              <Badge variant="secondary">{usersSheetRole?.user_count ?? 0} users assigned</Badge>
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </TooltipProvider>
  );
}
