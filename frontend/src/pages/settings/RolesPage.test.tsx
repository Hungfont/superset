import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import RolesPage from "./RolesPage";
import { rolesApi } from "@/api/roles";

vi.mock("@/api/roles", () => ({
  rolesApi: {
    getRoles: vi.fn(),
    createRole: vi.fn(),
    updateRole: vi.fn(),
    deleteRole: vi.fn(),
  },
}));

const toastSpy = vi.fn();
vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({ toast: toastSpy }),
}));

const sampleRoles = [
  { id: 1, name: "Admin", user_count: 2, permission_count: 12, built_in: true },
  { id: 2, name: "Viewer", user_count: 0, permission_count: 3, built_in: false },
];

function renderPage() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/admin/settings/roles"]}>
        <Routes>
          <Route path="/admin/settings/roles" element={<Auth007RoleCrud />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Auth007RoleCrud", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(rolesApi.getRoles).mockResolvedValue(sampleRoles);
    vi.mocked(rolesApi.createRole).mockResolvedValue({
      id: 9,
      name: "Analyst",
      user_count: 0,
      permission_count: 0,
      built_in: false,
    });
    vi.mocked(rolesApi.updateRole).mockResolvedValue({
      id: 2,
      name: "Editor",
      user_count: 0,
      permission_count: 3,
      built_in: false,
    });
    vi.mocked(rolesApi.deleteRole).mockResolvedValue();
  });

  it("renders AUTH-007 role CRUD header", async () => {
    renderPage();
    await screen.findByText("Viewer");
    expect(screen.getByRole("heading", { name: /auth-007 role crud management/i })).toBeInTheDocument();
  });

  it("creates a role from the dialog", async () => {
    renderPage();
    await screen.findByText("Viewer");

    await user.click(screen.getByRole("button", { name: /new role/i }));
    await user.type(screen.getByPlaceholderText(/role name/i), "Analyst");
    await user.click(screen.getByRole("button", { name: /^create$/i }));

    await waitFor(() => {
      expect(rolesApi.createRole).toHaveBeenCalled();
    });
    expect(vi.mocked(rolesApi.createRole).mock.calls[0]?.[0]).toEqual({ name: "Analyst" });
  });

  it("edits a role from the actions menu", async () => {
    renderPage();
    await screen.findByText("Viewer");

    await user.click(screen.getByRole("button", { name: /actions for viewer/i }));
    await user.click(await screen.findByRole("menuitem", { name: /^edit$/i }));

    const input = screen.getByPlaceholderText(/role name/i);
    await user.clear(input);
    await user.type(input, "Editor");
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() => {
      expect(rolesApi.updateRole).toHaveBeenCalledWith(2, { name: "Editor" });
    });
  });

  it("deletes a non-built-in role after confirmation", async () => {
    renderPage();
    await screen.findByText("Viewer");

    await user.click(screen.getByRole("button", { name: /actions for viewer/i }));
    await user.click(await screen.findByRole("menuitem", { name: /^delete$/i }));
    await user.click(screen.getByRole("button", { name: /^delete$/i }));

    await waitFor(() => {
      expect(rolesApi.deleteRole).toHaveBeenCalled();
    });
    expect(vi.mocked(rolesApi.deleteRole).mock.calls[0]?.[0]).toBe(2);
  });

  it("keeps delete disabled for built-in roles", async () => {
    renderPage();
    await screen.findByText("Admin");

    await user.click(screen.getByRole("button", { name: /actions for admin/i }));
    const deleteItem = await screen.findByRole("menuitem", { name: /^delete$/i });

    expect(deleteItem).toHaveAttribute("aria-disabled", "true");
  });

  it("shows assigned-users warning in delete dialog when role has users", async () => {
    vi.mocked(rolesApi.getRoles).mockResolvedValueOnce([
      { id: 3, name: "Moderator", user_count: 5, permission_count: 2, built_in: false },
    ]);

    renderPage();
    await screen.findByText("Moderator");

    await user.click(screen.getByRole("button", { name: /actions for moderator/i }));
    await user.click(await screen.findByRole("menuitem", { name: /^delete$/i }));

    expect(screen.getByText(/5 assigned users/i)).toBeInTheDocument();
  });

  it("shows conflict toast when delete fails with status 409", async () => {
    vi.mocked(rolesApi.deleteRole).mockRejectedValueOnce({ message: "role has assigned users", status: 409 });

    renderPage();
    await screen.findByText("Viewer");

    await user.click(screen.getByRole("button", { name: /actions for viewer/i }));
    await user.click(await screen.findByRole("menuitem", { name: /^delete$/i }));
    await user.click(screen.getByRole("button", { name: /^delete$/i }));

    await waitFor(() => {
      expect(toastSpy).toHaveBeenCalledWith(
        expect.objectContaining({ title: "Delete failed", variant: "destructive" }),
      );
    });
  });

  it("resets form when dialog is closed", async () => {
    renderPage();
    await screen.findByText("Viewer");

    await user.click(screen.getByRole("button", { name: /actions for viewer/i }));
    await user.click(await screen.findByRole("menuitem", { name: /^edit$/i }));
    expect(screen.getByPlaceholderText(/role name/i)).toHaveValue("Viewer");

    const dialog = screen.getByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: /close/i }));

    await waitFor(() => {
      expect(screen.queryByPlaceholderText(/role name/i)).not.toBeInTheDocument();
    });
  });
});
