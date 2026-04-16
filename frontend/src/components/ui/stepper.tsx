import * as React from "react";
import { Check, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

export type StepStatus = "wait" | "process" | "finish" | "error";

export interface StepItem {
  title: React.ReactNode;
  description?: React.ReactNode;
  subTitle?: React.ReactNode;
  icon?: React.ReactNode;
  status?: StepStatus;
  disabled?: boolean;
}

export interface StepperProps {
  items: StepItem[];
  current?: number;
  direction?: "horizontal" | "vertical";
  size?: "default" | "small";
  onStepChange?: (index: number) => void;
  className?: string;
}

function getStepStatus(index: number, current: number, explicit?: StepStatus): StepStatus {
  if (explicit) {
    return explicit;
  }

  if (index < current) {
    return "finish";
  }

  if (index === current) {
    return "process";
  }

  return "wait";
}

function getIndicator(status: StepStatus, index: number, icon?: React.ReactNode): React.ReactNode {
  if (icon) {
    return icon;
  }

  if (status === "finish") {
    return <Check aria-hidden="true" className="size-3.5" />;
  }

  if (status === "error") {
    return <X aria-hidden="true" className="size-3.5" />;
  }

  return index + 1;
}

const INDICATOR_STYLE: Record<StepStatus, string> = {
  wait: "border-muted-foreground/40 text-muted-foreground bg-background",
  process: "border-primary text-primary bg-primary/10",
  finish: "border-primary bg-primary text-primary-foreground",
  error: "border-destructive bg-destructive text-destructive-foreground",
};

const TITLE_STYLE: Record<StepStatus, string> = {
  wait: "text-muted-foreground",
  process: "text-foreground",
  finish: "text-foreground",
  error: "text-destructive",
};

export function Stepper({
  items,
  current = 0,
  direction = "horizontal",
  size = "default",
  onStepChange,
  className,
}: StepperProps) {
  return (
    <ol
      className={cn(
        "w-full",
        direction === "horizontal" ? "flex items-start" : "flex flex-col",
        className,
      )}
    >
      {items.map((item, index) => {
        const status = getStepStatus(index, current, item.status);
        const isLast = index === items.length - 1;
        const isCurrent = index === current;
        const isClickable = typeof onStepChange === "function" && !item.disabled;

        const trigger = (
          <>
            <span
              className={cn(
                "inline-flex shrink-0 items-center justify-center rounded-full border font-medium transition-colors",
                size === "small" ? "size-6 text-xs" : "size-8 text-sm",
                INDICATOR_STYLE[status],
              )}
              aria-hidden="true"
            >
              {getIndicator(status, index, item.icon)}
            </span>

            <span className={cn("min-w-0", direction === "horizontal" ? "text-left" : "pt-0.5")}>
              <span
                className={cn(
                  "block truncate font-medium",
                  size === "small" ? "text-xs" : "text-sm",
                  TITLE_STYLE[status],
                )}
              >
                {item.title}
              </span>

              {item.subTitle ? (
                <span className="mt-0.5 block text-xs text-muted-foreground">{item.subTitle}</span>
              ) : null}

              {item.description ? (
                <span className="mt-0.5 block text-xs text-muted-foreground">{item.description}</span>
              ) : null}
            </span>
          </>
        );

        return (
          <li
            key={`step-${index}`}
            className={cn(
              direction === "horizontal" ? "flex min-w-0 flex-1 items-start" : "relative flex",
            )}
          >
            <div
              className={cn(
                "inline-flex min-w-0 items-start",
                direction === "horizontal" ? "gap-2" : "gap-3",
              )}
            >
              {isClickable ? (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => onStepChange(index)}
                  className={cn(
                    "h-auto min-h-0 p-0 hover:bg-transparent",
                    direction === "horizontal" ? "items-start gap-2" : "items-start gap-3",
                  )}
                  aria-current={isCurrent ? "step" : undefined}
                >
                  {trigger}
                </Button>
              ) : (
                <div
                  aria-current={isCurrent ? "step" : undefined}
                  className={cn(
                    "inline-flex items-start",
                    direction === "horizontal" ? "gap-2" : "gap-3",
                  )}
                >
                  {trigger}
                </div>
              )}
            </div>

            {!isLast ? (
              direction === "horizontal" ? (
                <div className="mx-3 mt-4 min-w-6 flex-1">
                  <Separator className={cn(status === "finish" ? "bg-primary" : "bg-border")} />
                </div>
              ) : (
                <Separator
                  orientation="vertical"
                  className={cn(
                    "absolute left-3 top-7 h-[calc(100%-1rem)]",
                    status === "finish" ? "bg-primary" : "bg-border",
                  )}
                />
              )
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}
