import { z } from "zod";

export const DATABASE_TYPE_OPTIONS = [
  { value: "postgresql", label: "PostgreSQL", defaultPort: 5432, driver: "postgresql" },
  { value: "mysql", label: "MySQL", defaultPort: 3306, driver: "mysql" },
  { value: "snowflake", label: "Snowflake", defaultPort: 443, driver: "snowflake" },
  { value: "bigquery", label: "BigQuery", defaultPort: 443, driver: "bigquery" },
] as const;

export const createDatabaseSchema = z.object({
  db_type: z.string().min(1, "Select a database type"),
  database_name: z.string().trim().min(3, "Database name is required").max(128, "Max 128 characters"),
  host: z.string().trim().min(1, "Host is required"),
  port: z.coerce.number().int().min(1, "Port must be between 1 and 65535").max(65535, "Port must be between 1 and 65535"),
  database: z.string().trim().min(1, "Database is required"),
  username: z.string().trim().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
  allow_dml: z.boolean(),
  expose_in_sqllab: z.boolean(),
  allow_run_async: z.boolean(),
  allow_file_upload: z.boolean(),
  strict_test: z.boolean(),
  save_without_testing: z.boolean(),
  engine_params: z.string().optional(),
});

export type CreateDatabaseFormValues = z.infer<typeof createDatabaseSchema>;

export const CREATE_DATABASE_DEFAULT_VALUES: CreateDatabaseFormValues = {
  db_type: "",
  database_name: "",
  host: "",
  port: 5432,
  database: "",
  username: "",
  password: "",
  allow_dml: false,
  expose_in_sqllab: true,
  allow_run_async: false,
  allow_file_upload: false,
  strict_test: true,
  save_without_testing: false,
  engine_params: "",
};
