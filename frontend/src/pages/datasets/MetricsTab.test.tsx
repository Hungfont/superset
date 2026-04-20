import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MetricsTab } from "@/pages/datasets/MetricsTab";
import { datasetsApi } from "@/api/datasets";
import type { SqlMetric } from "@/api/datasets";

vi.mock("@/api/datasets", () => ({
  datasetsApi: {
    getMetrics: vi.fn(),
    createMetric: vi.fn(),
    updateMetric: vi.fn(),
    deleteMetric: vi.fn(),
    bulkUpdateMetrics: vi.fn(),
  },
}));

const toastSuccessMock = vi.fn();
const toastErrorMock = vi.fn();
const toastWarningMock = vi.fn();

vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({
    success: toastSuccessMock,
    error: toastErrorMock,
    warning: toastWarningMock,
  }),
}));

const mockMetrics: SqlMetric[] = [
  {
    id: 1,
    metric_name: "total_count",
    verbose_name: "Total Count",
    metric_type: "SUM",
    expression: "COUNT(*)",
    d3_format: ",.0f",
    created_on: "2024-01-01T00:00:00Z",
  },
  {
    id: 2,
    metric_name: "avg_sales",
    verbose_name: "Average Sales",
    metric_type: "AVG",
    expression: "AVG(sales)",
    d3_format: ",.2f",
    warning_text: "Excludes returns",
    created_on: "2024-01-01T00:00:00Z",
  },
];

const renderComponent = (metrics?: SqlMetric[]) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  if (metrics !== undefined) {
    vi.mocked(datasetsApi.getMetrics).mockResolvedValue(metrics);
  }

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <MetricsTab datasetId={1} initialMetrics={metrics} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
};

