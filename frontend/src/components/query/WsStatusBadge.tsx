import { Wifi, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { useWsStore, type WsConnectionStatus } from "@/stores/wsStore";

interface WsStatusBadgeProps {
  queryId: string;
}

export function WsStatusBadge({ queryId }: WsStatusBadgeProps) {
  const status: WsConnectionStatus = useWsStore(s => s.connections[queryId]?.status ?? "disconnected");

  if (!queryId) return null;

  if (status === "connected") {
    return (
      <Badge
        variant="outline"
        className="h-5 px-2 text-xs text-green-700 bg-green-50 border-green-200 gap-1"
      >
        <Wifi className="h-3 w-3" />
        WS Connected
      </Badge>
    );
  }

  if (status === "reconnecting" || status === "connecting") {
    return (
      <Badge
        variant="outline"
        className="h-5 px-2 text-xs text-amber-700 bg-amber-50 border-amber-200 gap-1"
      >
        <Loader2 className="h-3 w-3 animate-spin" />
        Reconnecting...
      </Badge>
    );
  }

  return (
    <Badge
      variant="outline"
      className="h-5 px-2 text-xs text-muted-foreground gap-1"
    >
      <Wifi className="h-3 w-3 opacity-50" />
      WS Disconnected
    </Badge>
  );
}
