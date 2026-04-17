import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CreateDatabasePage from "@/pages/admin/CreateDatabasePage";
import EditDatabasePage from "@/pages/admin/EditDatabasePage";
import { databasesApi } from "@/api/databases";

vi.mock("@/api/databases", () => ({
  databasesApi: {
    testConnection: vi.fn(),
    testConnectionById: vi.fn(),
    createDatabase: vi.fn(),
    updateDatabase: vi.fn(),
    getDatabase: vi.fn(),
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
let currentParams: Record<string, string> = { id: "" };

vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useParams: () => currentParams,
  };
});

function renderPage(databaseId?: string) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  // Set the params for this render
  currentParams = { id: databaseId || "" };

  return render(
    <MemoryRouter initialEntries={[databaseId ? `/databases/${databaseId}` : "/databases/new"]}>
      <QueryClientProvider client={queryClient}>
        {databaseId ? <EditDatabasePage /> : <CreateDatabasePage />}
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

    expect(screen.getByRole("button", { name: /test connection/i })).not.toBeDisabled();
    expect(screen.getByRole("button", { name: /(show|hide) error details/i })).toBeInTheDocument();

    expect(screen.getByRole("button", { name: /^save$/i })).toBeDisabled();

    await user.click(screen.getByLabelText(/save without testing/i));
    expect(screen.getByRole("button", { name: /^save$/i })).toBeEnabled();
  });

  it("creates database and navigates to list on success", async () => {
    let resolveTestConnection: ((value: { success: boolean; latency_ms?: number; db_version?: string }) => void) | null = null;
    vi.mocked(databasesApi.testConnection).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveTestConnection = resolve;
        }),
    );

    const successfulTestResult = {
      success: true,
      latency_ms: 31,
      db_version: "PostgreSQL 15.4",
    };

    vi.mocked(databasesApi.createDatabase).mockResolvedValue({
      id: 10,
      database_name: "analytics",
      backend: "postgresql",
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

    expect(screen.getByRole("button", { name: /testing/i })).toBeDisabled();

    resolveTestConnection!(successfulTestResult);

    await waitFor(() => {
      expect(screen.getByText(/connection successful/i)).toBeInTheDocument();
    });

    expect(screen.getAllByText(/postgresql 15.4/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/31ms/i)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() => {
      expect(databasesApi.createDatabase).toHaveBeenCalled();
    });

    expect(toastSuccessMock).toHaveBeenCalled();
    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/databases");
  });

  it("shows rate limit toast when test endpoint returns 429", async () => {
    vi.mocked(databasesApi.testConnection).mockRejectedValue(
      Object.assign(new Error("too many requests"), { status: 429 }),
    );

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
      expect(toastErrorMock).toHaveBeenCalledWith("Too many test attempts. Wait 60 seconds.");
    });
  });
});

describe("EditDatabasePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders edit mode when database ID is provided", async () => {
    const existingDb = {
      id: 10,
      database_name: "analytics",
      backend: "postgresql",
      sqlalchemy_uri: "postgresql://alice:secret@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
    };

    vi.mocked(databasesApi.getDatabase).mockResolvedValue(existingDb);

    renderPage("10");

    await waitFor(() => {
      expect(screen.getByText(/edit database connection/i)).toBeInTheDocument();
    });

    expect(databasesApi.getDatabase).toHaveBeenCalledWith(10);
  });

  it("updates database and navigates on success", async () => {
    const existingDb = {
      id: 10,
      database_name: "analytics",
      backend: "postgresql",
      sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
    };

    vi.mocked(databasesApi.getDatabase).mockResolvedValue(existingDb);
    vi.mocked(databasesApi.updateDatabase).mockResolvedValue(existingDb);
    vi.mocked(databasesApi.testConnectionById).mockResolvedValue({
      success: true,
      latency_ms: 25,
      db_version: "PostgreSQL 15.4",
    });

    renderPage("10");

    await waitFor(() => {
      expect(screen.getByText(/edit database connection/i)).toBeInTheDocument();
    });

    const user = userEvent.setup();

    // Test connection without changing password
    await user.click(screen.getByRole("button", { name: /^next$/i }));
    await user.click(screen.getByRole("button", { name: /^next$/i }));
    await user.click(screen.getByRole("button", { name: /test connection/i }));

    await waitFor(() => {
      expect(databasesApi.testConnectionById).toHaveBeenCalledWith(10);
    });

    // Save without changing password
    await user.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() => {
      expect(databasesApi.updateDatabase).toHaveBeenCalled();
    });

    expect(toastSuccessMock).toHaveBeenCalledWith("Database updated successfully");
    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/databases");
  });

  it("disables database type selection in edit mode", async () => {
    const existingDb = {
      id: 10,
      database_name: "analytics",
      backend: "postgresql",
      sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
    };

    vi.mocked(databasesApi.getDatabase).mockResolvedValue(existingDb);

    renderPage("10");

    await waitFor(() => {
      expect(screen.getByText(/edit database connection/i)).toBeInTheDocument();
    });

    const postgresButton = screen.getByLabelText(/postgresql/i);
    expect(postgresButton).toBeDisabled();
  });

  it("shows password placeholder for unchanged password in edit mode", async () => {
    const existingDb = {
      id: 10,
      database_name: "analytics",
      backend: "postgresql",
      sqlalchemy_uri: "postgresql://alice:***@localhost:5432/analytics",
      allow_dml: false,
      expose_in_sqllab: true,
      allow_run_async: false,
      allow_file_upload: false,
    };

    vi.mocked(databasesApi.getDatabase).mockResolvedValue(existingDb);

    renderPage("10");

    await waitFor(() => {
      expect(screen.getByText(/edit database connection/i)).toBeInTheDocument();
    });

    const passwordInput = screen.getByLabelText(/password/i) as HTMLInputElement;
    expect(passwordInput.placeholder).toContain("Leave blank to keep current password");
  });
});
