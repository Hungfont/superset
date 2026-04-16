import { Database, Plus } from "lucide-react";
import { useNavigate } from "react-router-dom";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function DatabasesPage() {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Database Connections</h1>
          <p className="text-sm text-muted-foreground">Manage external databases for SQL Lab and datasets.</p>
        </div>

        <Button onClick={() => navigate("/admin/settings/databases/new")}>
          <Plus data-icon="inline-start" />
          Connect a Database
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="size-4" />
            No databases connected yet
          </CardTitle>
          <CardDescription>Create your first encrypted connection.</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            DBC-001 is implemented as a three-step wizard. Use the button above to begin.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
