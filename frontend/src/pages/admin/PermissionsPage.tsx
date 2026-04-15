import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  permissionsApi,
  type Permission,
  type PermissionView,
  type ViewMenu,
} from "@/api/permissions";
import { useToast } from "@/hooks/use-toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";

interface PairLookup {
  [key: string]: PermissionView;
}

function pairKey(permissionId: number, viewMenuId: number): string {
  return `${permissionId}:${viewMenuId}`;
}

export default function PermissionsPage() {
  const queryClient = useQueryClient();
  const { success, error } = useToast();

  const [permissionName, setPermissionName] = useState("");
  const [viewMenuName, setViewMenuName] = useState("");
  const [searchText, setSearchText] = useState("");

  const permissionsQuery = useQuery({
    queryKey: ["permissions"],
    queryFn: permissionsApi.getPermissions,
  });

  const viewMenusQuery = useQuery({
    queryKey: ["view-menus"],
    queryFn: permissionsApi.getViewMenus,
  });

  const permissionViewsQuery = useQuery({
    queryKey: ["permission-views"],
    queryFn: permissionsApi.getPermissionViews,
  });

  const createPermissionMutation = useMutation({
    mutationFn: permissionsApi.createPermission,
    onSuccess: () => {
      success("Permission created");
      setPermissionName("");
      queryClient.invalidateQueries({ queryKey: ["permissions"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to create permission");
    },
  });

  const createViewMenuMutation = useMutation({
    mutationFn: permissionsApi.createViewMenu,
    onSuccess: () => {
      success("View menu created");
      setViewMenuName("");
      queryClient.invalidateQueries({ queryKey: ["view-menus"] });
    },
    onError: (err) => {
      error((err as Error).message || "Failed to create view menu");
    },
  });

  const createPermissionViewMutation = useMutation({
    mutationFn: permissionsApi.createPermissionView,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["permission-views"] });
      success("Permission mapping created");
    },
    onError: (err) => {
      error((err as Error).message || "Failed to create mapping");
    },
  });

  const deletePermissionViewMutation = useMutation({
    mutationFn: permissionsApi.deletePermissionView,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["permission-views"] });
      success("Permission mapping removed");
    },
    onError: (err) => {
      error((err as Error).message || "Failed to delete mapping");
    },
  });

  const permissions = permissionsQuery.data ?? [];
  const viewMenus = viewMenusQuery.data ?? [];
  const permissionViews = permissionViewsQuery.data ?? [];

  const mappingByKey = useMemo(() => {
    return permissionViews.reduce<PairLookup>((acc, item) => {
      acc[pairKey(item.permission_id, item.view_menu_id)] = item;
      return acc;
    }, {});
  }, [permissionViews]);

  const normalizedSearch = searchText.trim().toLowerCase();
  const filteredPermissions = useMemo(() => {
    if (!normalizedSearch) {
      return permissions;
    }
    return permissions.filter((item) => item.name.toLowerCase().includes(normalizedSearch));
  }, [permissions, normalizedSearch]);

  const filteredViewMenus = useMemo(() => {
    if (!normalizedSearch) {
      return viewMenus;
    }
    return viewMenus.filter((item) => item.name.toLowerCase().includes(normalizedSearch));
  }, [viewMenus, normalizedSearch]);

  const isLoading = permissionsQuery.isLoading || viewMenusQuery.isLoading || permissionViewsQuery.isLoading;

  function onCreatePermission() {
    const name = permissionName.trim();
    if (!name) {
      return;
    }
    createPermissionMutation.mutate({ name });
  }

  function onCreateViewMenu() {
    const name = viewMenuName.trim();
    if (!name) {
      return;
    }
    createViewMenuMutation.mutate({ name });
  }

  function onToggle(permissionId: number, viewMenuId: number) {
    const key = pairKey(permissionId, viewMenuId);
    const existing = mappingByKey[key];
    if (existing) {
      deletePermissionViewMutation.mutate(existing.id);
      return;
    }
    createPermissionViewMutation.mutate({ permission_id: permissionId, view_menu_id: viewMenuId });
  }

  function onSaveChanges() {
    success("Changes are saved automatically per toggle");
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Permission Management</h1>
          <p className="text-sm text-muted-foreground">Manage permissions, view menus, and permission matrix mappings.</p>
        </div>
      </header>

      <Tabs defaultValue="permissions" className="flex flex-col gap-3">
        <TabsList>
          <TabsTrigger value="permissions">Permissions</TabsTrigger>
          <TabsTrigger value="view-menus">View Menus</TabsTrigger>
          <TabsTrigger value="matrix">Permission Matrix</TabsTrigger>
        </TabsList>

        <TabsContent value="permissions">
          <div className="flex flex-col gap-3">
            <div className="flex gap-2">
              <Input
                value={permissionName}
                onChange={(event) => setPermissionName(event.target.value)}
                placeholder="New permission name"
                aria-label="Permission name"
              />
              <Button onClick={onCreatePermission} disabled={createPermissionMutation.isPending}>
                Add Permission
              </Button>
            </div>

            {permissionsQuery.isLoading ? (
              <Skeleton className="h-24 w-full" />
            ) : (
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead className="bg-muted/50 text-left">
                    <tr>
                      <th className="px-3 py-2 font-medium">Name</th>
                    </tr>
                  </thead>
                  <tbody>
                    {permissions.map((permission) => (
                      <tr key={permission.id} className="border-t">
                        <td className="px-3 py-2">{permission.name}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="view-menus">
          <div className="flex flex-col gap-3">
            <div className="flex gap-2">
              <Input
                value={viewMenuName}
                onChange={(event) => setViewMenuName(event.target.value)}
                placeholder="New view menu name"
                aria-label="View menu name"
              />
              <Button onClick={onCreateViewMenu} disabled={createViewMenuMutation.isPending}>
                Add View Menu
              </Button>
            </div>

            {viewMenusQuery.isLoading ? (
              <Skeleton className="h-24 w-full" />
            ) : (
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead className="bg-muted/50 text-left">
                    <tr>
                      <th className="px-3 py-2 font-medium">Name</th>
                    </tr>
                  </thead>
                  <tbody>
                    {viewMenus.map((viewMenu) => (
                      <tr key={viewMenu.id} className="border-t">
                        <td className="px-3 py-2">{viewMenu.name}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="matrix">
          <div className="flex flex-col gap-3">
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <Command className="p-2 md:w-96">
                <CommandInput
                  value={searchText}
                  onValueChange={setSearchText}
                  placeholder="Search permissions and view menus"
                />
                <CommandList>
                  <CommandEmpty>No results found.</CommandEmpty>
                  <CommandGroup heading="Permissions">
                    {filteredPermissions.slice(0, 5).map((permission: Permission) => (
                      <CommandItem key={`permission-${permission.id}`} onSelect={() => setSearchText(permission.name)}>
                        Permission: {permission.name}
                      </CommandItem>
                    ))}
                  </CommandGroup>
                  <CommandGroup heading="View Menus">
                    {filteredViewMenus.slice(0, 5).map((viewMenu: ViewMenu) => (
                      <CommandItem key={`view-menu-${viewMenu.id}`} onSelect={() => setSearchText(viewMenu.name)}>
                        View menu: {viewMenu.name}
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>

              <Button variant="outline" onClick={onSaveChanges}>
                Save Changes
              </Button>
            </div>

            {isLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : null}

            {!isLoading && (permissionsQuery.isError || viewMenusQuery.isError || permissionViewsQuery.isError) ? (
              <p className="text-sm text-destructive">Failed to load permission matrix</p>
            ) : null}

            {!isLoading && !permissionsQuery.isError && !viewMenusQuery.isError && !permissionViewsQuery.isError ? (
              <ScrollArea className="max-h-[55vh] rounded-md border">
                <table className="w-full min-w-[640px] text-sm">
                  <thead className="sticky top-0 bg-muted/95 text-left">
                    <tr>
                      <th className="px-3 py-2 font-medium">Permission</th>
                      {filteredViewMenus.map((viewMenu) => (
                        <th key={viewMenu.id} className="px-3 py-2 font-medium">
                          {viewMenu.name}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {filteredPermissions.map((permission) => (
                      <tr key={permission.id} className="border-t">
                        <td className="px-3 py-2">
                          <Badge variant="secondary">{permission.name}</Badge>
                        </td>
                        {filteredViewMenus.map((viewMenu) => {
                          const key = pairKey(permission.id, viewMenu.id);
                          const checked = !!mappingByKey[key];
                          return (
                            <td key={`${permission.id}-${viewMenu.id}`} className="px-3 py-2">
                              <Checkbox
                                checked={checked}
                                onCheckedChange={() => onToggle(permission.id, viewMenu.id)}
                                aria-label={`${permission.name} - ${viewMenu.name}`}
                                disabled={createPermissionViewMutation.isPending || deletePermissionViewMutation.isPending}
                              />
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </ScrollArea>
            ) : null}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
