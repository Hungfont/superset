import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Search, RefreshCw, Table, Eye, Copy, ChevronRight } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  getSchemaTables,
  expandTable as expandTableApi,
  collapseTable as collapseTableApi,
  clearSchemaState,
  type SchemaTableItem,
  type SchemaColumnItem,
} from "@/api/sqllab";

const COLUMN_TYPE_COLORS: Record<string, string> = {
  INT: "bg-blue-100 text-blue-800",
  INTEGER: "bg-blue-100 text-blue-800",
  BIGINT: "bg-blue-100 text-blue-800",
  SMALLINT: "bg-blue-100 text-blue-800",
  VARCHAR: "bg-green-100 text-green-800",
  TEXT: "bg-green-100 text-green-800",
  CHAR: "bg-green-100 text-green-800",
  BOOLEAN: "bg-yellow-100 text-yellow-800",
  BOOL: "bg-yellow-100 text-yellow-800",
  TIMESTAMP: "bg-orange-100 text-orange-800",
  TIMESTAMPTZ: "bg-orange-100 text-orange-800",
  DATE: "bg-orange-100 text-orange-800",
  FLOAT: "bg-purple-100 text-purple-800",
  DOUBLE: "bg-purple-100 text-purple-800",
  NUMERIC: "bg-purple-100 text-purple-800",
  DECIMAL: "bg-purple-100 text-purple-800",
  JSON: "bg-gray-100 text-gray-800",
  JSONB: "bg-gray-100 text-gray-800",
  UUID: "bg-pink-100 text-pink-800",
};

function getTypeBadgeClass(dataType: string): string {
  const upper = dataType.toUpperCase();
  return COLUMN_TYPE_COLORS[upper] ?? "bg-muted text-muted-foreground";
}

interface SchemaBrowserProps {
  tabId: number;
  currentSchema: string;
  onColumnClick: (columnName: string) => void;
}

export function SchemaBrowser({ tabId, currentSchema, onColumnClick }: SchemaBrowserProps) {
  const queryClient = useQueryClient();
  const [filterText, setFilterText] = useState("");
  const [expandedTables, setExpandedTables] = useState<Set<string>>(new Set());
  const [columnCache, setColumnCache] = useState<Map<string, SchemaColumnItem[]>>(new Map());
  const [selectedSchema, setSelectedSchema] = useState<string>(currentSchema);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ["schema-tables", tabId],
    queryFn: () => getSchemaTables(tabId),
    enabled: tabId > 0,
  });

  const expandMutation = useMutation({
    mutationFn: (tableName: string) => expandTableApi(tabId, tableName),
    onSuccess: (res) => {
      setColumnCache((prev) => new Map(prev).set(res.table_name, res.columns));
      setExpandedTables((prev) => new Set(prev).add(res.table_name));
    },
  });

  const collapseMutation = useMutation({
    mutationFn: (tableName: string) => collapseTableApi(tabId, tableName),
  });

  const handleTableToggle = useCallback(
    async (table: SchemaTableItem) => {
      const name = table.table_name;
      if (expandedTables.has(name)) {
        setExpandedTables((prev) => {
          const next = new Set(prev);
          next.delete(name);
          return next;
        });
        collapseMutation.mutate(name);
      } else if (columnCache.has(name)) {
        setExpandedTables((prev) => new Set(prev).add(name));
      } else {
        expandMutation.mutate(name);
      }
    },
    [expandedTables, columnCache, expandMutation, collapseMutation],
  );

  const handleRefresh = useCallback(() => {
    refetch();
  }, [refetch]);

  const handleSchemaChange = useCallback(
    async (newSchema: string) => {
      setSelectedSchema(newSchema);
      setExpandedTables(new Set());
      setColumnCache(new Map());
      await clearSchemaState(tabId);
      queryClient.invalidateQueries({ queryKey: ["schema-tables", tabId] });
    },
    [tabId, queryClient],
  );

  const schemas = data?.schemas ?? [];
  const tables = data?.tables ?? [];

  const filteredTables = filterText
    ? tables.filter((t) => t.table_name.toLowerCase().includes(filterText.toLowerCase()))
    : tables;

  const hasSchema = selectedSchema !== "";

  return (
    <div className="h-full border-r flex flex-col p-3 gap-2">
      <h3 className="text-sm font-semibold">Schema Browser</h3>

      <Select value={selectedSchema} onValueChange={handleSchemaChange}>
        <SelectTrigger className="h-8 text-xs">
          <SelectValue placeholder="Select schema" />
        </SelectTrigger>
        <SelectContent>
          {schemas.map((s) => (
            <SelectItem key={s} value={s}>
              {s}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="flex gap-1">
        <div className="relative flex-1">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
          <Input
            placeholder="Filter tables..."
            value={filterText}
            onChange={(e) => setFilterText(e.target.value)}
            className="pl-7 h-7 text-xs"
          />
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={handleRefresh}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>

      <ScrollArea className="flex-1">
        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-full" />
            ))}
          </div>
        ) : !hasSchema ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            Select a schema to browse tables
          </p>
        ) : filteredTables.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No tables found
          </p>
        ) : (
          <div className="space-y-0.5">
            {filteredTables.map((table) => (
              <Collapsible
                key={table.table_name}
                open={expandedTables.has(table.table_name)}
                onOpenChange={() => handleTableToggle(table)}
              >
                <CollapsibleTrigger className="flex items-center w-full rounded px-1 py-1 hover:bg-muted text-left text-xs">
                  <ChevronRight className="h-3 w-3 shrink-0 mr-1 transition-transform data-[state=open]:rotate-90" />
                  {table.table_type === "VIEW" ? (
                    <Eye className="h-3 w-3 shrink-0 mr-1 text-muted-foreground" />
                  ) : (
                    <Table className="h-3 w-3 shrink-0 mr-1 text-muted-foreground" />
                  )}
                  <span className="truncate flex-1">{table.table_name}</span>
                  {table.table_type === "VIEW" && (
                    <Badge variant="outline" className="ml-1 text-[10px] px-1 py-0 h-4">
                      VIEW
                    </Badge>
                  )}
                  <span className="text-[10px] text-muted-foreground ml-1">
                    {columnCache.get(table.table_name)?.length ?? "..."}
                  </span>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  {columnCache.get(table.table_name)?.map((col) => (
                    <div
                      key={col.name}
                      className="flex items-center pl-7 pr-1 py-0.5 hover:bg-muted cursor-pointer text-xs"
                      onClick={() => onColumnClick(col.name)}
                    >
                      <span className="truncate flex-1">{col.name}</span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Badge
                            variant="secondary"
                            className={`ml-1 text-[10px] px-1 py-0 h-4 font-normal ${getTypeBadgeClass(col.data_type)}`}
                          >
                            {col.data_type}
                          </Badge>
                        </TooltipTrigger>
                        <TooltipContent side="right">
                          <p>{col.data_type}{col.is_nullable ? " (nullable)" : ""}</p>
                        </TooltipContent>
                      </Tooltip>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-5 w-5 ml-0.5 shrink-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigator.clipboard.writeText(col.name);
                        }}
                      >
                        <Copy className="h-2.5 w-2.5" />
                      </Button>
                    </div>
                  ))}
                </CollapsibleContent>
              </Collapsible>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
