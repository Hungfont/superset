import { useState, useCallback, useMemo } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Pencil, Save, RotateCcw, Calculator, AlertCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { datasetsApi, type Column, type UpdateColumnPayload, type BulkUpdateColumnPayload } from "@/api/datasets";
import { useToast } from "@/hooks/use-toast";

interface ColumnsTabProps {
  datasetId: number;
  columns: Column[];
}

type LocalEdit = {
  original: Column;
  changes: UpdateColumnPayload;
};

export function ColumnsTab({ datasetId, columns: initialColumns }: ColumnsTabProps) {
  const { success, error } = useToast();
  const queryClient = useQueryClient();
  const [localEdits, setLocalEdits] = useState<Map<number, LocalEdit>>(new Map());
  const [editingCell, setEditingCell] = useState<{ colId: number; field: string } | null>(null);
  const [detailColumn, setDetailColumn] = useState<Column | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const columns = useMemo(() => initialColumns || [], [initialColumns]);
  const isDirty = localEdits.size > 0;

  const updateColumnMutation = useMutation({
    mutationFn: ({ columnId, payload }: { columnId: number; payload: UpdateColumnPayload }) =>
      datasetsApi.updateColumn(datasetId, columnId, payload),
    onSuccess: () => {
      success("Column updated successfully");
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
      queryClient.invalidateQueries({ queryKey: ["dataset-columns", datasetId] });
    },
    onError: (err: Error & { status?: number; message?: string }) => {
      if (err.status === 403) {
        error("You don't have permission to update columns");
        return;
      }
      if (err.status === 422) {
        error(err.message || "Invalid request");
        return;
      }
      error(err.message || "Failed to update column");
    },
  });

  const bulkUpdateMutation = useMutation({
    mutationFn: (payload: BulkUpdateColumnPayload) => datasetsApi.bulkUpdateColumns(datasetId, payload),
    onSuccess: () => {
      success("Columns updated successfully");
      setLocalEdits(new Map());
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
      queryClient.invalidateQueries({ queryKey: ["dataset-columns", datasetId] });
    },
    onError: (err: Error & { status?: number; message?: string }) => {
      if (err.status === 403) {
        error("You don't have permission to update columns");
        return;
      }
      if (err.status === 422) {
        error(err.message || "Invalid request - all changes rolled back");
        return;
      }
      error(err.message || "Failed to update columns - all changes rolled back");
    },
  });

  const handleFieldChange = useCallback((colId: number, field: keyof UpdateColumnPayload, value: unknown) => {
    setLocalEdits((prev) => {
      const existing = prev.get(colId);
      const original = existing?.original || columns.find((c) => c.id === colId);
      if (!original) return prev;

      const newChanges = { ...existing?.changes } as UpdateColumnPayload;
      (newChanges as Record<string, unknown>)[field] = value;

      const newEdits = new Map(prev);
      newEdits.set(colId, { original, changes: newChanges });
      return newEdits;
    });
  }, [columns]);

  const handleSingleSave = useCallback((colId: number) => {
    const edit = localEdits.get(colId);
    if (!edit) return;

    updateColumnMutation.mutate({
      columnId: colId,
      payload: edit.changes,
    });
    setLocalEdits((prev) => {
      const newEdits = new Map(prev);
      newEdits.delete(colId);
      return newEdits;
    });
  }, [localEdits, updateColumnMutation]);

  const handleBulkSave = useCallback(() => {
    const payload: BulkUpdateColumnPayload = {
      columns: Array.from(localEdits.values()).map((edit) => ({
        id: edit.original.id,
        ...edit.changes,
      })),
    };
    bulkUpdateMutation.mutate(payload);
  }, [localEdits, bulkUpdateMutation]);

  const handleReset = useCallback(() => {
    setLocalEdits(new Map());
  }, []);

  const openDetailSheet = useCallback((col: Column) => {
    setDetailColumn(col);
    setDetailOpen(true);
  }, []);

  const getEditedValue = useCallback((colId: number, field: keyof Column): unknown => {
    const edit = localEdits.get(colId);
    if (edit && field in edit.changes) {
      return edit.changes[field as keyof UpdateColumnPayload];
    }
    const original = columns.find((c) => c.id === colId);
    return original?.[field];
  }, [localEdits, columns]);

  const isEdited = useCallback((colId: number) => localEdits.has(colId), [localEdits]);

  return (
    <TooltipProvider>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="text-lg font-semibold">Columns</h3>
            {isDirty && (
              <Badge variant="secondary" className="bg-amber-100 text-amber-800">
                {localEdits.size} unsaved change{localEdits.size !== 1 ? "s" : ""}
              </Badge>
            )}
          </div>
          <div className="flex gap-2">
            {isDirty && (
              <>
                <Button variant="outline" size="sm" onClick={handleReset} disabled={bulkUpdateMutation.isPending}>
                  <RotateCcw className="mr-2 h-4 w-4" />
                  Reset
                </Button>
                <Button size="sm" onClick={handleBulkSave} disabled={bulkUpdateMutation.isPending}>
                  {bulkUpdateMutation.isPending ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="mr-2 h-4 w-4" />
                  )}
                  Save All Changes
                </Button>
              </>
            )}
          </div>
        </div>

        <div className="border rounded-md">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-3 py-2 text-left text-sm font-medium">Column Name</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Verbose Name</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Type</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-20">DateTime</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-20">Filterable</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-20">Group By</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Expression</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {columns.map((col) => (
                <tr
                  key={col.id}
                  className={`border-b hover:bg-muted/30 transition-colors ${
                    isEdited(col.id) ? "border-l-4 border-l-amber-500" : ""
                  } ${!col.is_active ? "opacity-50" : ""}`}
                >
                  <td className="px-3 py-2 text-sm font-mono">{col.column_name}</td>
                  <td className="px-3 py-2">
                    <Popover
                      open={editingCell?.colId === col.id && editingCell?.field === "verbose_name"}
                      onOpenChange={(open) => !open && setEditingCell(null)}
                    >
                      <PopoverTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-full justify-start font-normal"
                          onClick={() => setEditingCell({ colId: col.id, field: "verbose_name" })}
                        >
                          <Pencil className="mr-2 h-3 w-3" />
                          {(getEditedValue(col.id, "verbose_name") as string) || (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-80" align="start">
                        <div className="space-y-2">
                          <label className="text-sm font-medium">Verbose Name</label>
                          <Input
                            value={(getEditedValue(col.id, "verbose_name") as string) || ""}
                            onChange={(e) => handleFieldChange(col.id, "verbose_name", e.target.value)}
                            onBlur={() => setEditingCell(null)}
                            placeholder="Enter verbose name"
                            autoFocus
                          />
                        </div>
                      </PopoverContent>
                    </Popover>
                  </td>
                  <td className="px-3 py-2 text-sm">
                    {(getEditedValue(col.id, "column_type") as string) || col.type}
                  </td>
                  <td className="px-3 py-2 text-center">
                    <Switch
                      checked={(getEditedValue(col.id, "is_dttm") as boolean) ?? col.is_dttm}
                      onCheckedChange={(checked) => handleFieldChange(col.id, "is_dttm", checked)}
                    />
                  </td>
                  <td className="px-3 py-2 text-center">
                    <Switch
                      checked={(getEditedValue(col.id, "filterable") as boolean) ?? col.filterable ?? true}
                      onCheckedChange={(checked) => handleFieldChange(col.id, "filterable", checked)}
                    />
                  </td>
                  <td className="px-3 py-2 text-center">
                    <Switch
                      checked={(getEditedValue(col.id, "groupby") as boolean) ?? col.groupby ?? true}
                      onCheckedChange={(checked) => handleFieldChange(col.id, "groupby", checked)}
                    />
                  </td>
                  <td className="px-3 py-2">
                    <Popover
                      open={editingCell?.colId === col.id && editingCell?.field === "expression"}
                      onOpenChange={(open) => !open && setEditingCell(null)}
                    >
                      <PopoverTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-full justify-start font-mono text-xs"
                          onClick={() => setEditingCell({ colId: col.id, field: "expression" })}
                        >
                          {col.expression ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="truncate flex items-center gap-1">
                                  <Calculator className="h-3 w-3" />
                                  {col.expression.substring(0, 20)}...
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>This is a calculated column</p>
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </Button>
                      </PopoverTrigger>
                      <PopoverContent className="w-96" align="start">
                        <div className="space-y-2">
                          <label className="text-sm font-medium">Expression</label>
                          <Textarea
                            value={(getEditedValue(col.id, "expression") as string) || ""}
                            onChange={(e) => handleFieldChange(col.id, "expression", e.target.value)}
                            placeholder="Enter SQL expression"
                            className="font-mono text-sm min-h-[100px]"
                            autoFocus
                          />
                          <p className="text-xs text-muted-foreground">
                            Enter a valid SQL expression. E.g., SUM(column_name)
                          </p>
                        </div>
                      </PopoverContent>
                    </Popover>
                  </td>
                  <td className="px-3 py-2 text-center">
                    <div className="flex items-center justify-center gap-1">
                      {isEdited(col.id) && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => handleSingleSave(col.id)}
                          disabled={updateColumnMutation.isPending}
                        >
                          <Save className="h-4 w-4" />
                        </Button>
                      )}
                      <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => openDetailSheet(col)}>
                        <Pencil className="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {columns.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <AlertCircle className="h-8 w-8 mb-2" />
            <p>No columns found for this dataset</p>
          </div>
        )}

        <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
          <SheetContent className="sm:max-w-[540px]">
            <SheetHeader>
              <SheetTitle>Edit Column: {detailColumn?.column_name}</SheetTitle>
              <SheetDescription>
                Make changes to the column metadata. Click save when you're done.
              </SheetDescription>
            </SheetHeader>
            {detailColumn && (
              <ColumnDetailForm
                column={detailColumn}
                datasetId={datasetId}
                localEdit={localEdits.get(detailColumn.id)}
                onFieldChange={(field, value) => handleFieldChange(detailColumn.id, field, value)}
                onSave={() => {
                  handleSingleSave(detailColumn.id);
                  setDetailOpen(false);
                }}
                isSaving={updateColumnMutation.isPending}
              />
            )}
          </SheetContent>
        </Sheet>
      </div>
    </TooltipProvider>
  );
}

