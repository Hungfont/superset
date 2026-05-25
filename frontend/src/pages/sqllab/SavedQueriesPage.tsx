import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import Editor from "@monaco-editor/react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { DataTable } from "@/components/ui/data-table";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useSqlLabStore } from "@/stores/sqlLabStore";
import {
  fetchSavedQueries,
  updateSavedQuery,
  deleteSavedQuery,
  forkSavedQuery,
  type SavedQueryResponse,
} from "@/api/sqllab";
import type { ColumnDef } from "@tanstack/react-table";

export default function SavedQueriesPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const addTab = useSqlLabStore((s) => s.addTab);
  const [search, setSearch] = useState("");

  const [editingQuery, setEditingQuery] = useState<SavedQueryResponse | null>(null);
  const [editLabel, setEditLabel] = useState("");
  const [editDescription, setEditDescription] = useState("");
  const [editSQL, setEditSQL] = useState("");
  const [editPublished, setEditPublished] = useState(false);

  const [deletingQuery, setDeletingQuery] = useState<SavedQueryResponse | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["saved-queries", { q: search }],
    queryFn: () => fetchSavedQueries({ q: search || undefined, limit: 50 }),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...rest }: { id: number; label?: string; description?: string; sql?: string; published?: boolean }) =>
      updateSavedQuery(id, rest),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      toast("Saved query updated");
      setEditingQuery(null);
    },
    onError: (error: Error) => {
      toast("Failed to update: " + error.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteSavedQuery(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      toast("Saved query deleted");
      setDeletingQuery(null);
    },
    onError: (error: Error) => {
      toast("Failed to delete: " + error.message);
    },
  });

  const forkMutation = useMutation({
    mutationFn: (id: number) => forkSavedQuery(id),
    onSuccess: (forked) => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      addTab();
      const tabs = useSqlLabStore.getState().tabs;
      const newTabId = tabs[tabs.length - 1]?.id;
      if (newTabId) {
        useSqlLabStore.getState().updateTabSql(newTabId, forked.sql);
        useSqlLabStore.getState().updateTabLabel(newTabId, forked.label);
      }
      toast("Forked to new tab");
      navigate("/sqllab");
    },
    onError: (error: Error) => {
      toast("Failed to fork: " + error.message);
    },
  });

  const handleOpenEdit = (sq: SavedQueryResponse) => {
    setEditingQuery(sq);
    setEditLabel(sq.label);
    setEditDescription(sq.description || "");
    setEditSQL(sq.sql);
    setEditPublished(sq.published);
  };

  const handleSaveEdit = () => {
    if (!editingQuery) return;
    updateMutation.mutate({
      id: editingQuery.id,
      label: editLabel.trim() || undefined,
      description: editDescription.trim() || undefined,
      sql: editSQL || undefined,
      published: editPublished,
    });
  };

  const columns = useMemo<ColumnDef<SavedQueryResponse>[]>(() => [
    {
      accessorKey: "label",
      header: "Name",
      cell: ({ getValue }) => (
        <span className="font-medium">{getValue() as string}</span>
      ),
    },
    {
      accessorKey: "db_id",
      header: "Database",
      cell: ({ getValue }) => (
        <span className="text-muted-foreground text-sm">DB #{getValue() as number}</span>
      ),
    },
    {
      accessorKey: "schema",
      header: "Schema",
    },
    {
      accessorKey: "changed_on",
      header: "Modified",
      cell: ({ getValue }) => new Date(getValue() as string).toLocaleDateString(),
    },
    {
      accessorKey: "published",
      header: "Status",
      cell: ({ getValue }) =>
        getValue() ? (
          <Badge variant="secondary">Published</Badge>
        ) : (
          <Badge variant="outline">Private</Badge>
        ),
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const sq = row.original;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => navigate(`/sqllab?load=${sq.id}`)}>
                Load in SQL Lab
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleOpenEdit(sq)}>
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => forkMutation.mutate(sq.id)}>
                Fork
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setDeletingQuery(sq)}>
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ], [navigate, forkMutation]);

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Saved Queries</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Browse and load your saved queries
          </p>
        </div>
        <Button onClick={() => navigate("/sqllab")}>
          SQL Lab
        </Button>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search saved queries..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : (
        <DataTable data={data?.items ?? []} columns={columns} onRowClick={handleOpenEdit} />
      )}

      <Sheet open={editingQuery !== null} onOpenChange={(open) => { if (!open) setEditingQuery(null); }}>
        <SheetContent side="right" className="w-[500px] sm:max-w-[500px] flex flex-col">
          <SheetHeader>
            <SheetTitle>Edit Saved Query</SheetTitle>
            <SheetDescription>
              Update the query details. Changes are saved immediately.
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-4 py-4 overflow-y-auto">
            <div>
              <Label htmlFor="edit-label">Name</Label>
              <Input
                id="edit-label"
                value={editLabel}
                onChange={(e) => setEditLabel(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="edit-desc">Description</Label>
              <Textarea
                id="edit-desc"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                placeholder="What does this query do?"
                rows={3}
              />
            </div>
            <div>
              <Label htmlFor="edit-sql">SQL</Label>
              <div className="border rounded-md overflow-hidden h-[200px]">
                <Editor
                  language="sql"
                  theme="vs-dark"
                  value={editSQL}
                  onChange={(v) => setEditSQL(v || "")}
                  options={{
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    lineNumbers: "on",
                    fontSize: 13,
                  }}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Label htmlFor="edit-published">Published</Label>
                <p className="text-xs text-muted-foreground">
                  Visible to all team members in your organization
                </p>
              </div>
              <Switch
                id="edit-published"
                checked={editPublished}
                onCheckedChange={setEditPublished}
              />
            </div>
          </div>
          <SheetFooter>
            <Button variant="outline" onClick={() => setEditingQuery(null)}>
              Cancel
            </Button>
            <Button onClick={handleSaveEdit} disabled={updateMutation.isPending || !editLabel.trim()}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <AlertDialog open={deletingQuery !== null} onOpenChange={(open) => { if (!open) setDeletingQuery(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {deletingQuery?.label}?</AlertDialogTitle>
            <AlertDialogDescription>
              This cannot be undone. Any tabs referencing this saved query will be updated.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => { if (deletingQuery) deleteMutation.mutate(deletingQuery.id); }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
