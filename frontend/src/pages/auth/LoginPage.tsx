import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff, Loader2 } from "lucide-react";

import { loginSchema, type LoginFormValues } from "@/lib/validations/login";
import { useLogin } from "@/hooks/useLogin";
import type { LoginError } from "@/api/auth";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";

const LOCKOUT_DURATION_MS = 15 * 60 * 1000;

function safeLocalStorage() {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function formatCountdown(ms: number): string {
  const totalSec = Math.max(0, Math.ceil(ms / 1000));
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export default function LoginPage() {
  const [searchParams] = useSearchParams();
  const [showPassword, setShowPassword] = useState(false);
  const [lockoutMs, setLockoutMs] = useState<number | null>(null);
  const [rememberMe, setRememberMe] = useState(false);

  const isActivated = searchParams.get("activated") === "true";
  const from = searchParams.get("from");

  const savedUsername = safeLocalStorage()?.getItem("rememberedUsername") ?? "";

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: savedUsername,
      password: "",
    },
  });

  const { mutate, isPending, isError, error } = useLogin();
  const loginError = error as LoginError | null;
  const isLocked = isError && loginError?.status === 423;
  const isInactive = isError && loginError?.status === 403;

  // Lockout countdown timer
  useEffect(() => {
    if (!isLocked) {
      setLockoutMs(null);
      return;
    }
    const lockedUntilStr = loginError?.locked_until;
    const expiry = lockedUntilStr
      ? new Date(lockedUntilStr).getTime()
      : Date.now() + LOCKOUT_DURATION_MS;

    const tick = () => setLockoutMs(Math.max(0, expiry - Date.now()));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [isLocked, loginError]);

  function onSubmit(values: LoginFormValues) {
    const storage = safeLocalStorage();
    if (storage) {
      if (rememberMe) {
        storage.setItem("rememberedUsername", values.username);
      } else {
        storage.removeItem("rememberedUsername");
      }
    }
    mutate(values);
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-2xl font-bold">Sign in</CardTitle>
          <CardDescription>Enter your credentials to access your account</CardDescription>
        </CardHeader>

        <CardContent className="flex flex-col gap-4">
          {/* Redirected from protected route */}
          {from && !isActivated && (
            <Alert>
              <AlertDescription>Sign in to continue.</AlertDescription>
            </Alert>
          )}

          {/* Account just activated */}
          {isActivated && (
            <Alert className="border-green-500 bg-green-50 text-green-800">
              <AlertTitle>Account activated!</AlertTitle>
              <AlertDescription>Welcome aboard. Sign in to get started.</AlertDescription>
            </Alert>
          )}

          {/* Lockout error */}
          {isLocked && lockoutMs !== null && (
            <Alert variant="destructive" role="alert" aria-live="assertive">
              <AlertTitle>Account locked</AlertTitle>
              <AlertDescription>
                Too many failed attempts. Try again in{" "}
                <strong>{formatCountdown(lockoutMs)}</strong>.
              </AlertDescription>
            </Alert>
          )}

          {/* Inactive account */}
          {isInactive && (
            <Alert variant="destructive" role="alert" aria-live="assertive">
              <AlertTitle>Account inactive</AlertTitle>
              <AlertDescription>
                Your account is inactive. Please contact support.
              </AlertDescription>
            </Alert>
          )}

          {/* Generic error (401, 429, 500) */}
          {isError && !isLocked && !isInactive && (
            <Alert variant="destructive" role="alert" aria-live="assertive">
              <AlertDescription>{loginError?.message ?? "Something went wrong"}</AlertDescription>
            </Alert>
          )}

          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmit)}
              className="flex flex-col gap-4"
              aria-label="Sign in form"
            >
              {/* Username */}
              <FormField
                control={form.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username or email</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete="username"
                        placeholder="johndoe"
                        disabled={isPending}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Password */}
              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center justify-between">
                      <FormLabel>Password</FormLabel>
                      <Link
                        to="/forgot-password"
                        className="text-xs text-muted-foreground hover:underline"
                        tabIndex={-1}
                      >
                        Forgot password?
                      </Link>
                    </div>
                    <div className="relative">
                      <FormControl>
                        <Input
                          type={showPassword ? "text" : "password"}
                          autoComplete="current-password"
                          placeholder="••••••••"
                          disabled={isPending}
                          className="pr-10"
                          {...field}
                        />
                      </FormControl>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={showPassword ? "Hide password" : "Show password"}
                        className="absolute right-1 top-1/2 -translate-y-1/2 size-8"
                        onClick={() => setShowPassword((v) => !v)}
                        tabIndex={-1}
                        disabled={isPending}
                      >
                        {showPassword ? <EyeOff /> : <Eye />}
                      </Button>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Remember me */}
              <div className="flex items-center gap-2">
                <Checkbox
                  id="remember-me"
                  checked={rememberMe}
                  onCheckedChange={(checked) => setRememberMe(checked === true)}
                  disabled={isPending}
                />
                <Label htmlFor="remember-me" className="text-sm font-normal cursor-pointer">
                  Remember me
                </Label>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={isPending}
                aria-busy={isPending}
              >
                {isPending && <Loader2 className="mr-2 size-4 animate-spin" />}
                Sign In
              </Button>
            </form>
          </Form>

          <Separator />

          {/* OAuth placeholder */}
          <div className="flex flex-col gap-2">
            <Button variant="outline" className="w-full" type="button" disabled={isPending}>
              <svg className="mr-2 size-4" viewBox="0 0 24 24" aria-hidden="true">
                <path
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  fill="#4285F4"
                />
                <path
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  fill="#34A853"
                />
                <path
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
                  fill="#FBBC05"
                />
                <path
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  fill="#EA4335"
                />
              </svg>
              Continue with Google
            </Button>
          </div>
        </CardContent>

        <CardFooter>
          <p className="text-sm text-muted-foreground w-full text-center">
            Don&apos;t have an account?{" "}
            <Link
              to="/register"
              className="text-primary font-medium underline-offset-4 hover:underline"
            >
              Create one
            </Link>
          </p>
        </CardFooter>
      </Card>
    </div>
  );
}
