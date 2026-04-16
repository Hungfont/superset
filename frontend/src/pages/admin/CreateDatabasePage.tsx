import { useEffect, useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2, XCircle } from "lucide-react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

import { databasesApi, type CreateDatabasePayload, type TestConnectionResult } from "@/api/databases";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Stepper } from "@/components/ui/stepper";
import { useToast } from "@/hooks/use-toast";
import {
  CREATE_DATABASE_DEFAULT_VALUES,
  createDatabaseSchema,
  DATABASE_TYPE_OPTIONS,
  type CreateDatabaseFormValues,
} from "@/lib/validations/database";

function buildSQLAlchemyURI(values: CreateDatabaseFormValues): string {
  const selectedType = DATABASE_TYPE_OPTIONS.find((opt) => opt.value === values.db_type);
  const driver = selectedType?.driver ?? values.db_type;
  const username = encodeURIComponent(values.username);
  const password = encodeURIComponent(values.password);

  return `${driver}://${username}:${password}@${values.host}:${values.port}/${values.database}`;
}

function maskSQLAlchemyURI(sqlalchemyURI: string): string {
  return sqlalchemyURI.replace(/:\/\/([^:]+):([^@]+)@/, "://$1:***@");
}

function toPayload(values: CreateDatabaseFormValues): CreateDatabasePayload {
  return {
    database_name: values.database_name,
    sqlalchemy_uri: buildSQLAlchemyURI(values),
    allow_dml: values.allow_dml,
    expose_in_sqllab: values.expose_in_sqllab,
    allow_run_async: values.allow_run_async,
    allow_file_upload: values.allow_file_upload,
    strict_test: values.strict_test,
  };
}

