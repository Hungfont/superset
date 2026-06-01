import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import SaveChartDialog from "@/components/Explore/SaveChartDialog";
import { useExploreStore } from "@/stores/exploreStore";

vi.mock("@/api/charts", () => ({
  chartsApi: {
    create: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

import { toast } from "sonner";
import { chartsApi } from "@/api/charts";

const mockToastError = toast.error as ReturnType<typeof vi.fn>;
const mockToastSuccess = toast.success as ReturnType<typeof vi.fn>;

function renderDialog(open = true) {
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: 0 } } });
  const onOpenChange = vi.fn();
  const result = render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <SaveChartDialog open={open} onOpenChange={onOpenChange} />
      </MemoryRouter>
    </QueryClientProvider>
  );
  return { ...result, onOpenChange };
}

describe("SaveChartDialog", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
    useExploreStore.setState({
      datasourceId: "3",
      vizType: "bar",
      params: '{"metrics":["count"]}',
      queryContext: "",
    });
  });

  it("renders the dialog with title and badge when open", () => {
    renderDialog(true);
    expect(screen.getByText("Save Chart")).toBeInTheDocument();
    expect(screen.getByText("bar")).toBeInTheDocument();
    expect(
      screen.getByText("Give your chart a name to save it.")
    ).toBeInTheDocument();
  });

  it("renders form fields for chart name and description", () => {
    renderDialog(true);
    expect(screen.getByLabelText("Chart Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Description (optional)")).toBeInTheDocument();
  });

  it("has Save and Cancel buttons", () => {
    renderDialog(true);
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("shows validation error when name is empty", async () => {
    renderDialog(true);
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(
        screen.getByText("Chart name is required")
      ).toBeInTheDocument();
    });
  });

  it("calls chartsApi.create on valid submit", async () => {
    const mockCreate = vi
      .mocked(chartsApi.create)
      .mockResolvedValueOnce({
        id: 42,
        slice_name: "Revenue by Month",
        viz_type: "bar",
        datasource_id: "3",
        datasource_type: "table",
        datasource_name: "sales",
        params: '{"metrics":["count"]}',
        query_context: "",
        description: "",
        cache_timeout: 0,
        perm: "[sales](id:1)",
        schema_perm: "",
        certified_by: "",
        certification_details: "",
        last_saved_at: "2026-01-01T00:00:00Z",
        last_saved_by_fk: 10,
        created_on: "2026-01-01T00:00:00Z",
        changed_on: "2026-01-01T00:00:00Z",
      });

    renderDialog(true);

    await user.type(screen.getByLabelText("Chart Name"), "Revenue by Month");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(mockCreate).toHaveBeenCalled();
    });

    // TanStack useMutation passes (data, {client, meta, mutationKey}) — check first arg
    const callArg = mockCreate.mock.calls[0]?.[0] as Record<string, unknown> | undefined;
    expect(callArg).toBeDefined();
    expect(callArg?.slice_name).toBe("Revenue by Month");
    expect(callArg?.viz_type).toBe("bar");
    expect(callArg?.datasource_id).toBe("3");
    expect(callArg?.datasource_type).toBe("table");
    expect(callArg?.params).toBe('{"metrics":["count"]}');
  });

  it("shows validation error when name exceeds max length", async () => {
    renderDialog(true);

    const nameInput = screen.getByLabelText("Chart Name");
    // Type 256 chars to trigger max(255) validation
    const longName = "a".repeat(256);
    await user.type(nameInput, longName);
    await user.click(screen.getByRole("button", { name: "Save" }));

    // React Hook Form + Zod will render a max-length error via FormMessage
    await waitFor(() => {
      const formMessages = document.querySelectorAll("[id$='-form-item-message']");
      const hasError =
        formMessages.length > 0 &&
        Array.from(formMessages).some(
          (el) => el.textContent && el.textContent.trim().length > 0
        );
      expect(hasError).toBe(true);
    });
  });

  it("renders nothing when dialog is closed", () => {
    renderDialog(false);
    expect(screen.queryByText("Save Chart")).not.toBeInTheDocument();
  });

  it("shows error toast when no datasource is selected", async () => {
    useExploreStore.setState({ datasourceId: null });
    renderDialog(true);

    await user.type(screen.getByLabelText("Chart Name"), "Test Chart");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      // The toast appears via sonner — check for the toast region
      const toastEl = document.querySelector("[data-sonner-toast]");
      expect(toastEl).toBeTruthy();
    });
  });
});
