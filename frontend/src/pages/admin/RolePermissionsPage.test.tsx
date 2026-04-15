import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import RolePermissionsPage from "@/pages/admin/RolePermissionsPage";
import { rolesApi } from "@/api/roles";
import { permissionsApi } from "@/api/permissions";

vi.mock("@/api/roles", () => ({
  rolesApi: {
    getRoles: vi.fn(),
    getRolePermissions: vi.fn(),
    setRolePermissions: vi.fn(),
  },
}));

vi.mock("@/api/permissions", () => ({
  permissionsApi: {
    getPermissionViews: vi.fn(),
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
    <MemoryRouter initialEntries={["/admin/settings/roles/5/permissions"]}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/admin/settings/roles/:id/permissions" element={<RolePermissionsPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RolePermissionsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(rolesApi.getRoles).mockResolvedValue([
      { id: 5, name: "Analyst", user_count: 0, permission_count: 1, built_in: false },
    ]);
    vi.mocked(rolesApi.getRolePermissions).mockResolvedValue([100]);
    vi.mocked(rolesApi.setRolePermissions).mockResolvedValue([100, 101]);
    vi.mocked(permissionsApi.getPermissionViews).mockResolvedValue([
      { id: 100, permission_id: 1, view_menu_id: 10, permission_name: "can_read", view_menu_name: "Dashboard" },
      { id: 101, permission_id: 2, view_menu_id: 10, permission_name: "can_write", view_menu_name: "Dashboard" },
    ]);
  });

  it("tracks unsaved changes and enables Save", async () => {
    renderPage();

    const user = userEvent.setup();
    const addPermissionCheckbox = await screen.findByRole("checkbox", { name: /can_write.*dashboard/i });
    await user.click(addPermissionCheckbox);

    expect(screen.getByText(/unsaved changes/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save changes/i })).toBeEnabled();
  });

  it("prevents saving when no permission remains selected", async () => {
    renderPage();

    const user = userEvent.setup();
    const selectedCheckbox = await screen.findByRole("checkbox", { name: /can_read.*dashboard/i });
    await user.click(selectedCheckbox);

    expect(screen.getByRole("button", { name: /save changes/i })).toBeDisabled();
    expect(screen.getByText(/at least one permission/i)).toBeInTheDocument();
  });

  it("saves full assignment via PUT endpoint", async () => {
    renderPage();

    const user = userEvent.setup();
    const addPermissionCheckbox = await screen.findByRole("checkbox", { name: /can_write.*dashboard/i });
    await user.click(addPermissionCheckbox);
    await user.click(screen.getByRole("button", { name: /save changes/i }));

    await waitFor(() => {
      expect(rolesApi.setRolePermissions).toHaveBeenCalledWith(5, [100, 101]);
    });
    expect(toastSuccessMock).toHaveBeenCalled();
  });
});
