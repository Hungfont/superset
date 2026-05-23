import { useEffect, useCallback, useRef } from "react";
import { useToast } from "@/hooks/use-toast";
import { updateTab, type UpdateTabRequest } from "@/api/sqllab";
import { useSqlLabStore, type SqlLabTab } from "@/stores/sqlLabStore";
import { useDebounce } from "@/hooks/useDebounce";

type TabForAutoSave = Pick<SqlLabTab, 'id' | 'sql' | 'title' | 'schema' | 'databaseId' | 'catalog' | 'isDirty'>;

export function useAutoSaveTab(activeTabId: string | null, activeTab: TabForAutoSave | undefined) {
  const { toast } = useToast();

  const debouncedSql = useDebounce(activeTab?.sql, 1000);
  const debouncedLabel = useDebounce(activeTab?.title, 1000);

  const activeTabIdRef = useRef(activeTabId);
  activeTabIdRef.current = activeTabId;

  const mountedRef = useRef(false);

  const doSave = useCallback((changes: UpdateTabRequest) => {
    const tabId = activeTabIdRef.current;
    if (!tabId) return;
    const numericId = Number(tabId);
    if (isNaN(numericId)) return;

    updateTab(numericId, changes)
      .then(() => {
        if (activeTabIdRef.current === tabId) {
          useSqlLabStore.getState().clearTabDirty(tabId);
        }
      })
      .catch(() => {
        toast("Failed to save tab. Check connection.");
      });
  }, [toast]);

  // Auto-save SQL (debounced)
  useEffect(() => {
    if (!activeTabId || !activeTab) return;
    if (!mountedRef.current) return;
    doSave({ sql: activeTab.sql });
  }, [debouncedSql]);

  // Auto-save label (debounced)
  useEffect(() => {
    if (!activeTabId || !activeTab) return;
    if (!mountedRef.current) return;
    doSave({ label: activeTab.title });
  }, [debouncedLabel]);

  // Auto-save schema (immediate)
  useEffect(() => {
    if (!activeTabId || !activeTab) return;
    if (!mountedRef.current) return;
    doSave({ schema: activeTab.schema });
  }, [activeTab?.schema]);

  // Auto-save database_id (immediate, when changed)
  useEffect(() => {
    if (!activeTabId || !activeTab) return;
    if (!mountedRef.current) return;
    const dbId = activeTab.databaseId;
    if (dbId == null) return;
    doSave({ db_id: dbId });
  }, [activeTab?.databaseId]);

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
