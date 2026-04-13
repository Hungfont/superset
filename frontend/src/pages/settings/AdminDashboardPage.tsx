import { ShieldCheck, UserCog, Waypoints } from "lucide-react";
import { Link } from "react-router-dom";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function AdminDashboardPage() {
  return (
    <div className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold">Admin Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Khu vuc quan tri, chi role Admin moi co quyen truy cap.
        </p>
      </header>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <ShieldCheck className="h-4 w-4" />
              Role Control
            </CardTitle>
            <CardDescription>Quan ly role va policy cho he thong</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="secondary">/dashboard/roles</Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Waypoints className="h-4 w-4" />
              API Access
            </CardTitle>
            <CardDescription>Kiem soat cac API quan tri</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="outline">Admin-only</Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <UserCog className="h-4 w-4" />
              Security
            </CardTitle>
            <CardDescription>Phan quyen va bao mat theo role</CardDescription>
          </CardHeader>
          <CardContent>
            <Badge>RBAC</Badge>
          </CardContent>
        </Card>
      </div>

      <div>
        <Button asChild>
          <Link to="/dashboard/roles">Go to RolesPage</Link>
        </Button>
      </div>
    </div>
  );
}
