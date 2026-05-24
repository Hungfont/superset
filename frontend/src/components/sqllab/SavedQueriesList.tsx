import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Search } from "lucide-react";
import { fetchSavedQueries, type SavedQueryResponse } from "@/api/sqllab";

interface SavedQueriesListProps {
  onLoad: (sq: SavedQueryResponse) => void;
  activeDbId: number | null;
}

export function SavedQueriesList({ onLoad, activeDbId }: SavedQueriesListProps) {
  const [search, setSearch] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["saved-queries", { q: search }],
    queryFn: () => fetchSavedQueries({ q: search || undefined, limit: 50 }),
  });

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  const items = data?.items ?? [];

  return (
    <div className="flex flex-col h-full">
      <div className="relative mb-3">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
        <Input
          placeholder="Filter by label..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-8 h-8 text-sm"
        />
      </div>
      <ScrollArea className="flex-1">
        {items.length === 0 && (
          <p className="text-sm text-muted-foreground text-center py-8">
            {search ? "No matching queries" : "No saved queries yet"}
          </p>
        )}
        {items.map((sq) => (
          <div key={sq.id} className="flex items-center justify-between py-2 border-b">
            <div className="min-w-0 flex-1 mr-2">
              <p className="font-medium text-sm truncate">{sq.label}</p>
              <div className="flex items-center gap-1 mt-0.5">
                <Badge variant="outline" className="text-xs px-1 py-0">
                  DB #{sq.db_id}
                  {sq.db_id === activeDbId ? " (current)" : ""}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  {new Date(sq.changed_on).toLocaleDateString()}
                </span>
              </div>
            </div>
            <Button variant="ghost" size="sm" onClick={() => onLoad(sq)}>
              Load
            </Button>
          </div>
        ))}
      </ScrollArea>
    </div>
  );
}
