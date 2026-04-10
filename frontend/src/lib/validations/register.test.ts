import { describe, it, expect } from "vitest";
import { registerSchema, passwordStrength, strengthColor } from "./register";

describe("registerSchema", () => {
  const valid = {
    first_name: "John",
    last_name: "Doe",
    username: "johndoe",
    email: "john@example.com",
    password: "StrongP@ss1!",
    confirmPassword: "StrongP@ss1!",
  };

  it("accepts a valid payload", () => {
    expect(registerSchema.safeParse(valid).success).toBe(true);
  });

  it("rejects missing first_name", () => {
    const result = registerSchema.safeParse({ ...valid, first_name: "" });
    expect(result.success).toBe(false);
  });

  it("rejects invalid email", () => {
    const result = registerSchema.safeParse({ ...valid, email: "not-an-email" });
    expect(result.success).toBe(false);
  });

  it("rejects short password", () => {
    const result = registerSchema.safeParse({ ...valid, password: "Short1!", confirmPassword: "Short1!" });
    expect(result.success).toBe(false);
  });

  it("rejects password missing uppercase", () => {
    const result = registerSchema.safeParse({ ...valid, password: "weakpass1!aa", confirmPassword: "weakpass1!aa" });
    expect(result.success).toBe(false);
  });

  it("rejects password missing digit", () => {
    const result = registerSchema.safeParse({ ...valid, password: "WeakPassword!", confirmPassword: "WeakPassword!" });
    expect(result.success).toBe(false);
  });

  it("rejects password missing special char", () => {
    const result = registerSchema.safeParse({ ...valid, password: "WeakPassword12", confirmPassword: "WeakPassword12" });
    expect(result.success).toBe(false);
  });

  it("rejects mismatched confirmPassword", () => {
    const result = registerSchema.safeParse({ ...valid, confirmPassword: "DifferentPass1!" });
    expect(result.success).toBe(false);
    if (!result.success) {
      const paths = result.error.errors.map((e) => e.path.join("."));
      expect(paths).toContain("confirmPassword");
    }
  });

  it("rejects username with invalid chars", () => {
    const result = registerSchema.safeParse({ ...valid, username: "john doe" });
    expect(result.success).toBe(false);
  });

  it("rejects username shorter than 3 chars", () => {
    const result = registerSchema.safeParse({ ...valid, username: "jo" });
    expect(result.success).toBe(false);
  });
});

describe("passwordStrength", () => {
  it("returns 0 for empty string", () => {
    expect(passwordStrength("")).toBe(0);
  });

  it("returns 100 for a fully valid password", () => {
    expect(passwordStrength("StrongP@ss1!")).toBe(100);
  });

  it("returns partial score for missing criteria", () => {
    // lowercase + digit + special (no uppercase, no length≥12)
    const score = passwordStrength("abc1!");
    expect(score).toBeLessThan(100);
    expect(score).toBeGreaterThan(0);
  });
});

describe("strengthColor", () => {
  it("returns red for score ≤ 20", () => {
    expect(strengthColor(20)).toBe("bg-red-500");
  });

  it("returns orange for score 21-60", () => {
    expect(strengthColor(40)).toBe("bg-orange-400");
  });

  it("returns green for score > 60", () => {
    expect(strengthColor(100)).toBe("bg-green-500");
  });
});
