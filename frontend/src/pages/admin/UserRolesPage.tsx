import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, ChevronsUpDown, X } from "lucide-react";
import { useParams } from "react-router-dom";

import { rolesApi } from "@/api/roles";
import { userRolesApi } from "@/api/userRoles";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";

function normalizeIds(values: number[]): number[] {
  return [...new Set(values)].sort((a, b) => a - b);
}

function arraysEqual(left: number[], right: number[]): boolean {
  if (left.length !== right.length) {
    return false;
  }

  return left.every((value, index) => value === right[index]);
}

export default function UserRolesPage() {
  const queryClient = useQueryClient();
  const { success, error } = useToast();
  const { id } = useParams();

  const userId = Number(id);
  const hasValidUserID = Number.isFinite(userId) && userId > 0;

  const [open, setOpen] = useState(false);
  const [selectedRoleIds, setSelectedRoleIds] = useState<number[]>([]);

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: rolesApi.getRoles,
    enabled: hasValidUserID,
  });

  const userRolesQuery = useQuery({
    queryKey: ["user-roles", userId],
    queryFn: () => userRolesApi.getUserRoles(userId),
    enabled: hasValidUserID,
  });

  useEffect(() => {
    if (!userRolesQuery.isSuccess) {
      return;
    }
    setSelectedRoleIds(normalizeIds(userRolesQuery.data));
  }, [userRolesQuery.data, userRolesQuery.isSuccess]);

  const saveMutation = useMutation({
    mutationFn: (roleIDs: number[]) => userRolesApi.setUserRoles(userId, roleIDs),
    onSuccess: (updatedRoleIDs) => {
      queryClient.setQueryData(["user-roles", userId], updatedRoleIDs);
      setSelectedRoleIds(normalizeIds(updatedRoleIDs));
      success("User roles updated");
    },
    onError: (err) => {
      error((err as Error).message || "Failed to update user roles");
    },
  });

  const roles = rolesQuery.data ?? [];
  const selectedRoleSet = useMemo(() => new Set(selectedRoleIds), [selectedRoleIds]);
  const roleIDsFromServer = normalizeIds(userRolesQuery.data ?? []);
  const isDirty = !arraysEqual(normalizeIds(selectedRoleIds), roleIDsFromServer);

  const canSave = hasValidUserID && selectedRoleIds.length > 0 && isDirty && !saveMutation.isPending;

  const selectedRoles = useMemo(
    () => roles.filter((role) => selectedRoleSet.has(role.id)),
    [roles, selectedRoleSet],
  );

  const selectionLabel = selectedRoles.length
    ? `${selectedRoles.length} role${selectedRoles.length > 1 ? "s" : ""} selected`
    : "Select roles";

  const isLoading = rolesQuery.isLoading || userRolesQuery.isLoading;
  const hasError = rolesQuery.isError || userRolesQuery.isError;

  function toggleRole(roleID: number) {
    setSelectedRoleIds((previous) => {
      const set = new Set(previous);
      if (set.has(roleID)) {
        set.delete(roleID);
      } else {
        set.add(roleID);
      }
      return normalizeIds([...set]);
    });
  }

  function removeRole(roleID: number) {
    setSelectedRoleIds((previous) => previous.filter((value) => value !== roleID));
  }

  function saveRoles() {
    if (!canSave) {
      return;
    }
    saveMutation.mutate(normalizeIds(selectedRoleIds));
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>User Roles</CardTitle>
          <CardDescription>
            Replace all assigned roles for user <span className="font-medium">#{userId}</span>.
          </CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-4">
          {isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : null}

          {!isLoading && hasError ? (
            <p className="text-sm text-destructive">Failed to load role assignments.</p>
          ) : null}

          {!isLoading && !hasError ? (
            <>
              <div className="flex flex-col gap-2">
                <Popover open={open} onOpenChange={setOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="outline"
                      role="combobox"
                      aria-expanded={open}
                      className="w-full justify-between"
                    >
                      {selectionLabel}
                      <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>

                  <PopoverContent className="w-[360px] p-0" align="start">
                    <Command>
                      <CommandInput placeholder="Search roles..." aria-label="Search roles" />
                      <CommandList>
                        <CommandEmpty>No role found.</CommandEmpty>
                        <CommandGroup heading="Available roles">
                          {roles.map((role) => {
                            const selected = selectedRoleSet.has(role.id);
                            return (
                              <CommandItem
                                key={role.id}
                                value={role.name}
                                onSelect={() => toggleRole(role.id)}
                                aria-label={`Toggle role ${role.name}`}
                              >
                                <Check className={cn("size-4", selected ? "opacity-100" : "opacity-0")} />
                                <span>{role.name}</span>
                              </CommandItem>
                            );
                          })}
                        </CommandGroup>
                      </CommandList>
                    </Command>
                  </PopoverContent>
                </Popover>

                <div className="flex flex-wrap gap-2">
                  {selectedRoles.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No roles selected.</p>
                  ) : null}

                  {selectedRoles.map((role) => (
                    <Badge key={role.id} variant="secondary" className="flex items-center gap-1">
                      <span>{role.name}</span>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-4"
                        aria-label={`Remove role ${role.name}`}
                        onClick={() => removeRole(role.id)}
                      >
                        <X className="size-3" />
                      </Button>
                    </Badge>
                  ))}
                </div>
              </div>

              {selectedRoleIds.length === 0 ? (
                <Alert variant="destructive" role="alert">
                  <AlertTitle>Validation error</AlertTitle>
                  <AlertDescription>User must have at least one role</AlertDescription>
                </Alert>
              ) : null}

              <div className="flex items-center gap-2">
                <Button onClick={saveRoles} disabled={!canSave}>
                  Update Roles
                </Button>
                {isDirty ? <Badge variant="outline">Unsaved changes</Badge> : null}
              </div>
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
