import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CreateDatasetPage from "@/pages/datasets/CreateDatasetPage";
import { databasesApi } from "@/api/databases";
import { datasetsApi } from "@/api/datasets";

vi.mock("@/api/databases", () => ({
  databasesApi: {
    getDatabases: vi.fn(),
    getSchemas: vi.fn(),
    getTables: vi.fn(),
  },
}));

vi.mock("@/api/datasets", () => ({
  datasetsApi: {
    createDataset: vi.fn(),
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
    <MemoryRouter initialEntries={["/settings/datasets/new"]}>
      <QueryClientProvider client={queryClient}>
        <CreateDatasetPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("CreateDatasetPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(databasesApi.getDatabases).mockResolvedValue({
      items: [
        {
          id: 7,
          database_name: "analytics",
          backend: "postgresql",
          sqlalchemy_uri: "postgresql://superset:***@localhost:5432/analytics",
          allow_dml: false,
          expose_in_sqllab: true,
          allow_run_async: false,
          allow_file_upload: false,
        },
      ],
      pagination: { total: 1, page: 1, page_size: 50 },
    });

    vi.mocked(databasesApi.getSchemas).mockResolvedValue(["public"]);
    vi.mocked(databasesApi.getTables).mockResolvedValue({
      items: [{ name: "orders" }],
      pagination: { total: 1, page: 1, page_size: 50 },
    });
  });

  it("creates physical dataset and navigates to editor", async () => {
    vi.mocked(datasetsApi.createDataset).mockResolvedValue({
      id: 42,
      table_name: "orders",
      background_sync: true,
    });

    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /analytics/i }));
    await user.click(await screen.findByRole("button", { name: /public/i }));
    await user.click(await screen.findByRole("button", { name: /orders/i }));

    await user.click(screen.getByRole("button", { name: /create dataset/i }));

    await waitFor(() => {
      expect(vi.mocked(datasetsApi.createDataset).mock.calls[0]?.[0]).toEqual({
        database_id: 7,
        schema: "public",
        table_name: "orders",
      });
    });

    expect(toastSuccessMock).toHaveBeenCalled();
    expect(navigateMock).toHaveBeenCalledWith("/settings/datasets/42/edit");
  });

  it("shows error toast when create request fails", async () => {
    vi.mocked(datasetsApi.createDataset).mockRejectedValue(new Error("dataset already exists"));

    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: /analytics/i }));
    await user.click(await screen.findByRole("button", { name: /public/i }));
    await user.click(await screen.findByRole("button", { name: /orders/i }));

    await user.click(screen.getByRole("button", { name: /create dataset/i }));

    await waitFor(() => {
      expect(toastErrorMock).toHaveBeenCalledWith("dataset already exists");
    });
  });
});
