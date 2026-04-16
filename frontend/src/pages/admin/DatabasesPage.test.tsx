import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import DatabasesPage from "@/pages/admin/DatabasesPage";
import { databasesApi } from "@/api/databases";

vi.mock("@/api/databases", () => ({
  databasesApi: {
    getDatabases: vi.fn(),
    testConnectionById: vi.fn(),
    deleteDatabase: vi.fn(),
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
        <DatabasesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("DatabasesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(databasesApi.getDatabases).mockResolvedValue({
      items: [
        {
          id: 11,
          database_name: "analytics",
          backend: "postgresql",
          sqlalchemy_uri: "postgresql://superset:***@localhost:5432/analytics",
          allow_dml: false,
          expose_in_sqllab: true,
          allow_run_async: true,
          allow_file_upload: false,
          dataset_count: 3,
        },
      ],
      pagination: {
        total: 1,
        page: 1,
        page_size: 10,
      },
    });
    vi.mocked(databasesApi.testConnectionById).mockResolvedValue({
      success: true,
      latency_ms: 27,
      db_version: "PostgreSQL 15.4",
      driver: "postgresql",
    });
    vi.mocked(databasesApi.deleteDatabase).mockResolvedValue(undefined);
  });

  it("renders database rows", async () => {
    renderPage();

    expect(await screen.findByText("Database Connections")).toBeInTheDocument();
    expect(await screen.findByText("analytics")).toBeInTheDocument();
    expect(screen.getByText("postgresql")).toBeInTheDocument();
  });

  it("navigates to create page from CTA", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /connect a database/i }));

    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/databases/new");
  });

  it("tests connection from row action", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /actions for analytics/i }));
    await user.click(screen.getByRole("menuitem", { name: /test connection/i }));

    await waitFor(() => {
      expect(databasesApi.testConnectionById).toHaveBeenCalled();
    });
    expect(vi.mocked(databasesApi.testConnectionById).mock.calls[0][0]).toBe(11);
  });

  it("deletes database from dialog action", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /actions for analytics/i }));
    await user.click(screen.getByRole("menuitem", { name: /delete/i }));
    await user.click(screen.getByRole("button", { name: /delete database/i }));

    await waitFor(() => {
      expect(databasesApi.deleteDatabase).toHaveBeenCalled();
    });
    expect(vi.mocked(databasesApi.deleteDatabase).mock.calls[0][0]).toBe(11);
  });

  it("navigates to detail page on row click", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /open details for analytics/i }));

    expect(navigateMock).toHaveBeenCalledWith("/admin/settings/databases/11");
  });
});
