import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { AlertCircle, CheckCircle2, Code2, Loader2, TableIcon } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { databasesApi, type DatabaseTable } from "@/api/databases";
import { datasetsApi } from "@/api/datasets";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useToast } from "@/hooks/use-toast";

function resolveTableType(table: DatabaseTable): string {
  const candidate = (table as DatabaseTable & { type?: string }).type;
  if (!candidate) {
    return "table";
  }

  const normalized = candidate.trim().toLowerCase();
  if (normalized === "view") {
    return "view";
  }

  return "table";
}

export default function CreateDatasetPage() {
  const navigate = useNavigate();
  const { success, error } = useToast();

  const [selectedDbId, setSelectedDbId] = useState<number | null>(null);
  const [selectedSchema, setSelectedSchema] = useState("");
  const [selectedTable, setSelectedTable] = useState("");
  const [tableSearch, setTableSearch] = useState("");
  const [showViewsOnly, setShowViewsOnly] = useState(false);
  const [showTablesOnly, setShowTablesOnly] = useState(false);

  const [virtualDbId, setVirtualDbId] = useState<number | null>(null);
  const [virtualDatasetName, setVirtualDatasetName] = useState("");
  const [virtualSQL, setVirtualSQL] = useState("");
  const [validateSQL, setValidateSQL] = useState(false);
  const [validationResult, setValidationResult] = useState<{ valid: boolean; message?: string } | null>(null);

  const databasesQuery = useQuery({
    queryKey: ["databases", "dataset-create"],
    queryFn: () => databasesApi.getDatabases({ page: 1, pageSize: 200 }),
  });

  const schemasQuery = useQuery({
    queryKey: ["schemas", selectedDbId],
    queryFn: () => databasesApi.getSchemas(selectedDbId!),
    enabled: selectedDbId !== null,
  });

  const tablesQuery = useQuery({
    queryKey: ["tables", selectedDbId, selectedSchema],
    queryFn: () => databasesApi.getTables(selectedDbId!, { schema: selectedSchema, page: 1, pageSize: 200 }),
    enabled: selectedDbId !== null && selectedSchema !== "",
  });

  const filteredTables = useMemo(() => {
    const allTables = tablesQuery.data?.items ?? [];

    let result = allTables;
    if (tableSearch.trim() !== "") {
      const normalizedSearch = tableSearch.trim().toLowerCase();
      result = result.filter((table) => table.name.toLowerCase().includes(normalizedSearch));
    }

    if (showViewsOnly && !showTablesOnly) {
      result = result.filter((table) => resolveTableType(table) === "view");
    }

    if (showTablesOnly && !showViewsOnly) {
      result = result.filter((table) => resolveTableType(table) !== "view");
    }

    return result;
  }, [showTablesOnly, showViewsOnly, tableSearch, tablesQuery.data?.items]);

  const createMutation = useMutation({
    mutationFn: datasetsApi.createDataset,
    onSuccess: (created) => {
      success("Dataset created. Columns are being synced...");
      navigate(`/admin/settings/datasets/${created.id}/edit`);
    },
    onError: (err) => {
      const requestError = err as Error & { status?: number };
      if (requestError.status === 403) {
        error("Gamma role cannot create datasets");
        return;
      }
      if (requestError.status === 409) {
        error("Dataset already exists");
        return;
      }
      if (requestError.status === 422) {
        error("Invalid database selection");
        return;
      }

      error(requestError.message || "Failed to create dataset");
    },
  });

  const canSubmit = selectedDbId !== null && selectedSchema !== "" && selectedTable !== "" && !createMutation.isPending;

  const virtualCreateMutation = useMutation({
    mutationFn: datasetsApi.createVirtualDataset,
    onSuccess: (created) => {
      success("Virtual dataset created. Columns are being synced...");
      navigate(`/admin/settings/datasets/${created.id}/edit`);
    },
    onError: (err) => {
      const requestError = err as Error & { status?: number; message?: string };
      if (requestError.status === 403) {
        error("Gamma role cannot create datasets");
        return;
      }
      if (requestError.status === 409) {
        error("Dataset already exists");
        return;
      }
      if (requestError.status === 422) {
        error(requestError.message || "Invalid SQL");
        return;
      }

      error(requestError.message || "Failed to create virtual dataset");
    },
  });

  const canSubmitVirtual =
    virtualDbId !== null &&
    virtualDatasetName.trim() !== "" &&
    virtualSQL.trim() !== "" &&
    !virtualCreateMutation.isPending;

  const detectSQLIssues = (sql: string) => {
    const issues: string[] = [];
    const upperSQL = sql.toUpperCase().trim();
    if (!upperSQL.startsWith("SELECT")) {
      issues.push("SQL must start with SELECT");
    }
    if (sql.includes(";")) {
      issues.push("SQL should not contain semicolons");
    }
    return issues;
  };

  const sqlIssues = detectSQLIssues(virtualSQL);

  return (
    <div className="flex flex-col gap-4">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold">Create Dataset</h1>
        <p className="text-sm text-muted-foreground">
          Register a physical table as a dataset. Column metadata will sync in the background.
        </p>
      </header>

      <Tabs defaultValue="physical" className="flex flex-col gap-4">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="physical">Physical Table</TabsTrigger>
          <TabsTrigger value="virtual">Virtual SQL</TabsTrigger>
        </TabsList>

        <TabsContent value="physical" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Physical Dataset Wizard</CardTitle>
              <CardDescription>Select database, schema, and table before creating the dataset.</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <p className="text-sm font-medium">1. Select Database</p>
                {databasesQuery.isLoading ? (
                  <Skeleton className="h-20 w-full" />
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {(databasesQuery.data?.items ?? []).map((database) => (
                      <Button
                        key={database.id}
                        type="button"
                        variant={selectedDbId === database.id ? "default" : "outline"}
                        onClick={() => {
                          setSelectedDbId(database.id);
                          setSelectedSchema("");
                          setSelectedTable("");
                        }}
                      >
                        {database.database_name}
                      </Button>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-2">
                <p className="text-sm font-medium">2. Select Schema</p>
                {schemasQuery.isLoading ? (
                  <Skeleton className="h-16 w-full" />
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {(schemasQuery.data ?? []).map((schema) => (
                      <Button
                        key={schema}
                        type="button"
                        variant={selectedSchema === schema ? "default" : "outline"}
                        disabled={selectedDbId === null}
                        onClick={() => {
                          setSelectedSchema(schema);
                          setSelectedTable("");
                        }}
                      >
                        {schema}
                      </Button>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-3">
                <p className="text-sm font-medium">3. Select Table</p>
                <Input
                  placeholder="Search table name"
                  value={tableSearch}
                  onChange={(event) => setTableSearch(event.target.value)}
                  disabled={selectedSchema === ""}
                />

                <div className="flex flex-wrap gap-4">
                  <label className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={showViewsOnly}
                      onCheckedChange={(checked) => setShowViewsOnly(checked === true)}
                    />
                    <span>Show views only</span>
                  </label>
                  <label className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={showTablesOnly}
                      onCheckedChange={(checked) => setShowTablesOnly(checked === true)}
                    />
                    <span>Show tables only</span>
                  </label>
                </div>

                {tablesQuery.isLoading ? (
                  <Skeleton className="h-32 w-full" />
                ) : (
                  <div className="flex max-h-72 flex-col gap-2 overflow-y-auto rounded-md border p-2">
                    {filteredTables.map((table) => {
                      const tableType = resolveTableType(table);
                      return (
                        <Button
                          key={table.name}
                          type="button"
                          variant={selectedTable === table.name ? "default" : "ghost"}
                          className="justify-between"
                          onClick={() => setSelectedTable(table.name)}
                        >
                          <span className="flex items-center gap-2">
                            <TableIcon className="h-4 w-4" />
                            {table.name}
                          </span>
                          <Badge variant="secondary">{tableType}</Badge>
                        </Button>
                      );
                    })}
                    {filteredTables.length === 0 && (
                      <p className="text-sm text-muted-foreground">No tables found for current filters.</p>
                    )}
                  </div>
                )}
              </div>

              <div className="flex justify-end">
                <Button
                  type="button"
                  disabled={!canSubmit}
                  onClick={() =>
                    createMutation.mutate({
                      database_id: selectedDbId!,
                      schema: selectedSchema,
                      table_name: selectedTable,
                    })
                  }
                >
                  {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Create Dataset
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="virtual" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Virtual SQL Dataset</CardTitle>
              <CardDescription>
                Write a custom SELECT query to define a virtual dataset. The query will be validated before saving.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <p className="text-sm font-medium">1. Select Database</p>
                {databasesQuery.isLoading ? (
                  <Skeleton className="h-20 w-full" />
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {(databasesQuery.data?.items ?? []).map((database) => (
                      <Button
                        key={database.id}
                        type="button"
                        variant={virtualDbId === database.id ? "default" : "outline"}
                        onClick={() => {
                          setVirtualDbId(database.id);
                          setValidationResult(null);
                        }}
                      >
                        {database.database_name}
                      </Button>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex flex-col gap-2">
                <p className="text-sm font-medium">2. Dataset Name</p>
                <Input
                  placeholder="e.g., revenue_by_month"
                  value={virtualDatasetName}
                  onChange={(e) => setVirtualDatasetName(e.target.value)}
                  disabled={virtualDbId === null}
                />
              </div>

              <div className="flex flex-col gap-2">
                <p className="text-sm font-medium">3. SQL Query</p>
                <div className="rounded-md border">
                  <div className="flex items-center gap-2 border-b bg-muted/50 px-3 py-2">
                    <Code2 className="h-4 w-4" />
                    <span className="text-xs text-muted-foreground">SELECT * FROM ...</span>
                  </div>
                  <textarea
                    className="min-h-48 w-full resize-none border-0 bg-transparent p-3 font-mono text-sm focus:outline-none focus:ring-0"
                    placeholder="SELECT column1, column2 FROM table WHERE ..."
                    value={virtualSQL}
                    onChange={(e) => setVirtualSQL(e.target.value)}
                    disabled={virtualDbId === null}
                  />
                </div>

                {sqlIssues.length > 0 && (
                  <Alert variant="destructive" className="mt-2">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>SQL Issues</AlertTitle>
                    <AlertDescription>
                      <ul className="list-inside list-disc">
                        {sqlIssues.map((issue, i) => (
                          <li key={i}>{issue}</li>
                        ))}
                      </ul>
                    </AlertDescription>
                  </Alert>
                )}

                {validationResult && (
                  <Alert
                    variant={validationResult.valid ? "default" : "destructive"}
                    className={validationResult.valid ? "border-green-500 bg-green-50" : "border-red-500 bg-red-50"}
                  >
                    {validationResult.valid ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                    ) : (
                      <AlertCircle className="h-4 w-4" />
                    )}
                    <AlertTitle>{validationResult.valid ? "SQL Valid" : "Validation Error"}</AlertTitle>
                    <AlertDescription>{validationResult.message}</AlertDescription>
                  </Alert>
                )}
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  id="validate-sql"
                  checked={validateSQL}
                  onCheckedChange={(checked) => {
                    setValidateSQL(checked === true);
                    setValidationResult(null);
                  }}
                  disabled={virtualDbId === null || sqlIssues.length > 0}
                />
                <label htmlFor="validate-sql" className="text-sm">
                  Validate SQL before saving
                </label>
              </div>

              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  disabled={
                    !canSubmitVirtual ||
                    sqlIssues.length > 0 ||
                    (validateSQL && !validationResult?.valid)
                  }
                  onClick={() =>
                    virtualCreateMutation.mutate({
                      database_id: virtualDbId!,
                      table_name: virtualDatasetName,
                      sql: virtualSQL,
                      validate_sql: validateSQL,
                    })
                  }
                >
                  {virtualCreateMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create Virtual Dataset
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
