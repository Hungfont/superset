import { useState, useEffect, useRef } from "react";
import { useMutation } from "@tanstack/react-query";
import { queriesApi, type EstimateResult } from "@/api/queries";
import { useSqlLabStore } from "@/stores/sqlLabStore";

function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);
  useEffect(() => {
    const handler = setTimeout(() => setDebouncedValue(value), delay);
    return () => clearTimeout(handler);
  }, [value, delay]);
  return debouncedValue;
}

const SUPPORTED_BACKENDS = ["postgresql", "bigquery", "snowflake", "mysql"] as const;

interface UseEstimateOptions {
  sql: string;
  tabId: string;
  databaseId: number | null;
  backend: string | undefined;
  enabled: boolean;
}

export function useEstimate({ sql, tabId, databaseId, backend, enabled }: UseEstimateOptions) {
  const setEstimate = useSqlLabStore(s => s.setEstimate);
  const debouncedSql = useDebounce(sql, enabled ? 2000 : 0);
  const lastSqlRef = useRef<string>("");

  const isSupported = backend ? (SUPPORTED_BACKENDS as readonly string[]).includes(backend) : false;

  const mutation = useMutation({
    mutationFn: queriesApi.estimate,
    onSuccess: (data: EstimateResult) => {
      setEstimate(tabId, data);
    },
    onError: () => {
      setEstimate(tabId, null);
    },
  });

  useEffect(() => {
    if (!enabled || !databaseId || !isSupported || !debouncedSql.trim()) {
      setEstimate(tabId, null);
      return;
    }
    if (debouncedSql === lastSqlRef.current) return;
    lastSqlRef.current = debouncedSql;
    mutation.mutate({ sql: debouncedSql, database_id: databaseId });
  }, [debouncedSql, databaseId, enabled, isSupported]);

  const trigger = () => {
    if (!databaseId || !sql.trim()) return;
    mutation.mutate({ sql, database_id: databaseId });
  };

  return {
    estimate: useSqlLabStore(s => s.tabs.find(t => t.id === tabId)?.estimate ?? null),
    isLoading: mutation.isPending,
    trigger,
    isSupported,
  };
}
