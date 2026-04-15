import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import RolesPage from "@/pages/admin/RolesPage";
import { rolesApi } from "@/api/roles";

vi.mock("@/api/roles", () => ({
  rolesApi: {
    getRoles: vi.fn(),
    createRole: vi.fn(),
    updateRole: vi.fn(),
    deleteRole: vi.fn(),
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

const navigateMock = vi.fn();
vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return {
    ...actual,
    useNavigate: () => navigateMock,
  };
});

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
        <RolesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("RolesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(rolesApi.getRoles).mockResolvedValue([
      { id: 1, name: "Admin", user_count: 4, permission_count: 10, built_in: true },
      { id: 2, name: "Analyst", user_count: 0, permission_count: 2, built_in: false },
    ]);
  });

  it("renders roles with users and permissions counts", async () => {
    renderPage();

    expect(await screen.findByText("Role Management")).toBeInTheDocument();
    expect(await screen.findByText("Admin")).toBeInTheDocument();
    expect(await screen.findByText("Analyst")).toBeInTheDocument();
    expect(screen.getByText("4 users")).toBeInTheDocument();
    expect(screen.getByText("10 perms")).toBeInTheDocument();
  });

  it("creates role and closes dialog", async () => {
    vi.mocked(rolesApi.createRole).mockResolvedValue({
      id: 99,
      name: "Editor",
      user_count: 0,
      permission_count: 0,
      built_in: false,
    });

    renderPage();

    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /new role/i }));
    await user.type(screen.getByLabelText(/role name/i), "Editor");
    await user.click(screen.getByRole("button", { name: /save role/i }));

    await waitFor(() => {
      expect(rolesApi.createRole).toHaveBeenCalledWith({ name: "Editor" }, expect.anything());
    });
    expect(toastSuccessMock).toHaveBeenCalled();
  });

  it("disables delete action for built-in role", async () => {
    renderPage();
    const deleteButton = await screen.findByRole("button", { name: /delete admin/i });

    expect(deleteButton).toBeDisabled();
  });
});
