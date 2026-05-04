import { Play, RefreshCw, Loader2, ShieldAlert, Clock, StopCircle, CloudLightning } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface CacheBadgeProps {
  fromCache: boolean;
  durationMs?: number;
  cachedAt?: string;
  ttlSeconds?: number;
  onForceRefresh: () => void;
}

export function CacheBadge({
  fromCache,
  durationMs,
  cachedAt,
  ttlSeconds,
  onForceRefresh,
}: CacheBadgeProps) {
  if (fromCache && durationMs !== undefined) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-green-600 bg-green-50 hover:bg-green-100"
              onClick={onForceRefresh}
            >
              <RefreshCw className="mr-1 h-3 w-3" />
              <span>Cached ({durationMs}ms)</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <div className="text-xs space-y-1">
              <p>Results served from cache</p>
              {cachedAt && <p>Cached at {cachedAt}</p>}
              {ttlSeconds && <p>TTL: {ttlSeconds}s</p>}
              <p className="font-medium">Click to force refresh</p>
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  if (!fromCache && durationMs !== undefined) {
    return (
      <Badge
        variant="outline"
        className="h-6 text-muted-foreground bg-muted/30"
      >
        <Clock className="mr-1 h-3 w-3" />
        Live ({durationMs}ms)
      </Badge>
    );
  }

  return null;
}

interface RLSBadgeProps {
  executedSql: string;
  originalSql: string;
}

export function RLSBadge({ executedSql, originalSql }: RLSBadgeProps) {
  const rlsApplied = executedSql !== originalSql;

  if (!rlsApplied) {
    return null;
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <Badge variant="outline" className="h-6 text-orange-600 bg-orange-50 border-orange-200 cursor-help">
            <ShieldAlert className="mr-1 h-3 w-3" />
            RLS Active
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p className="text-xs">
            Row-level security filters were applied to this query
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

interface QueryStatusBadgeProps {
  status: "idle" | "running" | "success" | "error";
}

export function QueryStatusBadge({ status }: QueryStatusBadgeProps) {
  if (status === "idle") {
    return null;
  }

  if (status === "running") {
    return (
      <Badge
        variant="outline"
        className="h-6 text-amber-600 bg-amber-50 border-amber-200"
      >
        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        Running...
      </Badge>
    );
  }

  if (status === "success") {
    return (
      <Badge
        variant="outline"
        className="h-6 text-green-600 bg-green-50 border-green-200"
      >
        Done
      </Badge>
    );
  }

  if (status === "error") {
    return (
      <Badge
        variant="outline"
        className="h-6 text-red-600 bg-red-50 border-red-200"
      >
        Failed
      </Badge>
    );
  }

  return null;
}

interface RunButtonProps {
  onClick: () => void;
  disabled: boolean;
  isRunning: boolean;
}

export function RunButton({ onClick, disabled, isRunning }: RunButtonProps) {
  if (isRunning) {
    return (
      <Button disabled size="sm" className="gap-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        Running...
      </Button>
    );
  }

  return (
    <Button onClick={onClick} disabled={disabled} size="sm" className="gap-2">
      <Play className="h-4 w-4" />
      Run
    </Button>
  );
}

export interface AsyncStatusProps {
  status: "pending" | "queued" | "running" | "done" | "failed" | "stopped";
}

export function AsyncStatusBadge({ status }: AsyncStatusProps) {
  if (status === "pending" || status === "queued") {
    return (
      <Badge variant="outline" className="h-6 text-muted-foreground bg-muted/30">
        <Clock className="mr-1 h-3 w-3" />
        Queued
      </Badge>
    );
  }

  if (status === "running") {
    return (
      <Badge variant="outline" className="h-6 text-amber-600 bg-amber-50 border-amber-200">
        <Loader2 className="mr-1 h-3 w-3 animate-spin" />
        Running...
      </Badge>
    );
  }

  if (status === "done") {
    return (
      <Badge variant="outline" className="h-6 text-green-600 bg-green-50 border-green-200">
        Done
      </Badge>
    );
  }

  if (status === "failed" || status === "stopped") {
    return (
      <Badge variant="outline" className="h-6 text-red-600 bg-red-50 border-red-200">
        {status === "stopped" ? "Cancelled" : "Failed"}
      </Badge>
    );
  }

  return null;
}

interface RunAsyncButtonProps {
  onClick: () => void;
  disabled: boolean;
  isRunning: boolean;
  isQueued: boolean;
}

export function RunAsyncButton({ onClick, disabled, isRunning, isQueued }: RunAsyncButtonProps) {
  if (isRunning || isQueued) {
    return (
      <Button disabled size="sm" variant="outline" className="gap-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        Queued...
      </Button>
    );
  }

  return (
    <Button onClick={onClick} disabled={disabled} size="sm" variant="outline" className="gap-2">
      <CloudLightning className="h-4 w-4" />
      Run Async
    </Button>
  );
}

interface CancelButtonProps {
  onClick: () => void;
  disabled: boolean;
}

export function CancelButton({ onClick, disabled }: CancelButtonProps) {
  return (
    <Button onClick={onClick} disabled={disabled} size="sm" variant="destructive" className="gap-2">
      <StopCircle className="h-4 w-4" />
      Cancel
    </Button>
  );
}

interface AsyncProgressBarProps {
  status: "pending" | "queued" | "running" | "done" | "failed" | "stopped";
}

export function AsyncProgressBar({ status }: AsyncProgressBarProps) {
  if (status !== "running" && status !== "queued") {
    return null;
  }

  return (
    <div className="w-full space-y-1">
      <Progress value={status === "running" ? 50 : 20} className="h-1" />
      <div className="flex justify-between text-xs text-muted-foreground">
        <span>{status === "running" ? "Processing query..." : "Waiting in queue..."}</span>
      </div>
    </div>
  );
}

interface QueueBadgeProps {
  queue: string;
}

export function QueueBadge({ queue }: QueueBadgeProps) {
  const queueConfig: Record<string, { label: string; className: string }> = {
    critical: { label: "Priority", className: "text-purple-600 bg-purple-50 border-purple-200" },
    default: { label: "Standard", className: "text-blue-600 bg-blue-50 border-blue-200" },
    low: { label: "Background", className: "text-muted-foreground bg-muted/30 border-gray-200" },
  };

  const config = queueConfig[queue] || queueConfig.default;

  return (
    <Badge variant="outline" className={`h-5 text-xs ${config.className}`}>
      {config.label}
    </Badge>
  );
}