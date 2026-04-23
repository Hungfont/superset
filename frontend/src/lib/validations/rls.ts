import z from "zod";

export const rlsFilterSchema = z.object({
  name: z.string().trim().min(1, "Name is required").max(255, "Max 255 chars"),
  filter_type: z.enum(["Regular", "Base"]),
  clause: z.string().trim().min(1, "Clause is required").max(5000, "Max 5000 chars"),
  group_key: z.string().max(255).optional(),
  description: z.string().max(1000).optional(),
  role_ids: z.array(z.number()).min(1, "At least one role is required"),
  table_ids: z.array(z.number()).min(1, "At least one dataset is required"),
});

export type RLSFilterFormValues = z.infer<typeof rlsFilterSchema>;
