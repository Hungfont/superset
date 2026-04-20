import { useState, useCallback, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Pencil, Plus, Trash2, ShieldCheck, AlertCircle } from "lucide-react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { datasetsApi, type SqlMetric, type CreateMetricPayload, type UpdateMetricPayload } from "@/api/datasets";
import { useToast } from "@/hooks/use-toast";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

interface MetricsTabProps {
  datasetId: number;
  initialMetrics?: SqlMetric[];
}

const metricSchema = z.object({
  metric_name: z.string().min(3, "Metric name must be at least 3 characters").regex(/^[a-z][a-z0-9_]*$/, "Must be snake_case (e.g., total_count)"),
  verbose_name: z.string().optional(),
  metric_type: z.string().min(1, "Metric type is required"),
  expression: z.string().min(1, "Expression is required"),
  d3_format: z.string().optional(),
  warning_text: z.string().optional(),
  is_restricted: z.boolean().optional(),
  certified_by: z.string().optional(),
  certification_details: z.string().optional(),
  is_certified: z.boolean().optional(),
});

const D3_FORMAT_OPTIONS = [
  { value: "~g", label: "Default (no formatting)" },
  { value: ",.0f", label: "1,234" },
  { value: ",.1f", label: "1,234.5" },
  { value: ",.2f", label: "1,234.56" },
  { value: ",.3f", label: "1,234.567" },
  { value: ".0%", label: "12%" },
  { value: ".1%", label: "12.3%" },
  { value: ".2%", label: "12.34%" },
  { value: "SMART_NUMBER", label: "Smart Number (1.2K, 1.5M)" },
  { value: "$,.0f", label: "$1,234" },
  { value: "$,.2f", label: "$1,234.56" },
  { value: "€,.0f", label: "€1,234" },
  { value: "£,.0f", label: "£1,234" },
  { value: "¥,.0f", label: "¥1,234" },
];

type MetricFormData = z.infer<typeof metricSchema>;

const METRIC_TYPES = [
  { value: "SUM", label: "SUM", suggestion: "SUM(column_name)" },
  { value: "COUNT", label: "COUNT", suggestion: "COUNT(*)" },
  { value: "AVG", label: "AVG", suggestion: "AVG(column_name)" },
  { value: "MAX", label: "MAX", suggestion: "MAX(column_name)" },
  { value: "MIN", label: "MIN", suggestion: "MIN(column_name)" },
  { value: "COUNT_DISTINCT", label: "COUNT DISTINCT", suggestion: "COUNT(DISTINCT column_name)" },
  { value: "STDDEV", label: "Standard Deviation", suggestion: "STDDEV(column_name)" },
  { value: "VARIANCE", label: "Variance", suggestion: "VARIANCE(column_name)" },
  { value: "CUSTOM", label: "Custom SQL", suggestion: "" },
];

