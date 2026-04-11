import { useEffect, useState } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle, XCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

const HASH_REGEX = /^[0-9a-f]{64}$/i;
const REDIRECT_SECONDS = 3;

interface VerifyErrorResponse {
  error: string;
}

async function verifyEmail(hash: string): Promise<void> {
  const res = await fetch(`/api/v1/auth/verify?hash=${encodeURIComponent(hash)}`);
  if (!res.ok) {
    const body: VerifyErrorResponse = await res.json().catch(() => ({ error: "Unexpected error" }));
    throw Object.assign(new Error(body.error), { status: res.status });
  }
}

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    const status = (error as Error & { status?: number }).status;
    if (status === 410) return "This verification link has expired. Please register again.";
    if (status === 404) return "This link is invalid or has already been used.";
    return error.message;
  }
  return "An unexpected error occurred.";
}

export default function VerifyPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const hash = searchParams.get("hash") ?? "";

  const [countdown, setCountdown] = useState(REDIRECT_SECONDS);

  const isMalformed = hash !== "" && !HASH_REGEX.test(hash);

  const { isLoading, isSuccess, isError, error } = useQuery({
    queryKey: ["email-verify", hash],
    queryFn: () => verifyEmail(hash),
    enabled: !isMalformed && hash !== "",
    retry: false,
  });

  // Auto-redirect countdown on success
  useEffect(() => {
    if (!isSuccess) return;
    if (countdown <= 0) {
      navigate("/login?activated=true");
      return;
    }
    const timer = setTimeout(() => setCountdown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [isSuccess, countdown, navigate]);

  if (isMalformed || hash === "") {
    return (
      <VerifyLayout>
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <XCircle className="h-4 w-4" />
          <AlertTitle>Invalid link</AlertTitle>
          <AlertDescription>
            The verification link is malformed. Please use the link from your email.
          </AlertDescription>
        </Alert>
        <Button variant="outline" className="w-full mt-4" onClick={() => navigate("/register")}>
          Back to Register
        </Button>
      </VerifyLayout>
    );
  }

  if (isLoading) {
    return (
      <VerifyLayout>
        <div className="flex flex-col gap-3">
          <Skeleton className="h-6 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-2/3" />
          <Skeleton className="h-10 w-full mt-2" />
        </div>
      </VerifyLayout>
    );
  }

  if (isSuccess) {
    return (
      <VerifyLayout>
        <Alert role="alert" aria-live="assertive" className="border-green-500 text-green-700 bg-green-50">
          <CheckCircle className="h-4 w-4 text-green-600" />
          <AlertTitle>Account activated!</AlertTitle>
          <AlertDescription>
            You can now sign in. Redirecting in{" "}
            <Badge variant="secondary">{countdown}s</Badge>
          </AlertDescription>
        </Alert>
        <Button className="w-full mt-4" onClick={() => navigate("/login?activated=true")}>
          Go to Login
        </Button>
      </VerifyLayout>
    );
  }

  if (isError) {
    return (
      <VerifyLayout>
        <Alert variant="destructive" role="alert" aria-live="assertive">
          <XCircle className="h-4 w-4" />
          <AlertTitle>Verification failed</AlertTitle>
          <AlertDescription>{resolveErrorMessage(error)}</AlertDescription>
        </Alert>
        <Button variant="outline" className="w-full mt-4" onClick={() => navigate("/register")}>
          Back to Register
        </Button>
      </VerifyLayout>
    );
  }

  return null;
}

function VerifyLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardContent className="pt-6">{children}</CardContent>
      </Card>
    </div>
  );
}
