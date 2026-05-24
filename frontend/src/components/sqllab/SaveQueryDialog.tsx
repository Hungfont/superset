import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { useToast } from "@/hooks/use-toast";
import { saveQuery, type SaveQueryRequest } from "@/api/sqllab";

interface SaveQueryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  defaultLabel: string;
  dbId: number | null;
  schema: string;
  catalog?: string;
  sql: string;
}

export function SaveQueryDialog({
  open,
  onOpenChange,
  defaultLabel,
  dbId,
  schema,
  catalog,
  sql,
}: SaveQueryDialogProps) {
  const [label, setLabel] = useState(defaultLabel);
  const [description, setDescription] = useState("");
  const [published, setPublished] = useState(false);
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const handleOpenChange = (next: boolean) => {
    if (next) {
      setLabel(defaultLabel);
      setDescription("");
      setPublished(false);
    }
    onOpenChange(next);
  };

  const mutation = useMutation({
    mutationFn: (data: SaveQueryRequest) => saveQuery(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["saved-queries"] });
      toast("Query saved");
      onOpenChange(false);
    },
    onError: (error: Error) => {
      toast("Failed to save query: " + error.message);
    },
  });

  const handleSave = () => {
    if (!dbId || !label.trim() || !sql.trim()) return;
    mutation.mutate({
      db_id: dbId,
      label: label.trim(),
      schema,
      catalog,
      sql,
      description: description.trim() || undefined,
      published: published || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Save Query</DialogTitle>
          <DialogDescription>
            Save this query for later. It will appear in your Saved Queries library.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label htmlFor="sq-label">Name</Label>
            <Input
              id="sq-label"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Query name"
              onKeyDown={(e) => e.key === "Enter" && handleSave()}
            />
          </div>
          <div>
            <Label htmlFor="sq-desc">Description</Label>
            <Textarea
              id="sq-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does this query do? (optional)"
              rows={3}
            />
          </div>
          <div className="flex items-center justify-between">
            <div>
              <Label htmlFor="sq-published">Published</Label>
              <p className="text-xs text-muted-foreground">
                Visible to all team members in your organization
              </p>
            </div>
            <Switch
              id="sq-published"
              checked={published}
              onCheckedChange={setPublished}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={!label.trim() || !sql.trim() || mutation.isPending}
          >
            {mutation.isPending ? "Saving..." : "Save Query"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
