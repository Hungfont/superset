import z from "zod";

export const datasetMetadataSchema = z.object({
  table_name: z.string().min(1, "Dataset name is required").max(255, "Name must be less than 255 characters"),
  description: z.string().optional(),
  main_dttm_col: z.string().optional(),
  cache_timeout: z.number().min(-1, "Cache timeout must be -1, 0, or positive").optional(),
  normalize_columns: z.boolean().optional(),
  filter_select_enabled: z.boolean().optional(),
  is_featured: z.boolean().optional(),
  sql: z.string().optional(),
});

export type DatasetMetadataFormData = z.infer<typeof datasetMetadataSchema>;