describe("MetricsTab", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("rendering", () => {
    it("renders metrics table with data", async () => {
      renderComponent(mockMetrics);

      expect(await screen.findByText("total_count")).toBeInTheDocument();
      expect(screen.getByText("avg_sales")).toBeInTheDocument();
      expect(screen.getAllByText("SUM")[0]).toBeInTheDocument();
      expect(screen.getAllByText("AVG")[0]).toBeInTheDocument();
    });

    it("shows empty state when no metrics", async () => {
      renderComponent([]);

      expect(
        await screen.findByText("No metrics defined for this dataset"),
      ).toBeInTheDocument();
    });

    it("shows add metric button", async () => {
      renderComponent(mockMetrics);

      expect(await screen.findByRole("button", { name: /add metric/i })).toBeInTheDocument();
    });

    it("shows metric count badge", async () => {
      renderComponent(mockMetrics);

      expect(await screen.findByText("2")).toBeInTheDocument();
    });
  });

  describe("create metric", () => {
    it("opens dialog when add metric button is clicked", async () => {
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByRole("heading", { name: /create metric/i })).toBeInTheDocument();
    });

    it("creates metric successfully", async () => {
      vi.mocked(datasetsApi.createMetric).mockResolvedValue({ id: 3 });
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.type(screen.getByLabelText(/metric name/i), "new_metric");
      await user.selectOptions(screen.getByLabelText(/metric type/i), "SUM");
      await user.type(screen.getByLabelText(/expression/i), "COUNT(*)");

      await user.click(screen.getByRole("button", { name: /create metric/i }));

      await waitFor(() => {
        expect(datasetsApi.createMetric).toHaveBeenCalledWith(1, {
          metric_name: "new_metric",
          metric_type: "SUM",
          expression: "COUNT(*)",
          verbose_name: "",
          d3_format: "",
          warning_text: "",
          is_restricted: false,
          certified_by: "",
          certification_details: "",
        });
      });

      expect(toastSuccessMock).toHaveBeenCalledWith("Metric created successfully");
    });

    it("shows error when create fails with duplicate name", async () => {
      vi.mocked(datasetsApi.createMetric).mockRejectedValue(
        new Error("metric name already exists"),
      );
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.type(screen.getByLabelText(/metric name/i), "total_count");
      await user.selectOptions(screen.getByLabelText(/metric type/i), "SUM");
      await user.type(screen.getByLabelText(/expression/i), "COUNT(*)");

      await user.click(screen.getByRole("button", { name: /create metric/i }));

      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalled();
      });
    });
  });

  describe("edit metric", () => {
    it("opens edit dialog when edit button is clicked", async () => {
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      const actionButtons = await screen.findAllByRole("button");
      const editButton = actionButtons.find((btn) => 
        btn.querySelector("svg.lucide-pencil")
      );
      await user.click(editButton!);

      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByRole("heading", { name: /edit metric/i })).toBeInTheDocument();
      expect(screen.getByDisplayValue("total_count")).toBeInTheDocument();
    });

    it("updates metric successfully", async () => {
      vi.mocked(datasetsApi.updateMetric).mockResolvedValue({ id: 1 });
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      const editButtons = await screen.findAllByRole("button", { name: /pencil/i });
      await user.click(editButtons[0]);

      await user.clear(screen.getByLabelText(/verbose name/i));
      await user.type(screen.getByLabelText(/verbose name/i), "Updated Total Count");

      await user.click(screen.getByRole("button", { name: /save changes/i }));

      await waitFor(() => {
        expect(datasetsApi.updateMetric).toHaveBeenCalledWith(1, 1, {
          metric_name: "total_count",
          verbose_name: "Updated Total Count",
          metric_type: "SUM",
          expression: "COUNT(*)",
          d3_format: ",.0f",
        });
      });

      expect(toastSuccessMock).toHaveBeenCalledWith("Metric updated successfully");
    });
  });

  describe("delete metric", () => {
    it("opens delete confirmation dialog", async () => {
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      const actionButtons = await screen.findAllByRole("button");
      const deleteButton = actionButtons.find((btn) => 
        btn.querySelector("svg.lucide-trash2")
      );
      expect(deleteButton).toBeInTheDocument();
      await user.click(deleteButton!);

      expect(screen.getByRole("alertdialog")).toBeInTheDocument();
      expect(
        screen.getByText(/delete the metric "total_count"/i),
      ).toBeInTheDocument();
    });

    it("deletes metric successfully", async () => {
      vi.mocked(datasetsApi.deleteMetric).mockResolvedValue({ warnings: [] });
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      const actionButtons = await screen.findAllByRole("button");
      const deleteButton = actionButtons.find((btn) => 
        btn.querySelector("svg.lucide-trash2")
      );
      await user.click(deleteButton!);

      await user.click(screen.getByRole("button", { name: /^delete$/i }));

      await waitFor(() => {
        expect(datasetsApi.deleteMetric).toHaveBeenCalledWith(1, 1);
      });

      expect(toastSuccessMock).toHaveBeenCalledWith("Metric deleted successfully");
    });

    it("shows warning when metric is referenced in charts", async () => {
      vi.mocked(datasetsApi.deleteMetric).mockResolvedValue({
        warnings: ["Chart: Sales Dashboard", "Chart: Revenue Report"],
      });
      renderComponent(mockMetrics);
      const user = userEvent.setup();

      const actionButtons = await screen.findAllByRole("button");
      const deleteButton = actionButtons.find((btn) => 
        btn.querySelector("svg.lucide-trash2")
      );
      await user.click(deleteButton!);

      await user.click(screen.getByRole("button", { name: /^delete$/i }));

      await waitFor(() => {
        expect(toastWarningMock).toHaveBeenCalledWith(
          "Metric deleted. 2 charts may be affected.",
        );
      });
    });
  });

  describe("metric type suggestions", () => {
    it("suggests expression when metric type is selected", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.selectOptions(screen.getByLabelText(/metric type/i), "SUM");

      const expressionField = screen.getByLabelText(/expression/i);
      expect(expressionField).toHaveValue("SUM(column_name)");
    });

    it("suggests COUNT(*) for COUNT type", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.selectOptions(screen.getByLabelText(/metric type/i), "COUNT");

      const expressionField = screen.getByLabelText(/expression/i);
      expect(expressionField).toHaveValue("COUNT(*)");
    });
  });

  describe("d3 format preview", () => {
    it("shows format preview when format is selected", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.selectOptions(screen.getByLabelText(/format/i), ",.0f");

      expect(screen.getByText(/preview: 1,234/i)).toBeInTheDocument();
    });
  });

  describe("validation", () => {
    it("requires metric name", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.selectOptions(screen.getByLabelText(/metric type/i), "SUM");
      await user.type(screen.getByLabelText(/expression/i), "COUNT(*)");

      await user.click(screen.getByRole("button", { name: /create metric/i }));

      expect(screen.getByText(/metric name must be at least 3 characters/i)).toBeInTheDocument();
    });

    it("requires expression", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.type(screen.getByLabelText(/metric name/i), "test_count");

      await user.click(screen.getByRole("button", { name: /create metric/i }));

      expect(screen.getByText(/expression is required/i)).toBeInTheDocument();
    });

    it("validates metric name format", async () => {
      renderComponent([]);
      const user = userEvent.setup();

      await user.click(await screen.findByRole("button", { name: /add metric/i }));

      await user.type(screen.getByLabelText(/metric name/i), "Invalid Name");

      expect(
        screen.getByText(/must be snake_case/i),
      ).toBeInTheDocument();
    });
  });
});
