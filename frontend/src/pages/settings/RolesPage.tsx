import { useMemo, useState } from "react";
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
import { useToast } from "@/hooks/use-toast";

const roleFormSchema = z.object({
  name: z.string().min(1, "Name is required").max(64, "Max 64 chars"),
});

type RoleFormValues = z.infer<typeof roleFormSchema>;

export default function RolesPage() {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const [isUpsertOpen, setIsUpsertOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isUsersSheetOpen, setIsUsersSheetOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);

  const form = useForm<RoleFormValues>({
    resolver: zodResolver(roleFormSchema),
    defaultValues: { name: "" },
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
      toast({ title: "Create failed", description: error.message, variant: "destructive" });
    },
    onSuccess: () => {
      setIsUpsertOpen(false);
      form.reset();
      toast({ title: "Role created" });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, name }: { id: number; name: string }) => rolesApi.updateRole(id, { name }),
    onSuccess: () => {
      setIsUpsertOpen(false);
      setSelectedRole(null);
      form.reset();
      toast({ title: "Role updated" });
      queryClient.invalidateQueries({ queryKey: ["roles"] });
    },
    onError: (error) => {
      toast({ title: "Update failed", description: error.message, variant: "destructive" });
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
                  setSelectedRole(row.original);
                  form.reset({ name: row.original.name });
                  setIsUpsertOpen(true);
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
    [form],
  );

  const table = useReactTable({
    data: roles,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const isEditMode = selectedRole !== null && isUpsertOpen;

  const handleOpenCreate = () => {
    setSelectedRole(null);
    form.reset({ name: "" });
    setIsUpsertOpen(true);
  };

  const handleSubmitRole = (values: RoleFormValues) => {
    const trimmedName = values.name.trim();
    if (isEditMode && selectedRole) {
      updateRoleMutation.mutate({ id: selectedRole.id, name: trimmedName });
      return;
    }
    createRoleMutation.mutate({ name: trimmedName });
  };

  return (
    <main className="mx-auto w-full max-w-5xl p-6">
      <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">AUTH-007 Role CRUD Management</h1>
          <p className="text-sm text-muted-foreground">Create, update, and delete RBAC roles.</p>
        </div>
        <Button onClick={handleOpenCreate}>
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

      <Dialog
        open={isUpsertOpen}
        onOpenChange={(open) => {
          setIsUpsertOpen(open);
          if (!open) {
            form.reset();
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{isEditMode ? "Edit Role" : "Create Role"}</DialogTitle>
            <DialogDescription>
              {isEditMode ? "Update role name." : "Create a new role for RBAC assignments."}
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmitRole)} className="mt-4 flex flex-col gap-3">
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
                <Button type="button" variant="outline" onClick={() => setIsUpsertOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={createRoleMutation.isPending || updateRoleMutation.isPending}>
                  {isEditMode ? "Save" : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete role <strong>{selectedRole?.name}</strong>?
            </AlertDialogTitle>
            <AlertDialogDescription>
              This cannot be undone.
              {selectedRole && selectedRole.user_count > 0 ? (
                <span className="mt-2 block text-destructive">
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
    </main>
  );
}
