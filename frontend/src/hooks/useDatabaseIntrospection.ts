import { useQuery } from "@tanstack/react-query";

import {
  databasesApi,
  type DatabaseColumn,
  type DatabaseTable,
  type DatabaseTablesResponse,
} from "@/api/databases";

const DATABASE_SCHEMA_STALE_TIME_MS = 10 * 60 * 1000;

export function useDatabaseSchemasQuery(databaseId?: number, forceRefresh = false) {
  return useQuery<string[]>({
    queryKey: ["db-schemas", databaseId, forceRefresh],
    queryFn: () => databasesApi.getSchemas(databaseId!, forceRefresh),
    enabled: typeof databaseId === "number" && databaseId > 0,
    staleTime: DATABASE_SCHEMA_STALE_TIME_MS,
  });
}

export function useDatabaseTablesQuery(
  databaseId: number | undefined,
  schema: string | undefined,
  page = 1,
  pageSize = 50,
  forceRefresh = false,
) {
  return useQuery<DatabaseTablesResponse>({
    queryKey: ["db-tables", databaseId, schema, page, pageSize, forceRefresh],
    queryFn: () =>
      databasesApi.getTables(databaseId!, {
        schema: schema!,
        page,
        pageSize,
        forceRefresh,
      }),
    enabled: typeof databaseId === "number" && databaseId > 0 && typeof schema === "string" && schema.trim() !== "",
    staleTime: DATABASE_SCHEMA_STALE_TIME_MS,
  });
}

export function useDatabaseColumnsQuery(
  databaseId: number | undefined,
  schema: string | undefined,
  table: string | undefined,
  forceRefresh = false,
) {
  return useQuery<DatabaseColumn[]>({
    queryKey: ["db-columns", databaseId, schema, table, forceRefresh],
    queryFn: () =>
      databasesApi.getColumns(databaseId!, {
        schema: schema!,
        table: table!,
        forceRefresh,
      }),
    enabled:
      typeof databaseId === "number" &&
      databaseId > 0 &&
      typeof schema === "string" &&
      schema.trim() !== "" &&
      typeof table === "string" &&
      table.trim() !== "",
    staleTime: DATABASE_SCHEMA_STALE_TIME_MS,
  });
}

export type { DatabaseTable };
