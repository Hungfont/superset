import { LayoutDashboard, ShieldCheck } from "lucide-react";
import { NavLink, Outlet } from "react-router-dom";

import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

const adminNavItems = [
  {
    to: "/dashboard",
    label: "Dashboard",
    icon: LayoutDashboard,
    end: true,
  },
  {
    to: "/dashboard/roles",
    label: "Roles",
    icon: ShieldCheck,
    end: false,
  },
];

export default function AdminLayout() {
  return (
    <main className="min-h-screen bg-gradient-to-b from-slate-50 via-background to-background">
      <div className="mx-auto grid w-full max-w-7xl grid-cols-1 gap-4 px-4 py-4 md:grid-cols-[260px_1fr]">
        <aside className="rounded-xl border bg-card p-4">
          <div className="mb-3">
            <p className="text-sm font-semibold">Admin Navigation</p>
            <p className="text-xs text-muted-foreground">Danh sach chuc nang role Admin co the xu ly</p>
          </div>
          <Separator className="mb-3" />

          <nav className="flex flex-col gap-2">
            {adminNavItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    cn(
                      "flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors",
                      isActive
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground",
                    )
                  }
                >
                  <Icon className="h-4 w-4" />
                  <span>{item.label}</span>
                </NavLink>
              );
            })}
          </nav>
        </aside>

        <section className="min-w-0 rounded-xl border bg-card p-4 md:p-6">
          <Outlet />
        </section>
      </div>
    </main>
  );
}
