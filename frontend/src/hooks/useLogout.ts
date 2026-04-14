import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { authApi } from "@/api/auth";
import { useToast } from "@/hooks/use-toast";
import { useAuthStore } from "@/stores/authStore";

export function useLogout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const { success, error: notifyError } = useToast();

  return useMutation<void, Error, boolean>({
    mutationFn: (all = false) => authApi.logout(all, useAuthStore.getState().accessToken),
    onSuccess: (_data, all) => {
      clearAuth();
      queryClient.clear();
      navigate("/login", { replace: true });
      success(all ? "Signed out from all devices" : "Signed out");
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : "Unable to sign out";
      notifyError("Sign out failed", { description: message });
    },
  });
}
