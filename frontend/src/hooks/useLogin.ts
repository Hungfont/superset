import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { authApi } from "@/api/auth";
import { useAuthStore } from "@/stores/authStore";
import type { LoginFormValues } from "@/lib/validations/login";

export function useLogin() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);

  return useMutation({
    mutationFn: (data: LoginFormValues) => authApi.login(data),
    onSuccess: (data) => {
      // Decode minimal claims from the JWT without a library (header.payload.sig)
      const payload = data.access_token.split(".")[1];
      const claims = JSON.parse(atob(payload));
      setAuth(
        { id: Number(claims.sub), username: claims.uname, email: claims.email },
        data.access_token,
      );
      navigate("/");
    },
  });
}
