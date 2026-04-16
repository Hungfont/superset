import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CreateDatabasePage from "@/pages/admin/CreateDatabasePage";
import { databasesApi } from "@/api/databases";

vi.mock("@/api/databases", () => ({
  databasesApi: {
    testConnection: vi.fn(),
    createDatabase: vi.fn(),
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
        <CreateDatabasePage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("CreateDatabasePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("blocks moving to step 2 when db type is not selected", async () => {
    renderPage();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(screen.getByText(/select a database type/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /select db type/i })).toHaveAttribute("aria-current", "step");
  });

  it("only allows reviewing completed steps", async () => {
    renderPage();

    expect(screen.getByRole("button", { name: /select db type/i })).toHaveAttribute("aria-current", "step");
    expect(screen.queryByRole("button", { name: /configure connection/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /test & save/i })).not.toBeInTheDocument();
  });

  it("requires test success before save", async () => {
    vi.mocked(databasesApi.testConnection).mockResolvedValue({
      success: false,
      error: "authentication failed",
    });

    renderPage();
    const user = userEvent.setup();

    await user.click(screen.getByLabelText(/postgresql/i));
    await user.click(screen.getByRole("button", { name: /^next$/i }));

    await user.type(screen.getByLabelText(/database name/i), "analytics");
    await user.type(screen.getByLabelText(/^host$/i), "localhost");
    await user.type(screen.getByLabelText(/database$/i), "analytics");
    await user.type(screen.getByLabelText(/username/i), "alice");
    await user.type(screen.getByLabelText(/password/i), "secret");

    await user.click(screen.getByRole("button", { name: /^next$/i }));
    await user.click(screen.getByRole("button", { name: /test connection/i }));

    await waitFor(() => {
      expect(databasesApi.testConnection).toHaveBeenCalled();
    });

    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();

    await user.click(screen.getByLabelText(/save without testing/i));
    expect(screen.getByRole("button", { name: /^save$/i })).toBeEnabled();
  });

  it("creates database and navigates to list on success", async () => {
    vi.mocked(databasesApi.testConnection).mockResolvedValue({
      success: true,
      latency_ms: 31,
      db_version: "PostgreSQL 15.4",
    });

    vi.mocked(databasesApi.createDatabase).mockResolvedValue({
      id: 10,
      database_name: "analytics",
      sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
    });

    renderPage();
    const user = userEvent.setup();

    await user.click(screen.getByLabelText(/postgresql/i));
    await user.click(screen.getByRole("button", { name: /^next$/i }));

    await user.type(screen.getByLabelText(/database name/i), "analytics");
    await user.type(screen.getByLabelText(/^host$/i), "localhost");
    await user.type(screen.getByLabelText(/database$/i), "analytics");
    await user.type(screen.getByLabelText(/username/i), "alice");
    await user.type(screen.getByLabelText(/password/i), "secret");

    await user.click(screen.getByRole("button", { name: /^next$/i }));
    await user.click(screen.getByRole("button", { name: /test connection/i }));

    await waitFor(() => {
      expect(screen.getByText(/connection successful/i)).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() => {
      expect(databasesApi.createDatabase).toHaveBeenCalled();
    });

    expect(toastSuccessMock).toHaveBeenCalled();
    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/databases");
  });
});