export default function CreateDatabasePage() {
  const navigate = useNavigate();
  const { success, error } = useToast();

  const [step, setStep] = useState(0);
  const [maxUnlockedStep, setMaxUnlockedStep] = useState(0);
  const [stepError, setStepError] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<TestConnectionResult | null>(null);

  const form = useForm<CreateDatabaseFormValues>({
    resolver: zodResolver(createDatabaseSchema),
    defaultValues: CREATE_DATABASE_DEFAULT_VALUES,
  });

  const selectedType = form.watch("db_type");
  const saveWithoutTesting = form.watch("save_without_testing");
  const watchedHost = form.watch("host");
  const watchedDatabase = form.watch("database");
  const watchedUsername = form.watch("username");
  const watchedPassword = form.watch("password");
  const watchedPort = form.watch("port");
  const uriPreview = useMemo(() => {
    const values = form.getValues();
    if (!values.db_type || !values.host || !values.database || !values.username || !values.password) {
      return "";
    }
    return maskSQLAlchemyURI(buildSQLAlchemyURI(values));
  }, [form, selectedType, watchedHost, watchedDatabase, watchedUsername, watchedPassword, watchedPort]);

  useEffect(() => {
    const option = DATABASE_TYPE_OPTIONS.find((item) => item.value === selectedType);
    if (!option) {
      return;
    }

    form.setValue("port", option.defaultPort, { shouldValidate: true });
  }, [form, selectedType]);

  useEffect(() => {
    const subscription = form.watch((_values, info) => {
      if (!info.name || info.name === "save_without_testing") {
        return;
      }
      setTestResult(null);
    });

    return () => subscription.unsubscribe();
  }, [form]);

  const testMutation = useMutation({
    mutationFn: databasesApi.testConnection,
    onSuccess: (result) => {
      setTestResult(result);
      if (!result.success) {
        error(result.error || "Connection failed");
      }
    },
    onError: (err) => {
      setTestResult({ success: false, error: (err as Error).message || "Connection failed" });
      error((err as Error).message || "Connection failed");
    },
  });

  const createMutation = useMutation({
    mutationFn: databasesApi.createDatabase,
    onSuccess: () => {
      success("Database connected successfully");
      navigate("/admin/settings/databases");
    },
    onError: (err) => {
      error((err as Error).message || "Failed to create database");
    },
  });

  const canSave = !!(testResult?.success || saveWithoutTesting);
  const stepItems = useMemo(
    () => [
      { title: "Select DB Type", description: "Choose your engine", disabled: false },
      { title: "Configure Connection", description: "Host, port, credentials", disabled: maxUnlockedStep < 1 },
      {
        title: "Test & Save",
        description: "Validate and persist encrypted connection",
        disabled: maxUnlockedStep < 2,
      },
    ],
    [maxUnlockedStep],
  );

  async function handleNext() {
    setStepError(null);

    if (step === 0) {
      const dbType = form.getValues("db_type");
      if (!dbType) {
        setStepError("Select a database type before continuing.");
        return;
      }
      setMaxUnlockedStep((prev) => Math.max(prev, 1));
      setStep(1);
      return;
    }

    if (step === 1) {
      const valid = await form.trigger(["database_name", "host", "port", "database", "username", "password"]);
      if (!valid) {
        return;
      }
      setMaxUnlockedStep((prev) => Math.max(prev, 2));
      setStep(2);
    }
  }

  function handleBack() {
    setStepError(null);
    setStep((prev) => Math.max(0, prev - 1));
  }

  async function handleTestConnection() {
    const valid = await form.trigger(["database_name", "host", "port", "database", "username", "password"]);
    if (!valid) {
      setStep(1);
      return;
    }

    const payload = toPayload(form.getValues());
    testMutation.mutate(payload);
  }

  function onSubmit(values: CreateDatabaseFormValues) {
    if (!canSave) {
      return;
    }

    const payload = toPayload(values);
    createMutation.mutate(payload);
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold">Connect a Database</h1>
        <p className="text-sm text-muted-foreground">Create an encrypted connection using a guided three-step flow.</p>
      </header>

      <Stepper
        items={stepItems}
        current={step}
        onStepChange={(index) => {
          if (index > maxUnlockedStep) {
            setStepError("You can only review steps that are already completed.");
            return;
          }
          setStepError(null);
          setStep(index);
        }}
      />

      {stepError && (
        <Alert variant="destructive">
          <AlertTitle>Step validation failed</AlertTitle>
          <AlertDescription>{stepError}</AlertDescription>
        </Alert>
      )}

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-4">
          {step === 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Select DB Type</CardTitle>
                <CardDescription>Choose your engine to auto-populate defaults.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-2">
                {DATABASE_TYPE_OPTIONS.map((option) => {
                  const isActive = selectedType === option.value;
                  return (
                    <Button
                      key={option.value}
                      type="button"
                      variant={isActive ? "default" : "outline"}
                      className="justify-start"
                      aria-label={option.label}
                      onClick={() => {
                        form.setValue("db_type", option.value, { shouldValidate: true });
                        setStepError(null);
                      }}
                    >
                      {option.label}
                    </Button>
                  );
                })}
              </CardContent>
            </Card>
          )}

          {step === 1 && (
            <Card>
              <CardHeader>
                <CardTitle>Configure Connection</CardTitle>
                <CardDescription>Enter connection fields and capability toggles.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4 md:grid-cols-2">
                <FormField
                  control={form.control}
                  name="database_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Database name</FormLabel>
                      <FormControl>
                        <Input placeholder="analytics" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="host"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Host</FormLabel>
                      <FormControl>
                        <Input placeholder="localhost" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="port"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Port</FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="database"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Database</FormLabel>
                      <FormControl>
                        <Input placeholder="analytics" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="username"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Username</FormLabel>
                      <FormControl>
                        <Input placeholder="alice" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Password</FormLabel>
                      <FormControl>
                        <Input type="password" placeholder="Enter password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <div className="md:col-span-2 grid gap-3 md:grid-cols-2">
                  <FormField
                    control={form.control}
                    name="allow_dml"
                    render={({ field }) => (
                      <FormItem className="flex items-center gap-2 rounded border p-2">
                        <Checkbox checked={field.value} onCheckedChange={(checked) => field.onChange(checked === true)} />
                        <FormLabel>Allow DML</FormLabel>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="expose_in_sqllab"
                    render={({ field }) => (
                      <FormItem className="flex items-center gap-2 rounded border p-2">
                        <Checkbox checked={field.value} onCheckedChange={(checked) => field.onChange(checked === true)} />
                        <FormLabel>Expose in SQL Lab</FormLabel>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="allow_run_async"
                    render={({ field }) => (
                      <FormItem className="flex items-center gap-2 rounded border p-2">
                        <Checkbox checked={field.value} onCheckedChange={(checked) => field.onChange(checked === true)} />
                        <FormLabel>Allow Async</FormLabel>
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="allow_file_upload"
                    render={({ field }) => (
                      <FormItem className="flex items-center gap-2 rounded border p-2">
                        <Checkbox checked={field.value} onCheckedChange={(checked) => field.onChange(checked === true)} />
                        <FormLabel>Allow File Upload</FormLabel>
                      </FormItem>
                    )}
                  />
                </div>
              </CardContent>
            </Card>
          )}

          {step === 2 && (
            <Card>
              <CardHeader>
                <CardTitle>Test &amp; Save</CardTitle>
                <CardDescription>Test the connection before persisting encrypted credentials.</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                {uriPreview ? (
                  <Alert>
                    <AlertTitle>Connection string preview</AlertTitle>
                    <AlertDescription>{uriPreview}</AlertDescription>
                  </Alert>
                ) : null}

                <div className="flex items-center gap-2">
                  <Button type="button" onClick={handleTestConnection} disabled={testMutation.isPending}>
                    Test Connection
                  </Button>

                  {testResult?.success ? (
                    <Badge variant="secondary">Connected</Badge>
                  ) : null}

                  {testResult && !testResult.success ? <Badge variant="destructive">Failed</Badge> : null}

                  {typeof testResult?.latency_ms === "number" ? (
                    <Badge variant="outline">{testResult.latency_ms}ms</Badge>
                  ) : null}
                </div>

                {testResult?.success ? (
                  <Alert>
                    <CheckCircle2 className="size-4" />
                    <AlertTitle>Connection successful</AlertTitle>
                    <AlertDescription>
                      {testResult.db_version || "Driver test completed successfully."}
                    </AlertDescription>
                  </Alert>
                ) : null}

                {testResult && !testResult.success ? (
                  <Alert variant="destructive">
                    <XCircle className="size-4" />
                    <AlertTitle>Connection failed</AlertTitle>
                    <AlertDescription>{testResult.error || "Unknown connection error"}</AlertDescription>
                  </Alert>
                ) : null}

                <FormField
                  control={form.control}
                  name="save_without_testing"
                  render={({ field }) => (
                    <FormItem className="flex items-center gap-2 rounded border p-2">
                      <Checkbox
                        id="save_without_testing"
                        checked={field.value}
                        onCheckedChange={(checked) => field.onChange(checked === true)}
                      />
                      <FormLabel htmlFor="save_without_testing">Save without testing</FormLabel>
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>
          )}

          <div className="flex items-center justify-between">
            <Button type="button" variant="outline" onClick={handleBack} disabled={step === 0}>
              Back
            </Button>

            <div className="flex items-center gap-2">
              {step < 2 ? (
                <Button type="button" onClick={handleNext}>
                  Next
                </Button>
              ) : (
                <Button type="submit" disabled={!canSave || createMutation.isPending}>
                  Save
                </Button>
              )}
            </div>
          </div>
        </form>
      </Form>
    </div>
  );
}
