import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { Pencil, Plus, Settings2, Trash2 } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { z } from "zod";

import { rolesApi } from "@/api/roles";
import {
  usersApi,
  type CreateUserPayload,
  type UpdateUserPayload,
  type UserSummary,
} from "@/api/users";
import { useToast } from "@/hooks/use-toast";
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
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

const createUserSchema = z.object({
  first_name: z.string().trim().min(1, "First name is required"),
  last_name: z.string().trim().min(1, "Last name is required"),
  username: z.string().trim().min(1, "Username is required"),
  email: z.string().email("Invalid email"),
  password: z.string().min(12, "Password must be at least 12 characters"),
  active: z.boolean(),
  role_ids: z.array(z.number()).min(1, "At least one role is required"),
});

const updateUserSchema = createUserSchema.omit({ password: true });

type CreateUserFormValues = z.infer<typeof createUserSchema>;
type UpdateUserFormValues = z.infer<typeof updateUserSchema>;

const defaultCreateValues: CreateUserFormValues = {
  first_name: "",
  last_name: "",
  username: "",
  email: "",
  password: "",
  active: true,
  role_ids: [],
};

const columnHelper = createColumnHelper<UserSummary>();

function getUserRoleIDs(user: UserSummary): number[] {
  return Array.isArray(user.role_ids) ? user.role_ids : [];
}

