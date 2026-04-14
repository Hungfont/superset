import { useCallback, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { MoreHorizontal, Plus } from "lucide-react";
import { z } from "zod";

import { rolesApi, type Role } from "@/api/roles";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
import { useLoading } from "@/hooks/useLoading";
import { useToast } from "@/hooks/use-toast";

const roleFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name is required")
    .max(64, "Max 64 chars"),
});

type RoleFormValues = z.infer<typeof roleFormSchema>;

const DEFAULT_ROLE_FORM_VALUES: RoleFormValues = {
  name: "",
};

export default function RolesPage() {
  const queryClient = useQueryClient();
  const { success, error: notifyError } = useToast();
  const { isLoading, withLoading } = useLoading();

  const [isRoleDialogOpen, setIsRoleDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<"create" | "edit">("create");
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [usersSheetRole, setUsersSheetRole] = useState<Role | null>(null);

  const form = useForm<RoleFormValues>({
    resolver: zodResolver(roleFormSchema),
    defaultValues: DEFAULT_ROLE_FORM_VALUES,
  });

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
      notifyError("Create failed", { description: error.message });
    },
    onSuccess: () => {
      success("Role created");
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, name }: { id: number; name: string }) => rolesApi.updateRole(id, { name }),
    onMutate: async (payload) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previousRoles = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(
        ["roles"],
        previousRoles.map((role) =>
          role.id === payload.id
            ? {
                ...role,
                name: payload.name,
              }
            : role,
        ),
      );
      return { previousRoles };
    },
    onError: (error, _payload, context) => {
      if (context?.previousRoles) {
        queryClient.setQueryData(["roles"], context.previousRoles);
      }
      notifyError("Update failed", { description: error.message });
    },
    onSuccess: () => {
      success("Role updated");
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const deleteRoleMutation = useMutation({
    mutationFn: rolesApi.deleteRole,
    onMutate: async (roleID) => {
      await queryClient.cancelQueries({ queryKey: ["roles"] });
      const previousRoles = queryClient.getQueryData<Role[]>(["roles"]) ?? [];
      queryClient.setQueryData<Role[]>(
        ["roles"],
        previousRoles.filter((role) => role.id !== roleID),
      );
      return { previousRoles };
    },
    onError: (error, _id, context) => {
      if (context?.previousRoles) {
        queryClient.setQueryData(["roles"], context.previousRoles);
      }
      notifyError("Delete failed", { description: error.message });
    },
    onSuccess: () => {
      success("Role deleted");
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const roles = rolesQuery.data ?? [];

  const openCreateDialog = useCallback(() => {
    setDialogMode("create");
    setSelectedRole(null);
    form.reset(DEFAULT_ROLE_FORM_VALUES);
    setIsRoleDialogOpen(true);
  }, [form]);

  const openEditDialog = useCallback(
    (role: Role) => {
      setDialogMode("edit");
      setSelectedRole(role);
      form.reset({ name: role.name });
      setIsRoleDialogOpen(true);
    },
    [form],
  );

  const closeRoleDialog = useCallback(
    (open: boolean) => {
      setIsRoleDialogOpen(open);
      if (!open) {
        setSelectedRole(null);
        form.reset(DEFAULT_ROLE_FORM_VALUES);
      }
    },
    [form],
  );

  const openDeleteDialog = useCallback((role: Role) => {
    setSelectedRole(role);
    setIsDeleteDialogOpen(true);
  }, []);

  const handleDeleteDialogChange = useCallback((open: boolean) => {
    setIsDeleteDialogOpen(open);
    if (!open) {
      setSelectedRole(null);
    }
  }, []);

  const handleDeleteRole = useCallback(async () => {
    if (!selectedRole) {
      return;
    }

    try {
      await withLoading("delete", async () => {
        await deleteRoleMutation.mutateAsync(selectedRole.id);
      });
      handleDeleteDialogChange(false);
    } catch {
      return;
    }
  }, [deleteRoleMutation, handleDeleteDialogChange, selectedRole, withLoading]);

  const submitRoleForm = form.handleSubmit(async (values) => {
    try {
      if (dialogMode === "edit" && selectedRole) {
        await withLoading("update", async () => {
          await updateRoleMutation.mutateAsync({
            id: selectedRole.id,
            name: values.name,
          });
        });
      } else {
        await withLoading("create", async () => {
          await createRoleMutation.mutateAsync({ name: values.name });
        });
      }

      closeRoleDialog(false);
    } catch {
      return;
    }
  });

  const isRoleMutationLoading = isLoading("create") || isLoading("update");
  const isDeleteLoading = isLoading("delete");

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
              setUsersSheetRole(row.original);
            }}
          >
            <Badge variant="secondary">{row.original.user_count}</Badge>
          </Button>
        ),
      },
      {
        accessorKey: "permission_count",
        header: "Permissions",
        cell: ({ row }) => <Badge>{row.original.permission_count}</Badge>,
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
                  openEditDialog(row.original);
                }}
              >
                Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
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
                    openDeleteDialog(row.original);
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
    [openDeleteDialog, openEditDialog],
  );

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <main className="mx-auto w-full max-w-5xl p-6">
      <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">AUTH-007 Role CRUD Management</h1>
          <p className="text-sm text-muted-foreground">Create, update, and delete RBAC roles.</p>
        </div>
        <Button onClick={openCreateDialog}>
          <Plus className="mr-2 h-4 w-4" />
          New Role
        </Button>
      </header>

      {rolesQuery.isError ? (
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <AlertTitle>Unable to load roles</AlertTitle>
          <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <span>{rolesQuery.error.message}</span>
            <Button type="button" variant="outline" onClick={() => void rolesQuery.refetch()}>
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      ) : (
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
                  <tr key={row.id} tabIndex={0} className="border-t hover:bg-muted/20 focus-within:bg-muted/30">
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
      )}

      <Dialog open={isRoleDialogOpen} onOpenChange={closeRoleDialog}>
        <DialogContent aria-labelledby="role-dialog-title">
          <DialogHeader>
            <DialogTitle id="role-dialog-title">{dialogMode === "create" ? "Create role" : "Edit role"}</DialogTitle>
            <DialogDescription>
              {dialogMode === "create"
                ? "Add a new role for RBAC access control."
                : "Update the role name."}
            </DialogDescription>
          </DialogHeader>

          <Form {...form}>
            <form className="space-y-4" onSubmit={submitRoleForm}>
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
                  onClick={() => {
                    closeRoleDialog(false);
                  }}
                  disabled={isRoleMutationLoading}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={isRoleMutationLoading}>
                  {dialogMode === "create" ? "Create" : "Save"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isDeleteDialogOpen} onOpenChange={handleDeleteDialogChange}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete role {selectedRole ? `\"${selectedRole.name}\"` : ""}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This cannot be undone.
              {selectedRole && selectedRole.user_count > 0 ? (
                <span className="block pt-2 font-medium text-foreground">
                  {selectedRole.user_count} assigned users will block deletion.
                </span>
              ) : null}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleteLoading}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={isDeleteLoading}
              onClick={(event) => {
                event.preventDefault();
                void handleDeleteRole();
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Sheet
        open={Boolean(usersSheetRole)}
        onOpenChange={(open) => {
          if (!open) {
            setUsersSheetRole(null);
          }
        }}
      >
        <SheetContent>
          <SheetHeader>
            <SheetTitle>{usersSheetRole ? `${usersSheetRole.name} users` : "Role users"}</SheetTitle>
            <SheetDescription>
              {usersSheetRole
                ? `${usersSheetRole.user_count} assigned users`
                : "No role selected."}
            </SheetDescription>
          </SheetHeader>
        </SheetContent>
      </Sheet>
    </main>
  );
}
