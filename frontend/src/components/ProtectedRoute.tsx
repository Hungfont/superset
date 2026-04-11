import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { useTokenRefresh } from "@/hooks/useTokenRefresh";

/**
 * Wraps protected routes. Redirects to /login with saved `from` state when
 * the user is not authenticated. On login, navigate(state.from) returns them
 * to the originally requested path.
 */
export function ProtectedRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const location = useLocation();

  // Schedule silent refresh while the user is authenticated
  useTokenRefresh();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <Outlet />;
}
