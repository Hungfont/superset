import { Download, FileText, FileSpreadsheet, Braces, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useToast } from "@/hooks/use-toast";
import { useMutation } from "@tanstack/react-query";
import { queriesApi } from "@/api/queries";

interface DownloadButtonProps {
  queryId: string;
  disabled?: boolean;
}

export function DownloadButton({ queryId, disabled }: DownloadButtonProps) {
  const { toast } = useToast();

  const downloadMutation = useMutation({
    mutationFn: (format: "csv" | "xlsx" | "json") => queriesApi.download(queryId, format),
    onMutate: () => {
      toast("Preparing download...", {
        description: "Your file is being generated.",
      });
    },
    onSuccess: () => {
      toast("Download complete");
    },
    onError: (error: Error) => {
      toast.error("Download failed", {
        description: error.message,
      });
    },
  });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" disabled={disabled || downloadMutation.isPending}>
          {downloadMutation.isPending ? (
            <Loader2 className="h-4 w-4 mr-1 animate-spin" />
          ) : (
            <Download className="h-4 w-4 mr-1" />
          )}
          Download
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => downloadMutation.mutate("csv")}>
          <FileText className="h-4 w-4 mr-2" />
          Download as CSV
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => downloadMutation.mutate("xlsx")}>
          <FileSpreadsheet className="h-4 w-4 mr-2" />
          Download as Excel
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => downloadMutation.mutate("json")}>
          <Braces className="h-4 w-4 mr-2" />
          Download as JSON
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
