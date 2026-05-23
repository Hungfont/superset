import { useEffect, useCallback, useRef } from "react";
import { useToast } from "@/hooks/use-toast";
import { updateTab } from "@/api/sqllab";
import { useSqlLabStore, type SqlLabTab } from "@/stores/sqlLabStore";
import { useDebounce } from "@/hooks/useDebounce";

type TabForAutoSave = Pick<SqlLabTab, 'id' | 'sql' | 'title' | 'schema' | 'databaseId' | 'isDirty'>;

export function useAutoSaveTab(activeTabId: string | null, activeTab: TabForAutoSave | undefined) {
  const { toast } = useToast();

  const debouncedSql = useDebounce(activeTab?.sql, 1000);
  const debouncedLabel = useDebounce(activeTab?.title, 1000);

  const activeTabIdRef = useRef(activeTabId);
  activeTabIdRef.current = activeTabId;

  const mountedRef = useRef(false);

  // Auto-save: single atomic write of all fields to prevent race conditions
  useEffect(() => {
    if (!activeTabId || !activeTab || !mountedRef.current) return;
    const tabId = activeTabIdRef.current;
    if (!tabId) return;
    const numericId = Number(tabId);
    if (isNaN(numericId)) return;

    updateTab(numericId, {
      sql: activeTab.sql,
      label: activeTab.title,
      schema: activeTab.schema,
      db_id: activeTab.databaseId ?? undefined,
    })
      .then(() => {
        if (activeTabIdRef.current === tabId) {
          useSqlLabStore.getState().clearTabDirty(tabId);
        }
      })
      .catch(() => {
        toast("Failed to save tab. Check connection.");
      });
  }, [debouncedSql, debouncedLabel, activeTab?.schema, activeTab?.databaseId]);

  useEffect(() => {
    mountedRef.current = true;
  }, []);

  const linkLatestQuery = useCallback((queryId: string) => {
    if (!activeTabIdRef.current) return;
    const numericId = Number(activeTabIdRef.current);
    if (isNaN(numericId)) return;

    updateTab(numericId, { latest_query_id: queryId }).catch(() => {
      // Silently ignore linkage failures
    });
  }, []);

  return { linkLatestQuery };
}
