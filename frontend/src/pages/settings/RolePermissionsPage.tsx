import { useParams } from "react-router-dom";

export default function RolePermissionsPage() {
  const { id } = useParams();

  return (
    <main className="mx-auto w-full max-w-3xl p-6">
      <h1 className="text-2xl font-semibold">Role Permission Matrix</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        Role ID: {id}. Permission matrix integration can be attached to this route.
      </p>
    </main>
  );
}
