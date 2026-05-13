import { Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface EstimateResult {
  supported: boolean;
  driver?: string;
  total_cost?: number;
  estimated_rows?: number;
  bytes_processed?: number;
  estimated_cost_usd?: number;
}

interface EstimatePopoverProps {
  estimate: EstimateResult | null;
  isLoading: boolean;
  onTrigger: () => void;
  isSupported: boolean;
}

export function EstimatePopover({ estimate, isLoading, onTrigger, isSupported }: EstimatePopoverProps) {
  if (!isSupported) return null;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2" onClick={onTrigger}>
          <Zap className="h-4 w-4" />
          Estimate Cost
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72">
        {isLoading ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Estimating Cost...</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-3/4" />
            </CardContent>
          </Card>
        ) : estimate ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">
                Query Estimate {estimate.driver && `(${estimate.driver})`}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              {estimate.supported ? (
                <>
                  {estimate.estimated_rows !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Estimated Rows</span>
                      <span className="font-medium">{estimate.estimated_rows.toLocaleString()}</span>
                    </div>
                  )}
                  {estimate.total_cost !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Planner Cost</span>
                      <span className="font-medium">{estimate.total_cost.toFixed(1)}</span>
                    </div>
                  )}
                  {estimate.bytes_processed !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Bytes Processed</span>
                      <span className="font-medium">{(estimate.bytes_processed / 1e9).toFixed(2)} GB</span>
                    </div>
                  )}
                  {estimate.estimated_cost_usd !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Estimated Cost</span>
                      <span className="font-medium">${estimate.estimated_cost_usd.toFixed(4)}</span>
                    </div>
                  )}
                  <Alert variant="default" className="mt-2 bg-muted/30">
                    <AlertDescription className="text-xs">
                      Estimate only. Actual execution may differ.
                    </AlertDescription>
                  </Alert>
                </>
              ) : (
                <p className="text-muted-foreground">Cost estimation not supported for this database.</p>
              )}
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="py-4 text-center text-sm text-muted-foreground">
              Click Estimate Cost to analyze your query.
            </CardContent>
          </Card>
        )}
      </PopoverContent>
    </Popover>
  );
}