interface ColumnDetailFormProps {
  column: Column;
  datasetId: number;
  localEdit?: LocalEdit;
  onFieldChange: (field: keyof UpdateColumnPayload, value: unknown) => void;
  onSave: () => void;
  isSaving: boolean;
}

function ColumnDetailForm({
  column,
  localEdit,
  onFieldChange,
  onSave,
  isSaving,
}: ColumnDetailFormProps) {
  const getValue = useCallback(
    (field: keyof Column): unknown => {
      if (localEdit && field in localEdit.changes) {
        return localEdit.changes[field as keyof UpdateColumnPayload];
      }
      return column[field];
    },
    [column, localEdit]
  );

  return (
    <div className="space-y-4 py-4">
      <div className="grid gap-2">
        <label className="text-sm font-medium">Verbose Name</label>
        <Input
          value={(getValue("verbose_name") as string) || ""}
          onChange={(e) => onFieldChange("verbose_name", e.target.value)}
          placeholder="Enter verbose name"
        />
      </div>

      <div className="grid gap-2">
        <label className="text-sm font-medium">Description</label>
        <Textarea
          value={(getValue("description") as string) || ""}
          onChange={(e) => onFieldChange("description", e.target.value)}
          placeholder="Enter description"
        />
      </div>

      <div className="grid gap-2">
        <label className="text-sm font-medium">Type Override</label>
        <Input
          value={(getValue("column_type") as string) || ""}
          onChange={(e) => onFieldChange("column_type", e.target.value)}
          placeholder="Enter type (e.g., VARCHAR, INT)"
        />
      </div>

      <div className="grid gap-2">
        <label className="text-sm font-medium">Python Date Format</label>
        <Input
          value={(getValue("python_date_format") as string) || ""}
          onChange={(e) => onFieldChange("python_date_format", e.target.value)}
          placeholder="e.g., %Y-%m-%d"
        />
        <p className="text-xs text-muted-foreground">Valid strftime format patterns</p>
      </div>

      <div className="grid gap-2">
        <label className="text-sm font-medium">Expression</label>
        <Textarea
          value={(getValue("expression") as string) || ""}
          onChange={(e) => onFieldChange("expression", e.target.value)}
          placeholder="Enter SQL expression for calculated columns"
          className="font-mono text-sm min-h-[100px]"
        />
      </div>

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div className="space-y-0.5">
          <label className="text-sm font-medium">Is DateTime</label>
          <p className="text-xs text-muted-foreground">Mark as datetime column</p>
        </div>
        <Switch
          checked={(getValue("is_dttm") as boolean) ?? column.is_dttm}
          onCheckedChange={(checked) => onFieldChange("is_dttm", checked)}
        />
      </div>

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div className="space-y-0.5">
          <label className="text-sm font-medium">Filterable</label>
          <p className="text-xs text-muted-foreground">Allow filtering on this column</p>
        </div>
        <Switch
          checked={(getValue("filterable") as boolean) ?? column.filterable ?? true}
          onCheckedChange={(checked) => onFieldChange("filterable", checked)}
        />
      </div>

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div className="space-y-0.5">
          <label className="text-sm font-medium">Group By</label>
          <p className="text-xs text-muted-foreground">Allow grouping by this column</p>
        </div>
        <Switch
          checked={(getValue("groupby") as boolean) ?? column.groupby ?? true}
          onCheckedChange={(checked) => onFieldChange("groupby", checked)}
        />
      </div>

      <div className="flex items-center justify-between rounded-lg border p-4">
        <div className="space-y-0.5">
          <label className="text-sm font-medium">Exported</label>
          <p className="text-xs text-muted-foreground">Export this column in query results</p>
        </div>
        <Switch
          checked={(getValue("exported") as boolean) ?? column.exported ?? true}
          onCheckedChange={(checked) => onFieldChange("exported", checked)}
        />
      </div>

      <div className="flex justify-end gap-2 pt-4">
        <Button onClick={onSave} disabled={isSaving}>
          {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Save Changes
        </Button>
      </div>
    </div>
  );
}
