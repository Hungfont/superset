import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { useParams } from "react-router-dom";

import { datasetsApi, type UpdateDatasetMetadataPayload } from "@/api/datasets";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { useToast } from "@/hooks/use-toast";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { DatasetMetadataFormData, datasetMetadataSchema } from "@/lib/validations/dataset";
import { ColumnsTab } from "./ColumnsTab";
import { MetricsTab } from "./MetricsTab";

export default function EditDatasetPage() {
  const { id } = useParams<{ id: string }>();
  const datasetId = id ? parseInt(id, 10) : undefined;
  const { success, error } = useToast();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState("overview");

  const datasetForm = useForm<DatasetMetadataFormData>({
    resolver: zodResolver(datasetMetadataSchema),
    defaultValues: {
      table_name: "",
      description: "",
      main_dttm_col: "",
      cache_timeout: 0,
      normalize_columns: false,
      filter_select_enabled: false,
      is_featured: false,
      sql: "",
    },
  });

  const datasetQuery = useQuery({
    queryKey: ["dataset", datasetId],
    queryFn: () => datasetsApi.getDataset(datasetId!),
    enabled: datasetId !== undefined,
  });

  const updateMutation = useMutation({
    mutationFn: (payload: UpdateDatasetMetadataPayload) =>
      datasetsApi.updateDataset(datasetId!, payload),
    onSuccess: () => {
      success("Dataset saved successfully");
      queryClient.invalidateQueries({ queryKey: ["dataset", datasetId] });
    },
    onError: (err) => {
      const requestError = err as Error & { status?: number; message?: string };
      if (requestError.status === 403) {
        error("You don't have permission to update this dataset");
        return;
      }
      if (requestError.status === 422) {
        error(requestError.message || "Invalid request");
        return;
      }
      error(requestError.message || "Failed to update dataset");
    },
  });

  const dataset = datasetQuery.data;
  const datetimeColumns = dataset?.table_columns?.filter((col) => col.is_dttm) ?? [];
  const isVirtual = dataset?.type === "virtual";

  useEffect(() => {
    if (dataset) {
      datasetForm.reset({
        table_name: dataset.table_name ?? "",
        description: dataset.description ?? "",
        main_dttm_col: dataset.main_dttm_col ?? "",
        cache_timeout: dataset.cache_timeout ?? 0,
        normalize_columns: dataset.normalize_columns ?? false,
        filter_select_enabled: dataset.filter_select_enabled ?? false,
        is_featured: dataset.is_featured ?? false,
        sql: dataset.sql ?? "",
      });
    }
  }, [dataset, datasetForm]);

  if (datasetQuery.isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Loading...</CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4 mt-2" />
        </CardContent>
      </Card>
    );
  }

  const onSubmit = (data: UpdateDatasetMetadataPayload) => {
    updateMutation.mutate(data);
  };

  const hasChanges = datasetForm.formState.isDirty;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Edit Dataset</CardTitle>
        <CardDescription>
          {dataset?.table_name} ({dataset?.type})
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="columns">Columns</TabsTrigger>
              <TabsTrigger value="metrics">Metrics</TabsTrigger>
              <TabsTrigger value="settings">Settings</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-4">
              <Form {...datasetForm}>
                <form onSubmit={datasetForm.handleSubmit(onSubmit)} className="space-y-4">
                  {isVirtual && (
                    <FormField
                      control={datasetForm.control}
                      name="sql"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Virtual SQL</FormLabel>
                          <FormControl>
                            <Textarea
                              className="font-mono text-sm min-h-[200px]"
                              {...field}
                              placeholder="SELECT ..."
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}

                  <FormField
                    control={datasetForm.control}
                    name="table_name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Dataset Name</FormLabel>
                        <FormControl>
                          <Input {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="description"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Description</FormLabel>
                        <FormControl>
                          <Textarea
                            {...field}
                            placeholder="Describe this dataset..."
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="main_dttm_col"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Main Datetime Column</FormLabel>
                        <Select
                          value={field.value ?? ""}
                          onValueChange={field.onChange}
                        >
                          <FormControl>
                            <SelectTrigger>
                              <SelectValue placeholder="Select datetime column" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {datetimeColumns.length === 0 ? (
                              <p className="p-2 text-sm text-muted-foreground">
                                No datetime columns found
                              </p>
                            ) : (
                              datetimeColumns.map((col) => (
                                <SelectItem key={col.id} value={col.column_name}>
                                  {col.column_name}
                                </SelectItem>
                              ))
                            )}
                          </SelectContent>
                        </Select>
                        {datetimeColumns.length === 0 && (
                          <FormDescription>
                            No datetime columns available. Mark a column as datetime in the Columns tab.
                          </FormDescription>
                        )}
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="cache_timeout"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Cache Timeout (seconds)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            {...field}
                            onChange={(e) => field.onChange(e.target.value ? parseInt(e.target.value, 10) : 0)}
                            placeholder="0 = use default"
                          />
                        </FormControl>
                        <FormDescription>
                          0 = use default, -1 = disabled
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="normalize_columns"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                        <div className="space-y-0.5">
                          <FormLabel className="text-base">
                            Normalize Columns
                          </FormLabel>
                          <FormDescription>
                            Normalize column names to lowercase
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value ?? false}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="filter_select_enabled"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                        <div className="space-y-0.5">
                          <FormLabel className="text-base">
                            Filter Select Enabled
                          </FormLabel>
                          <FormDescription>
                            Enable filtering on this dataset
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value ?? false}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={datasetForm.control}
                    name="is_featured"
                    render={({ field }) => (
                      <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                        <div className="space-y-0.5">
                          <FormLabel className="text-base">
                            Featured (Admin only)
                          </FormLabel>
                          <FormDescription>
                            Mark this dataset as featured
                          </FormDescription>
                        </div>
                        <FormControl>
                          <Switch
                            checked={field.value ?? false}
                            onCheckedChange={field.onChange}
                          />
                        </FormControl>
                      </FormItem>
                    )}
                  />

                  <Button
                    type="submit"
                    disabled={!hasChanges || updateMutation.isPending}
                  >
                    {updateMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Save Changes
                  </Button>
                  {hasChanges && (
                    <p className="text-xs text-muted-foreground mt-2">
                      You have unsaved changes
                    </p>
                  )}
                </form>
              </Form>
            </TabsContent>

            <TabsContent value="columns">
              {datasetQuery.isLoading ? (
                <p className="text-sm text-muted-foreground">Loading columns...</p>
              ) : datasetId && dataset?.table_columns !== undefined ? (
                <ColumnsTab datasetId={datasetId} columns={dataset.table_columns} />
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <p>No columns found for this dataset</p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="metrics">
              {datasetQuery.isLoading ? (
                <p className="text-sm text-muted-foreground">Loading metrics...</p>
              ) : datasetId && dataset?.sql_metrics !== undefined ? (
                <MetricsTab datasetId={datasetId} initialMetrics={dataset.sql_metrics} />
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                  <p>No metrics found for this dataset</p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="settings">
              <p className="text-sm text-muted-foreground">
                Additional settings. Cache policy comes in DS-009.
              </p>
            </TabsContent>
          </Tabs>
      </CardContent>
    </Card>
  );
}
