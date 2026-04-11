import { create } from "zustand";

export interface AuthUser {
  id: number;
  username: string;
  email: string;
}

interface AuthState {
  user: AuthUser | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  refreshTimer: ReturnType<typeof setTimeout> | null;
  setAuth: (user: AuthUser, accessToken: string) => void;
  clearAuth: () => void;
  setAccessToken: (accessToken: string) => void;
  setRefreshTimer: (timer: ReturnType<typeof setTimeout> | null) => void;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  refreshTimer: null,

  setAuth: (user, accessToken) => {
    const prev = get().refreshTimer;
    if (prev !== null) clearTimeout(prev);
    set({ user, accessToken, isAuthenticated: true, refreshTimer: null });
  },

  clearAuth: () => {
    const prev = get().refreshTimer;
    if (prev !== null) clearTimeout(prev);
    set({ user: null, accessToken: null, isAuthenticated: false, refreshTimer: null });
  },

  setAccessToken: (accessToken) => set({ accessToken }),

  setRefreshTimer: (timer) => set({ refreshTimer: timer }),
}));
