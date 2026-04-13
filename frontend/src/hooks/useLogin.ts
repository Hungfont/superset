import { useMutation } from "@tanstack/react-query";
import { useNavigate, useLocation } from "react-router-dom";
import { authApi } from "@/api/auth";
import { useAuthStore } from "@/stores/authStore";
import type { LoginFormValues } from "@/lib/validations/login";

export function useLogin() {
  const navigate = useNavigate();
  const location = useLocation();
  const setAuth = useAuthStore((s) => s.setAuth);

  return useMutation({
    mutationFn: (data: LoginFormValues) => authApi.login(data),
    onSuccess: (data) => {
      // Decode minimal claims from the JWT without a library (header.payload.sig)
      const payload = data.access_token.split(".")[1];
      const claims = JSON.parse(atob(payload)) as {
        sub: string;
        uname: string;
        email: string;
        role?: string | string[];
        roles?: string[];
      };

      const rolesFromClaims = Array.isArray(claims.roles)
        ? claims.roles
        : Array.isArray(claims.role)
          ? claims.role
          : typeof claims.role === "string"
            ? [claims.role]
            : [];

      setAuth(
        {
          id: Number(claims.sub),
          username: claims.uname,
          email: claims.email,
          roles: rolesFromClaims,
        },
        data.access_token,
      );
      // Redirect back to the page the user originally requested, or "/".
      // Validate the path is same-origin relative to prevent open redirects.
      const rawFrom = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname;
      const from = rawFrom && rawFrom.startsWith("/") && !rawFrom.startsWith("//") ? rawFrom : "/";
      navigate(from, { replace: true });
    },
  });
}
