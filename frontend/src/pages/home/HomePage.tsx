import { useAuthStore } from "@/stores/authStore";
import { Loader2 } from "lucide-react";
import { Link } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { useLogout } from "@/hooks/useLogout";

export default function HomePage() {
  const user = useAuthStore((s) => s.user);
  const { mutate: logout, isPending } = useLogout();
  const isAdmin = (user?.roles ?? []).some((role) => role.toLowerCase() === "admin");

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4">
      <p className="text-muted-foreground">Welcome{user ? `, ${user.username}` : ""}.</p>
      <div className="flex flex-col gap-2 sm:flex-row">
        {isAdmin ? (
          <Button asChild variant="outline">
            <Link to="/dashboard">Go to Admin Dashboard</Link>
          </Button>
        ) : null}
        <Button onClick={() => logout(false)} disabled={isPending}>
          {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Sign out
        </Button>
        <Button variant="outline" onClick={() => logout(true)} disabled={isPending}>
          Sign out all devices
        </Button>
      </div>
    </div>
  );
}
