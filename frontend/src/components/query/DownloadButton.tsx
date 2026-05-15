import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useAuthStore } from "@/stores/authStore";

interface DownloadButtonProps {
  downloadUrl: string;
}

export function DownloadButton({ downloadUrl }: DownloadButtonProps) {
  const handleDownload = () => {
    const token = useAuthStore.getState().accessToken;
    const separator = downloadUrl.includes("?") ? "&" : "?";
    const url = token ? `${downloadUrl}${separator}token=${encodeURIComponent(token)}` : downloadUrl;
    window.open(url, "_blank");
  };

  return (
    <Alert className="bg-blue-50 border-blue-200">
      <AlertDescription className="text-blue-800 flex items-center justify-between">
        <span>Result set is too large for inline display.</span>
        <Button
          onClick={handleDownload}
          size="sm"
          variant="outline"
          className="ml-4 bg-white border-blue-300 text-blue-700 hover:bg-blue-100"
        >
          <Download className="h-4 w-4 mr-1" />
          Download Results
        </Button>
      </AlertDescription>
    </Alert>
  );
}
