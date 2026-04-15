import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useParams } from "react-router-dom";

import { rolesApi } from "@/api/roles";
import { permissionsApi, type PermissionView } from "@/api/permissions";
import { useToast } from "@/hooks/use-toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

function setFromIds(ids: number[]): Set<number> {
  return new Set(ids);
}

function areSetsEqual(left: Set<number>, right: Set<number>): boolean {
  if (left.size !== right.size) {
    return false;
  }

  for (const value of left) {
    if (!right.has(value)) {
      return false;
    }
  }

  return true;
}

function toSortedIds(values: Set<number>): number[] {
  return [...values].sort((a, b) => a - b);
}

function permissionDisplayName(item: PermissionView): string {
  const permissionName = item.permission_name ?? `permission:${item.permission_id}`;
  const viewMenuName = item.view_menu_name ?? `view:${item.view_menu_id}`;
  return `${permissionName} ${viewMenuName}`;
}

export default function RolePermissionsPage() {
  const queryClient = useQueryClient();
  const { success, error } = useToast();
  const { id } = useParams();

  const roleId = Number(id);
  const hasValidRoleId = Number.isFinite(roleId) && roleId > 0;

  const [searchValue, setSearchValue] = useState("");
  const [localAssignments, setLocalAssignments] = useState<Set<number>>(new Set());

  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: rolesApi.getRoles,
  });

  const permissionViewsQuery = useQuery({
    queryKey: ["permission-views"],
    queryFn: permissionsApi.getPermissionViews,
    enabled: hasValidRoleId,
  });

  const rolePermissionsQuery = useQuery({
    queryKey: ["role-permissions", roleId],
    queryFn: () => rolesApi.getRolePermissions(roleId),
    enabled: hasValidRoleId,
  });

  const serverAssignments = rolePermissionsQuery.data ?? [];

  useEffect(() => {
    if (!rolePermissionsQuery.isSuccess) {
      return;
    }

    setLocalAssignments(setFromIds(rolePermissionsQuery.data));
  }, [rolePermissionsQuery.data, rolePermissionsQuery.isSuccess]);

  const saveMutation = useMutation({
    mutationFn: (permissionViewIds: number[]) => rolesApi.setRolePermissions(roleId, permissionViewIds),
    onSuccess: (updatedPermissionViewIds) => {
      queryClient.setQueryData(["role-permissions", roleId], updatedPermissionViewIds);
      queryClient.invalidateQueries({ queryKey: ["roles"] });
      setLocalAssignments(setFromIds(updatedPermissionViewIds));
      success("Role permissions updated");
    },
    onError: (err) => {
      error((err as Error).message || "Failed to update role permissions");
    },
  });

  const roleName = useMemo(() => {
    const roles = rolesQuery.data ?? [];
    const role = roles.find((candidate) => candidate.id === roleId);
    return role?.name ?? `Role #${roleId}`;
  }, [roleId, rolesQuery.data]);

  const normalizedSearch = searchValue.trim().toLowerCase();
  const filteredPermissionViews = useMemo(() => {
    const permissionViews = permissionViewsQuery.data ?? [];
    if (!normalizedSearch) {
      return permissionViews;
    }

    return permissionViews.filter((item) => {
      const permissionName = (item.permission_name ?? "").toLowerCase();
      const viewMenuName = (item.view_menu_name ?? "").toLowerCase();
      return permissionName.includes(normalizedSearch) || viewMenuName.includes(normalizedSearch);
    });
  }, [normalizedSearch, permissionViewsQuery.data]);

  const groupedPermissionViews = useMemo(() => {
    const groups = new Map<string, PermissionView[]>();

    filteredPermissionViews.forEach((item) => {
      const groupName = item.view_menu_name?.trim() || "Ungrouped";
      const existing = groups.get(groupName) ?? [];
      groups.set(groupName, [...existing, item]);
    });

    return [...groups.entries()];
  }, [filteredPermissionViews]);

  const serverAssignmentSet = useMemo(() => setFromIds(serverAssignments), [serverAssignments]);
  const isDirty = !areSetsEqual(localAssignments, serverAssignmentSet);
  const hasAtLeastOnePermission = localAssignments.size > 0;
  const canSave = hasValidRoleId && isDirty && hasAtLeastOnePermission && !saveMutation.isPending;

  useEffect(() => {
    if (!isDirty) {
      return;
    }

    const beforeUnloadHandler = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", beforeUnloadHandler);
    return () => {
      window.removeEventListener("beforeunload", beforeUnloadHandler);
    };
  }, [isDirty]);

  const isLoading = rolesQuery.isLoading || permissionViewsQuery.isLoading || rolePermissionsQuery.isLoading;
  const hasError = rolesQuery.isError || permissionViewsQuery.isError || rolePermissionsQuery.isError;

  function togglePermission(permissionViewId: number, checked: boolean) {
    setLocalAssignments((previous) => {
      const next = new Set(previous);
      if (checked) {
        next.add(permissionViewId);
      } else {
        next.delete(permissionViewId);
      }
      return next;
    });
  }

  function resetAssignments() {
    setLocalAssignments(setFromIds(serverAssignments));
  }

  function saveAssignments() {
    if (!canSave) {
      return;
    }
    saveMutation.mutate(toSortedIds(localAssignments));
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div className="flex flex-col gap-2">
              <CardTitle>Role Permissions</CardTitle>
              <CardDescription>
                Assign permission views for <span className="font-medium">{roleName}</span>.
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              {isDirty ? <Badge variant="secondary">Unsaved changes</Badge> : null}
              <Button variant="outline" onClick={resetAssignments} disabled={!isDirty || saveMutation.isPending}>
                Reset
              </Button>
              <Button onClick={saveAssignments} disabled={!canSave}>
                Save Changes
              </Button>
            </div>
          </div>

          <div className="relative max-w-md">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={searchValue}
              onChange={(event) => setSearchValue(event.target.value)}
              className="pl-10"
              placeholder="Search permission or view menu"
              aria-label="Search permissions"
            />
          </div>

          {!hasAtLeastOnePermission ? (
            <p className="text-sm text-destructive">At least one permission must be assigned.</p>
          ) : null}
        </CardHeader>

        <CardContent>
          {isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : null}

          {!isLoading && hasError ? (
            <p className="text-sm text-destructive">Failed to load role permissions.</p>
          ) : null}

          {!isLoading && !hasError ? (
            <ScrollArea className="max-h-[60vh] rounded-md border">
              <div className="flex flex-col">
                {groupedPermissionViews.length === 0 ? (
                  <p className="px-4 py-6 text-sm text-muted-foreground">No permission views found.</p>
                ) : null}

                {groupedPermissionViews.map(([groupName, items], groupIndex) => (
                  <div key={groupName} className="flex flex-col">
                    {groupIndex > 0 ? <Separator /> : null}
                    <div className="sticky top-0 z-10 bg-background/95 px-4 py-2 backdrop-blur">
                      <Badge variant="outline">{groupName}</Badge>
                    </div>

                    <div className="flex flex-col">
                      {items.map((item) => {
                        const checked = localAssignments.has(item.id);
                        const label = permissionDisplayName(item);
                        return (
                          <label
                            key={item.id}
                            className="flex cursor-pointer items-center justify-between gap-3 border-t px-4 py-3"
                          >
                            <span className="text-sm">{label.replace(" ", " · ")}</span>
                            <Checkbox
                              checked={checked}
                              onCheckedChange={(value) => togglePermission(item.id, value === true)}
                              aria-label={label}
                              disabled={saveMutation.isPending}
                            />
                          </label>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