export default function UsersPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { success, error } = useToast();

  const [selectedUser, setSelectedUser] = useState<UserSummary | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeactivateOpen, setIsDeactivateOpen] = useState(false);
  const [editRoleIDs, setEditRoleIDs] = useState<number[]>([]);

  const usersQuery = useQuery({
    queryKey: ["admin-users"],
    queryFn: usersApi.getUsers,
  });

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: rolesApi.getRoles,
  });

  const createForm = useForm<CreateUserFormValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: defaultCreateValues,
  });

  const editForm = useForm<UpdateUserFormValues>({
    resolver: zodResolver(updateUserSchema),
    defaultValues: {
      first_name: "",
      last_name: "",
      username: "",
      email: "",
      active: true,
      role_ids: [],
    },
  });

  const createMutation = useMutation({
    mutationFn: (payload: CreateUserPayload) => usersApi.createUser(payload),
    onSuccess: () => {
      success("User created");
      setIsCreateOpen(false);
      createForm.reset(defaultCreateValues);
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to create user");
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ userID, payload }: { userID: number; payload: UpdateUserPayload }) =>
      usersApi.updateUser(userID, payload),
    onSuccess: () => {
      success("User updated");
      setIsEditOpen(false);
      setSelectedUser(null);
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to update user");
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: (userID: number) => usersApi.deactivateUser(userID),
    onSuccess: () => {
      success("User deactivated");
      setIsDeactivateOpen(false);
      setSelectedUser(null);
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to deactivate user");
    },
  });

  const users = usersQuery.data ?? [];
  const roles = rolesQuery.data ?? [];

  const roleNameByID = useMemo(() => {
    return roles.reduce<Record<number, string>>((acc, role) => {
      acc[role.id] = role.name;
      return acc;
    }, {});
  }, [roles]);

  const columns = useMemo(
    () => [
      columnHelper.accessor("username", {
        header: "Username",
        cell: ({ row }) => <span className="font-medium">{row.original.username}</span>,
      }),
      columnHelper.accessor("email", {
        header: "Email",
      }),
      columnHelper.display({
        id: "status",
        header: "Status",
        cell: ({ row }) => (
          <Badge variant={row.original.active ? "secondary" : "outline"}>
            {row.original.active ? "Active" : "Inactive"}
          </Badge>
        ),
      }),
      columnHelper.display({
        id: "roles",
        header: "Roles",
        cell: ({ row }) => {
          const names = getUserRoleIDs(row.original).map((roleID) => roleNameByID[roleID] ?? `#${roleID}`);
          return <span className="text-sm text-muted-foreground">{names.join(", ")}</span>;
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: ({ row }) => {
          const user = row.original;
          return (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={() => openEditDialog(user)}
                aria-label={`Edit ${user.username}`}
              >
                <Pencil className="size-4" />
              </Button>

              <Button
                variant="outline"
                size="icon"
                onClick={() => openDeactivateDialog(user)}
                aria-label={`Deactivate ${user.username}`}
              >
                <Trash2 className="size-4" />
              </Button>

              <Button
                variant="outline"
                size="sm"
                onClick={() => navigate(`/admin/settings/users/${user.id}`)}
                aria-label={`Manage roles for ${user.username}`}
              >
                <Settings2 className="size-4" />
                Roles
              </Button>
            </div>
          );
        },
      }),
    ],
    [navigate, roleNameByID],
  );

  const table = useReactTable({
    data: users,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  function openCreateDialog() {
    createForm.reset(defaultCreateValues);
    setIsCreateOpen(true);
  }

  function openEditDialog(user: UserSummary) {
    const userRoleIDs = getUserRoleIDs(user);
    setSelectedUser(user);
    setEditRoleIDs(userRoleIDs);
    editForm.reset({
      first_name: user.first_name,
      last_name: user.last_name,
      username: user.username,
      email: user.email,
      active: user.active,
      role_ids: userRoleIDs,
    });
    setIsEditOpen(true);
  }

  function openDeactivateDialog(user: UserSummary) {
    setSelectedUser(user);
    setIsDeactivateOpen(true);
  }

  function toggleCreateRole(roleID: number) {
    const current = createForm.getValues("role_ids");
    const set = new Set(current);
    if (set.has(roleID)) {
      set.delete(roleID);
    } else {
      set.add(roleID);
    }
    createForm.setValue("role_ids", [...set], { shouldValidate: true });
  }

  function toggleEditRole(roleID: number) {
    const set = new Set(editRoleIDs);
    if (set.has(roleID)) {
      set.delete(roleID);
    } else {
      set.add(roleID);
    }
    const updated = [...set];
    setEditRoleIDs(updated);
    editForm.setValue("role_ids", updated, { shouldValidate: true });
  }

  function submitCreate(values: CreateUserFormValues) {
    createMutation.mutate(values);
  }

  function submitEdit(values: UpdateUserFormValues) {
    if (!selectedUser) {
      return;
    }

    updateMutation.mutate({
      userID: selectedUser.id,
      payload: {
        ...values,
        role_ids: editRoleIDs,
      },
    });
  }

  function confirmDeactivate() {
    if (!selectedUser) {
      return;
    }
    deactivateMutation.mutate(selectedUser.id);
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">User Management</h1>
          <p className="text-sm text-muted-foreground">Manage users and navigate to role assignment.</p>
        </div>
        <Button onClick={openCreateDialog}>
          <Plus className="size-4" />
          New User
        </Button>
      </header>

      {usersQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : null}

      {usersQuery.isError ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load users</AlertTitle>
          <AlertDescription>{(usersQuery.error as Error).message}</AlertDescription>
        </Alert>
      ) : null}

      {!usersQuery.isLoading && !usersQuery.isError ? (
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

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create user</DialogTitle>
            <DialogDescription>Create an active user and assign initial roles.</DialogDescription>
          </DialogHeader>

          <Form {...createForm}>
            <form className="flex flex-col gap-3" onSubmit={createForm.handleSubmit(submitCreate)}>
              <FormField
                control={createForm.control}
                name="first_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>First name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="last_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Last name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input type="email" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Password</FormLabel>
                    <FormControl>
                      <Input type="password" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="flex flex-col gap-2 rounded-md border p-3">
                <p className="text-sm font-medium">Roles</p>
                <div className="flex flex-wrap gap-2">
                  {roles.map((role) => {
                    const selected = createForm.watch("role_ids").includes(role.id);
                    return (
                      <Button
                        key={role.id}
                        type="button"
                        size="sm"
                        variant={selected ? "default" : "outline"}
                        onClick={() => toggleCreateRole(role.id)}
                      >
                        {role.name}
                      </Button>
                    );
                  })}
                </div>
                {createForm.formState.errors.role_ids ? (
                  <p className="text-sm text-destructive">{createForm.formState.errors.role_ids.message}</p>
                ) : null}
              </div>

              <DialogFooter>
                <Button type="submit" disabled={createMutation.isPending}>
                  Save user
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <Dialog open={isEditOpen} onOpenChange={setIsEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit user</DialogTitle>
            <DialogDescription>Update profile fields and assigned roles.</DialogDescription>
          </DialogHeader>

          <Form {...editForm}>
            <form className="flex flex-col gap-3" onSubmit={editForm.handleSubmit(submitEdit)}>
              <FormField
                control={editForm.control}
                name="first_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>First name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={editForm.control}
                name="last_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Last name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={editForm.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={editForm.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input type="email" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="flex flex-col gap-2 rounded-md border p-3">
                <p className="text-sm font-medium">Roles</p>
                <div className="flex flex-wrap gap-2">
                  {roles.map((role) => {
                    const selected = editRoleIDs.includes(role.id);
                    return (
                      <Button
                        key={role.id}
                        type="button"
                        size="sm"
                        variant={selected ? "default" : "outline"}
                        onClick={() => toggleEditRole(role.id)}
                      >
                        {role.name}
                      </Button>
                    );
                  })}
                </div>
                {editRoleIDs.length === 0 ? (
                  <p className="text-sm text-destructive">At least one role is required</p>
                ) : null}
              </div>

              <DialogFooter>
                <Button type="submit" disabled={updateMutation.isPending || editRoleIDs.length === 0}>
                  Save user
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={isDeactivateOpen} onOpenChange={setIsDeactivateOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Deactivate user</AlertDialogTitle>
            <AlertDialogDescription>
              User {selectedUser?.username} will be marked inactive and signed out.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(event) => {
                event.preventDefault();
                confirmDeactivate();
              }}
              disabled={deactivateMutation.isPending}
            >
              Deactivate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