export function MetricsTab({ datasetId, initialMetrics }: MetricsTabProps) {
  const { success, error, warning } = useToast();
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingMetric, setEditingMetric] = useState<SqlMetric | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<SqlMetric | null>(null);

  const metricsQuery = useQuery({
    queryKey: ["dataset-metrics", datasetId],
    queryFn: () => datasetsApi.getMetrics(datasetId),
    initialData: initialMetrics,
  });

  const metrics = useMemo(() => metricsQuery.data || [], [metricsQuery.data]);

  const form = useForm<MetricFormData>({
    resolver: zodResolver(metricSchema),
    defaultValues: {
      metric_name: "",
      verbose_name: "",
      metric_type: "SUM",
      expression: "",
      d3_format: "",
      warning_text: "",
      is_restricted: false,
      is_certified: false,
      certified_by: "",
      certification_details: "",
    },
  });

  const createMutation = useMutation({
    mutationFn: (payload: CreateMetricPayload) => datasetsApi.createMetric(datasetId, payload),
    onSuccess: () => {
      success("Metric created successfully");
      queryClient.invalidateQueries({ queryKey: ["dataset-metrics", datasetId] });
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
      setDialogOpen(false);
      form.reset();
    },
    onError: (err: Error & { status?: number; message?: string }) => {
      if (err.status === 403) {
        error("You don't have permission to create metrics");
        return;
      }
      if (err.status === 409) {
        error("A metric with this name already exists");
        return;
      }
      if (err.status === 422) {
        error(err.message || "Expression must contain an aggregate function");
        return;
      }
      error(err.message || "Failed to create metric");
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ metricId, payload }: { metricId: number; payload: UpdateMetricPayload }) =>
      datasetsApi.updateMetric(datasetId, metricId, payload),
    onSuccess: () => {
      success("Metric updated successfully");
      queryClient.invalidateQueries({ queryKey: ["dataset-metrics", datasetId] });
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
      setDialogOpen(false);
      setEditingMetric(null);
      form.reset();
    },
    onError: (err: Error & { status?: number; message?: string }) => {
      if (err.status === 403) {
        error("You don't have permission to update metrics");
        return;
      }
      if (err.status === 409) {
        error("A metric with this name already exists");
        return;
      }
      if (err.status === 422) {
        error(err.message || "Expression must contain an aggregate function");
        return;
      }
      error(err.message || "Failed to update metric");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (metricId: number) => datasetsApi.deleteMetric(datasetId, metricId),
    onSuccess: (data) => {
      if (data.warnings && data.warnings.length > 0) {
        warning(`Metric deleted. ${data.warnings.length} charts may be affected.`);
      } else {
        success("Metric deleted successfully");
      }
      queryClient.invalidateQueries({ queryKey: ["dataset-metrics", datasetId] });
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
      setDeleteTarget(null);
    },
    onError: (err: Error & { status?: number; message?: string }) => {
      if (err.status === 403) {
        error("You don't have permission to delete metrics");
        return;
      }
      error(err.message || "Failed to delete metric");
    },
  });

  const handleOpenCreate = useCallback(() => {
    setEditingMetric(null);
    form.reset({
      metric_name: "",
      verbose_name: "",
      metric_type: "SUM",
      expression: "",
      d3_format: "",
      warning_text: "",
      is_restricted: false,
      is_certified: false,
      certified_by: "",
      certification_details: "",
    });
    setDialogOpen(true);
  }, [form]);

  const handleOpenEdit = useCallback((metric: SqlMetric) => {
    setEditingMetric(metric);
    form.reset({
      metric_name: metric.metric_name,
      verbose_name: metric.verbose_name || "",
      metric_type: metric.metric_type,
      expression: metric.expression,
      d3_format: metric.d3_format || "",
      warning_text: metric.warning_text || "",
      is_restricted: metric.is_restricted || false,
      is_certified: !!metric.certified_by,
      certified_by: metric.certified_by || "",
      certification_details: metric.certification_details || "",
    });
    setDialogOpen(true);
  }, [form]);

  const handleMetricTypeChange = useCallback((value: string) => {
    form.setValue("metric_type", value);
    const metricType = METRIC_TYPES.find((t) => t.value === value);
    if (metricType?.suggestion) {
      const currentExpression = form.getValues("expression");
      if (!currentExpression) {
        form.setValue("expression", metricType.suggestion);
      }
    }
  }, [form]);

  const onSubmit = useCallback((data: MetricFormData) => {
    if (editingMetric) {
      updateMutation.mutate({
        metricId: editingMetric.id,
        payload: data,
      });
    } else {
      createMutation.mutate(data);
    }
  }, [editingMetric, createMutation, updateMutation]);

  const handleDelete = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation]);

  const getAggregateWarning = useCallback((expression: string): boolean => {
    const lower = expression.toLowerCase();
    const aggregateKeywords = ["sum(", "count(", "avg(", "max(", "min(", "stddev(", "variance("];
    return !aggregateKeywords.some((keyword) => lower.includes(keyword));
  }, []);

  const formatPreview = useCallback((format: string | undefined, value: number = 1234567.89): string => {
    if (!format || format === "~g") return "1,234,567.89";
    try {
      const testValue = value;
      let result = format;
      
      if (format.includes(",")) {
        const decimals = format.match(/\.([0-9])?f$/)?.[1] || "0";
        const precision = parseInt(decimals, 10);
        const parts = testValue.toFixed(precision).split(".");
        parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
        result = parts.join(".");
      } else if (format.includes(".")) {
        const decimals = format.match(/\.([0-9])?f$/)?.[1] || "0";
        result = testValue.toFixed(parseInt(decimals, 10)).replace(".", " · ");
      } else if (format === ".1%" || format === ".0%") {
        result = (testValue * 100).toFixed(format === ".1%" ? 1 : 0) + "%";
      } else if (format === "SMART_NUMBER") {
        if (testValue >= 1000000) {
          result = (testValue / 1000000).toFixed(1) + "M";
        } else if (testValue >= 1000) {
          result = (testValue / 1000).toFixed(1) + "K";
        } else {
          result = testValue.toFixed(0);
        }
      } else {
        result = testValue.toString();
      }
      
      return result;
    } catch {
      return "Invalid format";
    }
  }, []);

  const isPending = createMutation.isPending || updateMutation.isPending;

  return (
    <TooltipProvider>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="text-lg font-semibold">Metrics</h3>
            <Badge variant="secondary">{metrics.length}</Badge>
          </div>
          <Button onClick={handleOpenCreate}>
            <Plus className="mr-2 h-4 w-4" />
            Add Metric
          </Button>
        </div>

        <div className="border rounded-md">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-3 py-2 text-left text-sm font-medium">Metric Name</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Verbose Name</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Type</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Expression</th>
                <th className="px-3 py-2 text-left text-sm font-medium">Format</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-24">Certified</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-24">Restricted</th>
                <th className="px-3 py-2 text-center text-sm font-medium w-20">Actions</th>
              </tr>
            </thead>
            <tbody>
              {metrics.map((metric) => (
                <tr key={metric.id} className="border-b hover:bg-muted/30 transition-colors">
                  <td className="px-3 py-2 text-sm font-mono">{metric.metric_name}</td>
                  <td className="px-3 py-2 text-sm">
                    {metric.verbose_name || <span className="text-muted-foreground">-</span>}
                  </td>
                  <td className="px-3 py-2 text-sm">
                    <Badge variant="outline">{metric.metric_type}</Badge>
                  </td>
                  <td className="px-3 py-2 text-sm font-mono text-xs max-w-[200px] truncate">
                    {metric.expression}
                  </td>
                  <td className="px-3 py-2 text-sm">
                    {metric.d3_format || <span className="text-muted-foreground">-</span>}
                  </td>
                  <td className="px-3 py-2 text-center">
                    {metric.certified_by ? (
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <ShieldCheck className="h-4 w-4 text-green-600" />
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-80" align="start">
                          <div className="space-y-2">
                            <div className="flex items-center gap-2">
                              <ShieldCheck className="h-4 w-4 text-green-600" />
                              <span className="font-medium">Certified</span>
                            </div>
                            <p className="text-sm text-muted-foreground">Certified by: {metric.certified_by}</p>
                            {metric.certification_details && (
                              <p className="text-sm">{metric.certification_details}</p>
                            )}
                          </div>
                        </PopoverContent>
                      </Popover>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-center">
                    {metric.is_restricted ? (
                      <Badge variant="destructive">Restricted</Badge>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-center">
                    <div className="flex items-center justify-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => handleOpenEdit(metric)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(metric)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {metrics.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <AlertCircle className="h-8 w-8 mb-2" />
            <p>No metrics defined for this dataset</p>
            <Button variant="link" onClick={handleOpenCreate}>
              Add your first metric
            </Button>
          </div>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>{editingMetric ? "Edit Metric" : "Create Metric"}</DialogTitle>
              <DialogDescription>
                {editingMetric
                  ? "Make changes to the metric. Click save when you're done."
                  : "Create a new metric for this dataset."}
              </DialogDescription>
            </DialogHeader>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="metric_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Metric Name</FormLabel>
                        <FormControl>
                          <Input {...field} placeholder="total_count" />
                        </FormControl>
                        <FormDescription>snake_case, min 3 chars</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="verbose_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Verbose Name</FormLabel>
                        <FormControl>
                          <Input {...field} placeholder="Total Count" />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="metric_type"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Metric Type</FormLabel>
                        <Select value={field.value} onValueChange={handleMetricTypeChange}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Select type" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {METRIC_TYPES.map((type) => (
                              <SelectItem key={type.value} value={type.value}>
                                {type.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="d3_format"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Format</FormLabel>
                        <Select value={field.value || ""} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Select format" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {D3_FORMAT_OPTIONS.map((fmt) => (
                              <SelectItem key={fmt.value} value={fmt.value}>
                                {fmt.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          Preview: <span className="font-mono">{formatPreview(field.value)}</span>
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>

                <FormField
                  control={form.control}
                  name="expression"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Expression</FormLabel>
                      <FormControl>
                        <Textarea
                          {...field}
                          placeholder="SUM(column_name)"
                          className="font-mono text-sm min-h-[80px]"
                        />
                      </FormControl>
                      {getAggregateWarning(field.value) && (
                        <FormDescription className="text-amber-600">
                          Warning: Expression should contain an aggregate function (SUM, COUNT, AVG, MAX, MIN)
                        </FormDescription>
                      )}
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="warning_text"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Warning Text (Optional)</FormLabel>
                      <FormControl>
                        <Textarea
                          {...field}
                          placeholder="Enter warning message for users..."
                          className="min-h-[60px]"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className="flex items-center gap-4">
                  <FormField
                    control={form.control}
                    name="is_restricted"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                        <FormControl>
                          <Switch
                            checked={field.value ?? false}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                        <FormLabel className="font-normal">Restrict access</FormLabel>
                      </FormItem>
                    )}
                  />
                </div>

                {form.watch("is_restricted") && (
                  <div className="grid grid-cols-2 gap-4">
                    <FormField
                      control={form.control}
                      name="certified_by"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Certified By</FormLabel>
                          <FormControl>
                            <Input {...field} placeholder="Enter certifier name" />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    <FormField
                      control={form.control}
                      name="certification_details"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Certification Details</FormLabel>
                          <FormControl>
                            <Input {...field} placeholder="Enter details" />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>
                )}

                <div className="flex justify-end gap-2 pt-4">
                  <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                    Cancel
                  </Button>
                  <Button type="submit" disabled={isPending}>
                    {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    {editingMetric ? "Save Changes" : "Create Metric"}
                  </Button>
                </div>
              </form>
            </Form>
          </DialogContent>
        </Dialog>

        <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete Metric</AlertDialogTitle>
              <AlertDialogDescription>
                Are you sure you want to delete the metric "{deleteTarget?.metric_name}"? This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              >
                {deleteMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Delete
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </TooltipProvider>
  );
}
