import { useCallback, useState } from "react";

type LoadingCounts = Record<string, number>;
type AsyncOrSyncAction<T> = () => Promise<T> | T;

const DEFAULT_LOADING_KEY = "default";

export function useLoading() {
  const [loadingCounts, setLoadingCounts] = useState<LoadingCounts>({});

  const startLoading = useCallback((key = DEFAULT_LOADING_KEY) => {
    setLoadingCounts((previousCounts) => ({
      ...previousCounts,
      [key]: (previousCounts[key] ?? 0) + 1,
    }));
  }, []);

  const stopLoading = useCallback((key = DEFAULT_LOADING_KEY) => {
    setLoadingCounts((previousCounts) => {
      const nextCount = (previousCounts[key] ?? 0) - 1;
      if (nextCount > 0) {
        return {
          ...previousCounts,
          [key]: nextCount,
        };
      }

      const { [key]: _removedCount, ...restCounts } = previousCounts;
      return restCounts;
    });
  }, []);

  const isLoading = useCallback(
    (key?: string) => {
      if (key) {
        return Boolean(loadingCounts[key]);
      }

      return Object.keys(loadingCounts).length > 0;
    },
    [loadingCounts],
  );

  const withLoading = useCallback(
    async <T>(
      keyOrAction: string | AsyncOrSyncAction<T>,
      maybeAction?: AsyncOrSyncAction<T>,
    ): Promise<T> => {
      const key = typeof keyOrAction === "string" ? keyOrAction : DEFAULT_LOADING_KEY;
      const action = typeof keyOrAction === "string" ? maybeAction : keyOrAction;

      if (!action) {
        throw new Error("withLoading requires an action callback");
      }

      startLoading(key);
      try {
        return await action();
      } finally {
        stopLoading(key);
      }
    },
    [startLoading, stopLoading],
  );

  return {
    isLoading,
    startLoading,
    stopLoading,
    withLoading,
  };
}
