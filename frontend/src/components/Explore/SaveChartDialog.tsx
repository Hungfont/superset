import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { chartsApi } from "@/api/charts";
import { useExploreStore } from "@/stores/exploreStore";

const formSchema = z.object({
  slice_name: z.string().min(1, "Chart name is required").max(255),
  description: z.string().optional(),
});
type FormValues = z.infer<typeof formSchema>;

interface Props {
  open: boolean;
  onOpenChange: (o: boolean) => void;
}

export default function SaveChartDialog({ open, onOpenChange }: Props) {
  const [sp, setSp] = useSearchParams();
  const { datasourceId, vizType, params, queryContext, markClean } =
    useExploreStore();
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { slice_name: "", description: "" },
  });

  const mutation = useMutation({
    mutationFn: chartsApi.create,
    onSuccess: (chart) => {
      markClean();
      sp.set("slice_id", String(chart.id));
      setSp(sp, { replace: true });
      toast.success("Chart saved successfully");
      onOpenChange(false);
      form.reset();
    },
    onError: (e: Error) => toast.error(e.message || "Failed to save chart"),
  });

  const onSubmit = (v: FormValues) => {
    if (!datasourceId) {
      toast.error("No datasource selected");
      return;
    }
    mutation.mutate({
      slice_name: v.slice_name,
      viz_type: vizType,
      datasource_id: datasourceId,
      datasource_type: "table",
      params: params || undefined,
      query_context: queryContext || undefined,
      description: v.description || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Save Chart
            <Badge variant="secondary">{vizType}</Badge>
          </DialogTitle>
          <DialogDescription>
            Give your chart a name to save it.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="space-y-4"
          >
            <FormField
              control={form.control}
              name="slice_name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Chart Name</FormLabel>
                  <FormControl>
                    <Input placeholder="e.g. Revenue by Month" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description (optional)</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder="What this chart shows..."
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Saving..." : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
