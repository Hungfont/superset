import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import LoginPage from "./LoginPage";

vi.mock("@/api/auth", () => ({
  authApi: {
    login: vi.fn(),
  },
}));

// Prevent Zustand authStore from leaking state between tests
vi.mock("@/lib/api/client", () => ({
  isTokenExpired: vi.fn().mockReturnValue(false),
}));

const mockAuthState = {
  user: null as { id: number; username: string; email: string } | null,
  accessToken: null as string | null,
  isAuthenticated: false,
  setAuth: vi.fn(),
  clearAuth: vi.fn(),
};

vi.mock("@/stores/authStore", () => ({
  useAuthStore: vi.fn((selector: (s: typeof mockAuthState) => unknown) =>
    selector(mockAuthState)
  ),
}));

import { authApi } from "@/api/auth";
import { isTokenExpired } from "@/lib/api/client";

function renderPage(route = "/login") {
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: 0 } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[route]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<div>Home page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

// Minimal valid JWT for testing (header.payload.sig — not cryptographically valid)
const fakeJwt = [
  btoa(JSON.stringify({ alg: "RS256", typ: "JWT" })),
  btoa(JSON.stringify({ sub: "1", uname: "johndoe", email: "john@example.com" })),
  "fakesig",
].join(".");

describe("LoginPage", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset auth state to unauthenticated between tests
    mockAuthState.user = null;
    mockAuthState.accessToken = null;
    mockAuthState.isAuthenticated = false;
    vi.mocked(isTokenExpired).mockReturnValue(false);
  });

  it("renders username and password fields and sign-in button", () => {
    renderPage();
    expect(screen.getByLabelText(/username or email/i)).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("shows validation error when submitting empty form", async () => {
    renderPage();
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() => {
      expect(screen.getByText(/username is required/i)).toBeInTheDocument();
    });
  });

  it("shows ?activated=true success alert", () => {
    renderPage("/login?activated=true");
    expect(screen.getByText(/account activated/i)).toBeInTheDocument();
  });

  it("disables button and shows spinner while request is pending", async () => {
    vi.mocked(authApi.login).mockReturnValue(new Promise(() => {}));
    renderPage();
    await user.type(screen.getByLabelText(/username or email/i), "johndoe");
    await user.type(screen.getByLabelText("Password"), "password123");
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /sign in/i })).toBeDisabled();
    });
  });

  it("shows invalid credentials alert on 401", async () => {
    const err = Object.assign(new Error("invalid credentials"), { status: 401 });
    vi.mocked(authApi.login).mockRejectedValue(err);
    renderPage();
    await user.type(screen.getByLabelText(/username or email/i), "johndoe");
    await user.type(screen.getByLabelText("Password"), "wrongpass");
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
    });
  });

  it("shows lockout alert with locked_until on 423", async () => {
    const lockedUntil = new Date(Date.now() + 10 * 60 * 1000).toISOString();
    const err = Object.assign(new Error("account locked"), {
      status: 423,
      locked_until: lockedUntil,
    });
    vi.mocked(authApi.login).mockRejectedValue(err);
    renderPage();
    await user.type(screen.getByLabelText(/username or email/i), "johndoe");
    await user.type(screen.getByLabelText("Password"), "wrongpass");
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() => {
      expect(screen.getByText(/account locked/i)).toBeInTheDocument();
    });
  });

  it("shows password with toggle button", async () => {
    renderPage();
    const passwordInput = screen.getByLabelText("Password");
    expect(passwordInput).toHaveAttribute("type", "password");
    await user.click(screen.getByRole("button", { name: /show password/i }));
    expect(passwordInput).toHaveAttribute("type", "text");
  });

  it("pre-fills username from localStorage when remember-me was checked", () => {
    window.localStorage.setItem("rememberedUsername", "saveduser");
    renderPage();
    expect(screen.getByLabelText(/username or email/i)).toHaveValue("saveduser");
    window.localStorage.removeItem("rememberedUsername");
  });

  it("navigates to home on successful login", async () => {
    vi.mocked(authApi.login).mockResolvedValue({
      access_token: fakeJwt,
      refresh_token: "refresh-abc",
    });
    renderPage();
    await user.type(screen.getByLabelText(/username or email/i), "johndoe");
    await user.type(screen.getByLabelText("Password"), "StrongP@ss1!");
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() => {
      expect(screen.getByText(/home page/i)).toBeInTheDocument();
    });
  });

  // ── Already-authenticated redirect ────────────────────────────────────────

  it("redirects to / immediately when user already has a valid (non-expired) session", () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.accessToken = "some-token";
    mockAuthState.user = { id: 1, username: "johndoe", email: "john@example.com" };
    vi.mocked(isTokenExpired).mockReturnValue(false);

    renderPage();

    expect(screen.getByText(/home page/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/username or email/i)).not.toBeInTheDocument();
  });

  it("shows the login form when user is authenticated but token is expired", () => {
    mockAuthState.isAuthenticated = true;
    mockAuthState.accessToken = "expired-token";
    mockAuthState.user = { id: 1, username: "johndoe", email: "john@example.com" };
    vi.mocked(isTokenExpired).mockReturnValue(true);

    renderPage();

    expect(screen.getByLabelText(/username or email/i)).toBeInTheDocument();
    expect(screen.queryByText(/home page/i)).not.toBeInTheDocument();
  });

  it("shows the login form when user is not authenticated", () => {
    mockAuthState.isAuthenticated = false;
    mockAuthState.accessToken = null;
    mockAuthState.user = null;

    renderPage();

    expect(screen.getByLabelText(/username or email/i)).toBeInTheDocument();
    expect(screen.queryByText(/home page/i)).not.toBeInTheDocument();
  });
});
