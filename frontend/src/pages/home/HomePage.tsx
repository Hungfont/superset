import { useAuthStore } from "@/stores/authStore";

export default function HomePage() {
  const user = useAuthStore((s) => s.user);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-muted-foreground">
        Welcome{user ? `, ${user.username}` : ""}.
      </p>
    </div>
  );
}
