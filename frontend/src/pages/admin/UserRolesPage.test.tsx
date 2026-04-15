import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import UserRolesPage from "@/pages/admin/UserRolesPage";
import { rolesApi } from "@/api/roles";
import { userRolesApi } from "@/api/userRoles";

vi.mock("@/api/roles", () => ({
  rolesApi: {
    getRoles: vi.fn(),
  },
}));

vi.mock("@/api/userRoles", () => ({
  userRolesApi: {
    getUserRoles: vi.fn(),
    setUserRoles: vi.fn(),
  },
}));

const toastSuccessMock = vi.fn();
const toastErrorMock = vi.fn();

vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({
    success: toastSuccessMock,
    error: toastErrorMock,
  }),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <MemoryRouter initialEntries={["/admin/settings/users/7"]}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/admin/settings/users/:id" element={<UserRolesPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("UserRolesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(rolesApi.getRoles).mockResolvedValue([
      { id: 1, name: "Admin", user_count: 1, permission_count: 10, built_in: true },
      { id: 3, name: "Gamma", user_count: 2, permission_count: 6, built_in: true },
      { id: 5, name: "Analyst", user_count: 4, permission_count: 3, built_in: false },
    ]);

    vi.mocked(userRolesApi.getUserRoles).mockResolvedValue([1, 3]);
    vi.mocked(userRolesApi.setUserRoles).mockResolvedValue([1, 3, 5]);
  });

  it("shows currently selected role badges", async () => {
    renderPage();

    expect(await screen.findByText("Admin")).toBeInTheDocument();
    expect(screen.getByText("Gamma")).toBeInTheDocument();
  });

  it("disables save and shows error when all roles removed", async () => {
    renderPage();

    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /remove role admin/i }));
    await user.click(screen.getByRole("button", { name: /remove role gamma/i }));

    expect(screen.getByText(/user must have at least one role/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /update roles/i })).toBeDisabled();
  });

  it("updates roles with PUT endpoint payload", async () => {
    renderPage();

    const user = userEvent.setup();
    await user.click(await screen.findByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: /toggle role analyst/i }));
    await user.click(screen.getByRole("button", { name: /update roles/i }));

    await waitFor(() => {
      expect(userRolesApi.setUserRoles).toHaveBeenCalledWith(7, [1, 3, 5]);
    });
    expect(toastSuccessMock).toHaveBeenCalled();
  });
});
