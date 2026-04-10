import { useMutation } from "@tanstack/react-query";
import { authApi, type RegisterPayload } from "@/api/auth";

export function useRegister() {
  return useMutation({
    mutationFn: (data: RegisterPayload) => authApi.register(data),
  });
}
