import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import UsersPage from "@/pages/admin/UsersPage";
import { usersApi } from "@/api/users";
import { rolesApi } from "@/api/roles";

vi.mock("@/api/users", () => ({
  usersApi: {
    getUsers: vi.fn(),
    createUser: vi.fn(),
    updateUser: vi.fn(),
    deactivateUser: vi.fn(),
  },
}));

vi.mock("@/api/roles", () => ({
  rolesApi: {
    getRoles: vi.fn(),
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
        <UsersPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("UsersPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(rolesApi.getRoles).mockResolvedValue([
      { id: 1, name: "Admin", user_count: 1, permission_count: 10, built_in: true },
      { id: 3, name: "Gamma", user_count: 2, permission_count: 6, built_in: true },
    ]);

    vi.mocked(usersApi.getUsers).mockResolvedValue([
      {
        id: 7,
        first_name: "Alice",
        last_name: "Nguyen",
        username: "alice",
        email: "alice@example.com",
        active: true,
        login_count: 5,
        role_ids: [1, 3],
      },
    ]);
  });

  it("renders users table", async () => {
    renderPage();

    expect(await screen.findByText("User Management")).toBeInTheDocument();
    expect(await screen.findByText("alice")).toBeInTheDocument();
    expect(screen.getByText("alice@example.com")).toBeInTheDocument();
  });

  it("creates user from dialog", async () => {
    vi.mocked(usersApi.createUser).mockResolvedValue({
      id: 9,
      first_name: "Bob",
      last_name: "Tran",
      username: "bob",
      email: "bob@example.com",
      active: true,
      login_count: 0,
      role_ids: [1],
    });

    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /new user/i }));
    await user.type(screen.getByLabelText(/first name/i), "Bob");
    await user.type(screen.getByLabelText(/last name/i), "Tran");
    await user.type(screen.getByLabelText(/username/i), "bob");
    await user.type(screen.getByLabelText(/email/i), "bob@example.com");
    await user.type(screen.getByLabelText(/password/i), "StrongPass@123");
    await user.click(screen.getByRole("button", { name: /save user/i }));

    await waitFor(() => {
      expect(usersApi.createUser).toHaveBeenCalled();
    });
  });

  it("deactivates user after confirm", async () => {
    vi.mocked(usersApi.deactivateUser).mockResolvedValue(undefined);

    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /deactivate alice/i }));
    await user.click(screen.getByRole("button", { name: /deactivate/i }));

    await waitFor(() => {
      expect(usersApi.deactivateUser).toHaveBeenCalledWith(7);
    });
  });

  it("navigates to user roles page", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /manage roles for alice/i }));

    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/users/7");
  });
});
