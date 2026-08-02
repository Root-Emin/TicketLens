"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Inbox, LayoutDashboard, LogOut } from "lucide-react";

import { useMe } from "@/lib/api/hooks";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/tickets", label: "Queue", icon: Inbox },
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
];

async function logout() {
  await fetch("/api/auth/logout", { method: "POST" });
  window.location.href = "/login";
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { data: me } = useMe();

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-60 flex-col border-r border-border bg-surface">
        <div className="px-5 py-5 text-lg font-bold tracking-tight">
          Ticket<span className="text-accent">Lens</span>
        </div>

        <nav className="flex-1 space-y-1 px-3">
          {nav.map((item) => {
            const active =
              pathname === item.href || pathname.startsWith(item.href + "/");
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-accent/10 text-accent"
                    : "text-muted-foreground hover:bg-surface-muted hover:text-foreground",
                )}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="border-t border-border p-3">
          {me && (
            <div className="mb-2 px-2">
              <div className="truncate text-sm font-medium text-foreground">
                {me.first_name} {me.last_name}
              </div>
              <div className="truncate text-xs text-muted-foreground">
                {me.email}
              </div>
            </div>
          )}
          <button
            onClick={logout}
            className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-surface-muted hover:text-foreground"
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </button>
        </div>
      </aside>

      <main className="flex-1 overflow-x-hidden bg-background">{children}</main>
    </div>
  );
}
