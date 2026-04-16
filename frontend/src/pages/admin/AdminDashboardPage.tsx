import { Plus, ShieldCheck, UserCog, Users, Waypoints } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useForm } from "react-hook-form";
import z from "zod";
import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Role, rolesApi } from "@/api/roles";
import { useToast } from "@/hooks/use-toast";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";

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

export default function AdminDashboardPage() {
    const queryClient = useQueryClient();
    const { success, error } = useToast();

    const [isUpsertOpen, setIsUpsertOpen] = useState(false);
    const [isDeleteOpen, setIsDeleteOpen] = useState(false);
    const [selectedRole, setSelectedRole] = useState<Role | null>(null);
    const [usersSheetRole, setUsersSheetRole] = useState<Role | null>(null);

    
  const form = useForm<RoleNameValues>({
    resolver: zodResolver(roleNameSchema),
    defaultValues: { name: "" },
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

  function submitRole(values: RoleNameValues) {
    if (selectedRole) {
      updateMutation.mutate({ roleId: selectedRole.id, payload: { name: values.name } });
      return;
    }
    createMutation.mutate({ name: values.name });
  }
  return (
    <div className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold">Admin Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Khu vuc quan tri, chi role Admin moi co quyen truy cap.
        </p>
      </header>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <ShieldCheck className="h-4 w-4" />
              Role Control
            </CardTitle>
            <CardDescription>Quan ly role va policy cho he thong</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="secondary">/admin/settings/roles</Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Waypoints className="h-4 w-4" />
              API Access
            </CardTitle>
            <CardDescription>Kiem soat cac API quan tri</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="outline">Admin-only</Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <UserCog className="h-4 w-4" />
              Security
            </CardTitle>
            <CardDescription>Phan quyen va bao mat theo role</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge>RBAC</Badge>
          </CardContent>
        </Card>
      </div>

      <div>
        <Button onClick={openCreateDialog}>
            <Plus />
            New Role
          </Button>
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
    </div>
  );
}
