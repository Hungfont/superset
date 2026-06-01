import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Search, Plus, BarChart2, ShieldCheck, MoreHorizontal } from "lucide-react";
import { chartsApi, type ChartListItem, type ChartListParams } from "@/api/charts";
import { useAuthStore } from "@/stores/authStore";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const VIZ_TYPE_COLORS: Record<string, string> = {
  line: "bg-blue-100 text-blue-800",
  area: "bg-blue-100 text-blue-800",
  bar: "bg-green-100 text-green-800",
  column: "bg-green-100 text-green-800",
  pie: "bg-orange-100 text-orange-800",
  donut: "bg-orange-100 text-orange-800",
  table: "bg-gray-100 text-gray-800",
  big_number: "bg-purple-100 text-purple-800",
  big_number_total: "bg-purple-100 text-purple-800",
  map: "bg-teal-100 text-teal-800",
};

function vizTypeColor(vizType: string): string {
  return VIZ_TYPE_COLORS[vizType] ?? "bg-gray-100 text-gray-800";
}

function formatDate(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function ChartsPage() {
  const navigate = useNavigate();
  const currentUser = useAuthStore((s) => s.user);
  const [q, setQ] = useState("");
  const [vizType, setVizType] = useState("");
  const [owner, setOwner] = useState("");
  const [certified, setCertified] = useState(false);
  const [page, setPage] = useState(1);

  const filters: ChartListParams = {
    ...(q ? { q } : {}),
    ...(vizType && vizType !== "all" ? { viz_type: vizType } : {}),
    ...(owner === "mine" && currentUser ? { owner: currentUser.id } : {}),
    ...(certified ? { certified: true } : {}),
    page,
    page_size: 20,
  };

  const { data, isLoading } = useQuery({
    queryKey: ["charts", filters],
    queryFn: () => chartsApi.list(filters),
  });

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 0;

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Charts</h1>
        <Button onClick={() => navigate("/explore")}>
          <Plus className="mr-2 h-4 w-4" />
          Chart
        </Button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search charts..."
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
            className="pl-9"
            aria-label="Search charts"
          />
        </div>
        <Select
          value={vizType}
          onValueChange={(v) => {
            setVizType(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[160px]">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="bar">Bar</SelectItem>
            <SelectItem value="line">Line</SelectItem>
            <SelectItem value="pie">Pie</SelectItem>
            <SelectItem value="table">Table</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={owner}
          onValueChange={(v) => {
            setOwner(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-[130px]">
            <SelectValue placeholder="All Owners" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="mine">Mine</SelectItem>
          </SelectContent>
        </Select>
        <div className="flex items-center gap-2">
          <Switch
            id="certified-only"
            checked={certified}
            onCheckedChange={(v) => {
              setCertified(v);
              setPage(1);
            }}
          />
          <label
            htmlFor="certified-only"
            className="text-sm text-muted-foreground cursor-pointer"
          >
            Certified only
          </label>
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !data || data.items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <BarChart2 className="h-12 w-12 text-muted-foreground mb-4" />
          <p className="text-lg font-medium">No charts yet</p>
          <p className="text-sm text-muted-foreground mb-4">
            Create your first chart to get started.
          </p>
          <Button onClick={() => navigate("/explore")}>
            <Plus className="mr-2 h-4 w-4" />
            Create your first chart
          </Button>
        </div>
      ) : (
        <>
          <Table aria-label="Charts list">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[80px]">Thumbnail</TableHead>
                <TableHead aria-sort="descending">Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Dataset</TableHead>
                <TableHead>Dashboards</TableHead>
                <TableHead>Modified</TableHead>
                <TableHead>Certified</TableHead>
                <TableHead className="w-[60px]">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.items.map((chart) => (
                <ChartRow
                  key={chart.id}
                  chart={chart}
                  onEdit={() => navigate(`/explore?slice_id=${chart.id}`)}
                />
              ))}
            </TableBody>
          </Table>

          {totalPages > 1 && (
            <div className="flex items-center justify-between pt-4">
              <p className="text-sm text-muted-foreground">
                Showing {(page - 1) * 20 + 1}–
                {Math.min(page * 20, data.total)} of {data.total}
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ChartRow({
  chart,
  onEdit,
}: {
  chart: ChartListItem;
  onEdit: () => void;
}) {
  return (
    <TableRow>
      <TableCell>
        <div className="h-10 w-16 bg-muted rounded flex items-center justify-center overflow-hidden">
          <BarChart2 className="h-6 w-6 text-muted-foreground" />
        </div>
      </TableCell>
      <TableCell className="font-medium">{chart.slice_name}</TableCell>
      <TableCell>
        <Badge variant="secondary" className={vizTypeColor(chart.viz_type)}>
          {chart.viz_type}
        </Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {chart.datasource_name}
      </TableCell>
      <TableCell>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Badge variant="secondary" className="cursor-pointer">
                {chart.dashboard_count}
              </Badge>
            </TooltipTrigger>
            <TooltipContent>
              <p>
                Used in {chart.dashboard_count} dashboard
                {chart.dashboard_count !== 1 ? "s" : ""}
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </TableCell>
      <TableCell className="text-muted-foreground text-sm">
        {formatDate(chart.last_saved_at)}
      </TableCell>
      <TableCell>
        {chart.certified_by ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Avatar className="h-6 w-6">
                  <AvatarFallback>
                    <ShieldCheck className="h-4 w-4 text-green-600" />
                  </AvatarFallback>
                </Avatar>
              </TooltipTrigger>
              <TooltipContent>
                <p>Certified by {chart.certified_by}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : null}
      </TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem>Duplicate</DropdownMenuItem>
            <DropdownMenuItem>Add to Dashboard</DropdownMenuItem>
            <DropdownMenuItem className="text-destructive">
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}
