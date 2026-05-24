import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DataTable } from "@/components/ui/data-table";
import { Skeleton } from "@/components/ui/skeleton";
import { MoreHorizontal } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { fetchSavedQueries, type SavedQueryResponse } from "@/api/sqllab";
import type { ColumnDef } from "@tanstack/react-table";

export default function SavedQueriesPage() {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [search, setSearch] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["saved-queries", { q: search }],
    queryFn: () => fetchSavedQueries({ q: search || undefined, limit: 50 }),
  });

  const columns = useMemo<ColumnDef<SavedQueryResponse>[]>(() => [
    {
      accessorKey: "label",
      header: "Name",
      cell: ({ getValue }) => (
        <span className="font-medium">{getValue() as string}</span>
      ),
    },
    {
      accessorKey: "db_id",
      header: "Database",
      cell: ({ getValue }) => (
        <span className="text-muted-foreground text-sm">DB #{getValue() as number}</span>
      ),
    },
    {
      accessorKey: "schema",
      header: "Schema",
    },
    {
      accessorKey: "changed_on",
      header: "Modified",
      cell: ({ getValue }) => new Date(getValue() as string).toLocaleDateString(),
    },
    {
      accessorKey: "published",
      header: "Status",
      cell: ({ getValue }) =>
        getValue() ? (
          <Badge variant="secondary">Published</Badge>
        ) : (
          <Badge variant="outline">Private</Badge>
        ),
    },
    {
      id: "actions",
      cell: ({ row }) => {
        const sq = row.original;
        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => navigate(`/sqllab?load=${sq.id}`)}>
                Load in SQL Lab
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => toast("Edit — coming in SQL-005")}>
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => toast("Fork — coming in SQL-005")}>
                Fork
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => toast("Delete — coming in SQL-005")}>
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ], [navigate, toast]);

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Saved Queries</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Browse and load your saved queries
          </p>
        </div>
        <Button onClick={() => navigate("/sqllab")}>
          SQL Lab
        </Button>
      </div>

      <div className="mb-4">
        <Input
          placeholder="Search saved queries..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : (
        <DataTable data={data?.items ?? []} columns={columns} />
      )}
    </div>
  );
}
