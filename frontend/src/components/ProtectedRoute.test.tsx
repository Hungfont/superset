import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { ProtectedRoute } from "@/components/ProtectedRoute";

// Prevent useTokenRefresh from scheduling real timers in tests
vi.mock("@/hooks/useTokenRefresh", () => ({ useTokenRefresh: vi.fn() }));

function renderWithRouter(initialPath: string, authenticated: boolean, roles: string[] = []) {
  useAuthStore.setState({
    user: authenticated ? { id: 1, username: "u", email: "u@x.com", roles } : null,
    accessToken: authenticated ? "token" : null,
    isAuthenticated: authenticated,
    refreshTimer: null,
  });

  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route path="/" element={<div>Home</div>} />
        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Route>
        <Route element={<ProtectedRoute requiredRole="Admin" />}>
          <Route path="/admin" element={<div>Admin Area</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("ProtectedRoute", () => {
  it("renders the protected page when authenticated", () => {
    renderWithRouter("/dashboard", true);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
  });

  it("redirects to /login when not authenticated", () => {
    renderWithRouter("/dashboard", false);
    expect(screen.getByText("Login Page")).toBeInTheDocument();
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("renders admin route when user has Admin role", () => {
    renderWithRouter("/admin", true, ["Admin"]);
    expect(screen.getByText("Admin Area")).toBeInTheDocument();
  });

  it("redirects to home when user lacks Admin role", () => {
    renderWithRouter("/admin", true, ["Gamma"]);
    expect(screen.getByText("Home")).toBeInTheDocument();
    expect(screen.queryByText("Admin Area")).not.toBeInTheDocument();
  });
});
