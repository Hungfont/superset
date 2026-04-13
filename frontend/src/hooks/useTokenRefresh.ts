/**
 * Schedules a silent token refresh at (exp - 60s) whenever accessToken changes.
 * On success, updates the store and reschedules. On failure, shows a toast and
 * clears auth to redirect the user to /login.
 */

import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { useToast } from "@/hooks/use-toast";

const REFRESH_BEFORE_EXPIRY_MS = 60_000;

function parseExp(token: string): number | null {
  try {
    const payload = token.split(".")[1];
    const claims = JSON.parse(atob(payload)) as { exp?: number };
    return typeof claims.exp === "number" ? claims.exp : null;
  } catch {
    return null;
  }
}

export function useTokenRefresh() {
  const accessToken = useAuthStore((s) => s.accessToken);
  const setAccessToken = useAuthStore((s) => s.setAccessToken);
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const setRefreshTimer = useAuthStore((s) => s.setRefreshTimer);
  const navigate = useNavigate();
  const { toast } = useToast();

  useEffect(() => {
    if (!accessToken) return;

    const exp = parseExp(accessToken);
    if (exp === null) return;

    // Use 0 when the token is already expired or inside the 60-second window
    // so the refresh fires on the next tick rather than silently doing nothing.
    const msUntilRefresh = Math.max(0, exp * 1000 - Date.now() - REFRESH_BEFORE_EXPIRY_MS);

    const timer = setTimeout(async () => {
      try {
        const res = await fetch("/api/v1/auth/refresh", {
          method: "POST",
          credentials: "include",
        });
        if (!res.ok) throw new Error("refresh failed");
        const data = (await res.json()) as { access_token: string };
        setAccessToken(data.access_token);
      } catch {
        clearAuth();
        toast({
          title: "Session expired",
          description: "Your session expired. Please sign in again.",
          variant: "destructive",
          // role="alert" is set by the shadcn Toast component automatically
        });
        navigate("/login");
      }
    }, msUntilRefresh);

    setRefreshTimer(timer);

    return () => {
      clearTimeout(timer);
      setRefreshTimer(null);
    };
  }, [accessToken, clearAuth, navigate, setAccessToken, setRefreshTimer, toast]);
}
