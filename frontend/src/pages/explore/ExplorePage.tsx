import { useState, useMemo } from "react";
import ReactECharts from "echarts-for-react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { queriesApi } from "@/api/queries";
import { databasesApi } from "@/api/databases";

type ChartType = "bar" | "line" | "pie" | "area";

interface ChartConfig {
  type: ChartType;
  xAxisColumn: string;
  yAxisColumn: string;
  title: string;
}

export default function ExplorePage() {
  const [databaseId, setDatabaseId] = useState<number | null>(null);
  const [sql, setSql] = useState("");
  const [chartConfig, setChartConfig] = useState<ChartConfig>({
    type: "bar",
    xAxisColumn: "",
    yAxisColumn: "",
    title: "Chart",
  });
  const [error, setError] = useState<string | null>(null);

  const { data: databasesData, isLoading: databasesLoading } = useQuery({
    queryKey: ["databases"],
    queryFn: () => databasesApi.getDatabases({}),
  });

  const executeMutation = useMutation({
    mutationFn: queriesApi.execute,
    onSuccess: (data) => {
      setError(null);
      if (data.columns.length >= 2) {
        setChartConfig((prev) => ({
          ...prev,
          xAxisColumn: data.columns[0].name,
          yAxisColumn: data.columns[1].name,
        }));
      }
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleRunChart = () => {
    if (!databaseId || !sql) return;
    executeMutation.mutate({ database_id: databaseId, sql });
  };

  const chartData = executeMutation.data;
  const isLoading = executeMutation.isPending;

  const chartOption = useMemo(() => {
    if (!chartData || chartData.data.length === 0) {
      return {};
    }

    const xAxisData = chartData.data.map(
      (row) => row[chartConfig.xAxisColumn]
    );
    const yAxisData = chartData.data.map(
      (row) => row[chartConfig.yAxisColumn]
    );

    const baseOptions = {
      title: {
        text: chartConfig.title,
      },
      tooltip: {
        trigger: "axis",
      },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "3%",
        containLabel: true,
      },
    };

    switch (chartConfig.type) {
      case "bar":
        return {
          ...baseOptions,
          xAxis: { type: "category" as const, data: xAxisData },
          yAxis: { type: "value" as const },
          series: [
            {
              data: yAxisData,
              type: "bar",
              itemStyle: { color: "#3b82f6" },
            },
          ],
        };

      case "line":
        return {
          ...baseOptions,
          xAxis: { type: "category" as const, data: xAxisData },
          yAxis: { type: "value" as const },
          series: [
            {
              data: yAxisData,
              type: "line",
              smooth: true,
              itemStyle: { color: "#3b82f6" },
              areaStyle: { color: "rgba(59, 130, 246, 0.2)" },
            },
          ],
        };

      case "area":
        return {
          ...baseOptions,
          xAxis: { type: "category" as const, data: xAxisData },
          yAxis: { type: "value" as const },
          series: [
            {
              data: yAxisData,
              type: "line",
              smooth: true,
              itemStyle: { color: "#8b5cf6" },
              areaStyle: { color: "rgba(139, 92, 246, 0.3)" },
            },
          ],
        };

      case "pie":
        return {
          ...baseOptions,
          series: [
            {
              data: chartData.data.map((row, index) => ({
                name: String(row[chartConfig.xAxisColumn] || `Item ${index}`),
                value: Number(row[chartConfig.yAxisColumn]) || 0,
              })),
              type: "pie",
              radius: "60%",
              itemStyle: {
                color: (_: unknown, index: number) => {
                  const colors = [
                    "#3b82f6",
                    "#8b5cf6",
                    "#ec4899",
                    "#f59e0b",
                    "#10b981",
                  ];
                  return colors[index % colors.length];
                },
              },
            },
          ],
        };

      default:
        return {};
    }
  }, [chartData, chartConfig]);

  const columns = chartData?.columns || [];

  return (
    <div className="container mx-auto py-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Explore</h1>
        <Select
          onValueChange={(value) => setDatabaseId(parseInt(value, 10))}
          value={databaseId?.toString() || ""}
        >
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="Select database" />
          </SelectTrigger>
          <SelectContent>
            {databasesLoading ? (
              <SelectItem value="loading" disabled>
                Loading...
              </SelectItem>
            ) : (
              databasesData?.items?.map((db) => (
                <SelectItem key={db.id} value={db.id.toString()}>
                  {db.database_name}
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="text-lg">Query</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <textarea
              value={sql}
              onChange={(e) => setSql(e.target.value)}
              placeholder="SELECT category, COUNT(*) as count FROM sales GROUP BY category"
              className="w-full h-32 p-3 font-mono text-sm bg-muted/30 border rounded-md resize-none"
              disabled={isLoading}
            />

            <Button
              onClick={handleRunChart}
              disabled={!databaseId || !sql || isLoading}
              className="w-full"
            >
              {isLoading ? "Running..." : "Run Chart"}
            </Button>

            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-lg">Chart</CardTitle>
              <div className="flex gap-2">
                <Select
                  value={chartConfig.type}
                  onValueChange={(value) =>
                    setChartConfig((prev) => ({
                      ...prev,
                      type: value as ChartType,
                    }))
                  }
                >
                  <SelectTrigger className="w-[100px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="bar">Bar</SelectItem>
                    <SelectItem value="line">Line</SelectItem>
                    <SelectItem value="area">Area</SelectItem>
                    <SelectItem value="pie">Pie</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {columns.length >= 2 && (
              <div className="flex gap-4 mb-4">
                <div className="flex-1">
                  <label className="text-sm text-muted-foreground block mb-1">
                    X-Axis (Category)
                  </label>
                  <Select
                    value={chartConfig.xAxisColumn}
                    onValueChange={(value) =>
                      setChartConfig((prev) => ({
                        ...prev,
                        xAxisColumn: value,
                      }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select column" />
                    </SelectTrigger>
                    <SelectContent>
                      {columns.map((col) => (
                        <SelectItem key={col.name} value={col.name}>
                          {col.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1">
                  <label className="text-sm text-muted-foreground block mb-1">
                    Y-Axis (Value)
                  </label>
                  <Select
                    value={chartConfig.yAxisColumn}
                    onValueChange={(value) =>
                      setChartConfig((prev) => ({
                        ...prev,
                        yAxisColumn: value,
                      }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select column" />
                    </SelectTrigger>
                    <SelectContent>
                      {columns.map((col) => (
                        <SelectItem key={col.name} value={col.name}>
                          {col.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}

            {isLoading ? (
              <Skeleton className="h-[400px] w-full" />
            ) : chartData && chartData.data.length > 0 ? (
              <ReactECharts
                option={chartOption}
                style={{ height: "400px", width: "100%" }}
                opts={{ renderer: "canvas" }}
              />
            ) : (
              <div className="h-[400px] flex items-center justify-center text-muted-foreground">
                Run a query to generate a chart
              </div>
            )}

            {chartData && (
              <div className="mt-4 text-sm text-muted-foreground">
                <Badge variant="outline" className="mr-2">
                  {chartData.data.length} rows
                </Badge>
                {chartData.from_cache && (
                  <Badge variant="outline" className="text-green-600">
                    Cached
                  </Badge>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {chartData && chartData.data.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Data Preview</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-auto max-h-[300px]">
              <table className="w-full border-collapse text-sm">
                <thead className="sticky top-0 bg-muted/50">
                  <tr>
                    {columns.map((col) => (
                      <th
                        key={col.name}
                        className="px-3 py-2 text-left font-medium border-b"
                      >
                        {col.name}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {chartData.data.slice(0, 10).map((row, index) => (
                    <tr key={index} className="border-b hover:bg-muted/30">
                      {columns.map((col) => (
                        <td key={col.name} className="px-3 py-2">
                          {String(row[col.name] ?? "")}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
              {chartData.data.length > 10 && (
                <div className="text-center py-2 text-muted-foreground text-sm">
                  ...and {chartData.data.length - 10} more rows
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}