import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import PermissionsPage from "@/pages/admin/PermissionsPage";
import { permissionsApi } from "@/api/permissions";

vi.mock("@/api/permissions", () => ({
  permissionsApi: {
    getPermissions: vi.fn(),
    createPermission: vi.fn(),
    getViewMenus: vi.fn(),
    createViewMenu: vi.fn(),
    getPermissionViews: vi.fn(),
    createPermissionView: vi.fn(),
    deletePermissionView: vi.fn(),
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
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <PermissionsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("PermissionsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(permissionsApi.getPermissions).mockResolvedValue([
      { id: 1, name: "can_read" },
      { id: 2, name: "can_write" },
    ]);
    vi.mocked(permissionsApi.getViewMenus).mockResolvedValue([
      { id: 10, name: "Dashboard" },
      { id: 11, name: "Chart" },
    ]);
    vi.mocked(permissionsApi.getPermissionViews).mockResolvedValue([
      {
        id: 100,
        permission_id: 1,
        view_menu_id: 10,
        permission_name: "can_read",
        view_menu_name: "Dashboard",
      },
    ]);
  });

  it("renders all three tabs and base matrix labels", async () => {
    renderPage();

    const user = userEvent.setup();

    expect(await screen.findByRole("tab", { name: /permissions/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /view menus/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /permission matrix/i })).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /permission matrix/i }));

    expect(await screen.findByText("can_read")).toBeInTheDocument();
    expect(await screen.findByText("Dashboard")).toBeInTheDocument();
  });

  it("creates and deletes permission-view pairs from matrix toggles", async () => {
    vi.mocked(permissionsApi.createPermissionView).mockResolvedValue({
      id: 101,
      permission_id: 2,
      view_menu_id: 11,
      permission_name: "can_write",
      view_menu_name: "Chart",
    });
    vi.mocked(permissionsApi.deletePermissionView).mockResolvedValue();

    renderPage();

    const user = userEvent.setup();

    await user.click(await screen.findByRole("tab", { name: /permission matrix/i }));

    const createCell = await screen.findByRole("checkbox", { name: /can_write - chart/i });
    await user.click(createCell);

    await waitFor(() => {
      expect(permissionsApi.createPermissionView).toHaveBeenCalledWith(
        { permission_id: 2, view_menu_id: 11 },
        expect.anything(),
      );
    });

    const deleteCell = await screen.findByRole("checkbox", { name: /can_read - dashboard/i });
    await user.click(deleteCell);

    await waitFor(() => {
      expect(permissionsApi.deletePermissionView).toHaveBeenCalledWith(100, expect.anything());
    });
  });
});
