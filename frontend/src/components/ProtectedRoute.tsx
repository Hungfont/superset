import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { useTokenRefresh } from "@/hooks/useTokenRefresh";

interface ProtectedRouteProps {
  requiredRole?: string;
}

/**
 * Wraps protected routes. Redirects to /login with saved `from` state when
 * the user is not authenticated. On login, navigate(state.from) returns them
 * to the originally requested path.
 */
export function ProtectedRoute({ requiredRole }: ProtectedRouteProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const user = useAuthStore((s) => s.user);
  const location = useLocation();

  // Schedule silent refresh while the user is authenticated
  useTokenRefresh();

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (requiredRole) {
    const userRoles = user?.roles ?? [];
    const isRoleMatched = userRoles.some(
      (role) => role.toLowerCase() === requiredRole.toLowerCase(),
    );

    if (!isRoleMatched) {
      return <Navigate to="/" replace />;
    }
  }

  return <Outlet />;
}
