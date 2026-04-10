import { z } from "zod";

export const registerSchema = z
  .object({
    first_name: z.string().min(1, "First name is required"),
    last_name: z.string().min(1, "Last name is required"),
    username: z
      .string()
      .min(3, "Username must be at least 3 characters")
      .max(64, "Username must be at most 64 characters")
      .regex(/^[a-zA-Z0-9_]+$/, "Username may only contain letters, numbers, and underscores"),
    email: z.string().email("Enter a valid email address"),
    password: z
      .string()
      .min(12, "Password must be at least 12 characters")
      .regex(/[A-Z]/, "Password must contain at least one uppercase letter")
      .regex(/[a-z]/, "Password must contain at least one lowercase letter")
      .regex(/[0-9]/, "Password must contain at least one digit")
      .regex(/[^a-zA-Z0-9]/, "Password must contain at least one special character"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  });

export type RegisterFormValues = z.infer<typeof registerSchema>;

/** Returns a 0-100 strength score for a password. */
export function passwordStrength(password: string): number {
  if (!password) return 0;
  let score = 0;
  if (password.length >= 12) score += 20;
  if (/[A-Z]/.test(password)) score += 20;
  if (/[a-z]/.test(password)) score += 20;
  if (/[0-9]/.test(password)) score += 20;
  if (/[^a-zA-Z0-9]/.test(password)) score += 20;
  return score;
}

/** Maps strength score to a Tailwind color class. */
export function strengthColor(score: number): string {
  if (score <= 20) return "bg-red-500";
  if (score <= 60) return "bg-orange-400";
  return "bg-green-500";
}
