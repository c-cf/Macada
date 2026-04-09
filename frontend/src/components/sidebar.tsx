"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState, useRef, useEffect } from "react";
import { useAuth } from "@/lib/auth-context";

interface NavItem {
  readonly label: string;
  readonly href?: string;
  readonly children?: readonly NavItem[];
}

const NAV_ITEMS: readonly NavItem[] = [
  {
    label: "Agents",
    children: [
      { label: "Agents", href: "/agents" },
      { label: "Sessions", href: "/sessions" },
      { label: "Environments", href: "/environments" },
    ],
  },
  {
    label: "Analytics",
    children: [
      { label: "Usage", href: "/analytics/usage" },
      { label: "Cost", href: "/analytics/cost" },
      { label: "Logs", href: "/analytics/logs" },
    ],
  },
];

function ChevronDown({ open }: { readonly open: boolean }) {
  return (
    <svg
      className={`w-4 h-4 transition-transform ${open ? "rotate-0" : "-rotate-90"}`}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M19 9l-7 7-7-7"
      />
    </svg>
  );
}

function NavSection({ item }: { readonly item: NavItem }) {
  const pathname = usePathname();
  const [open, setOpen] = useState(true);

  const isChildActive = item.children?.some((child) =>
    child.href ? pathname.startsWith(child.href) : false
  );

  if (!item.children) {
    const isActive = item.href ? pathname.startsWith(item.href) : false;
    return (
      <Link
        href={item.href ?? "#"}
        className={`block px-4 py-2 text-sm rounded-md transition-colors ${
          isActive
            ? "bg-sidebar-active text-foreground font-medium border-l-2 border-primary"
            : "text-muted-foreground hover:bg-sidebar-hover hover:text-foreground"
        }`}
      >
        {item.label}
      </Link>
    );
  }

  return (
    <div>
      <button
        onClick={() => setOpen((prev) => !prev)}
        className={`w-full flex items-center justify-between px-4 py-2 text-sm rounded-md transition-colors ${
          isChildActive
            ? "text-foreground font-medium"
            : "text-muted-foreground hover:bg-sidebar-hover hover:text-foreground"
        }`}
      >
        <span>{item.label}</span>
        <ChevronDown open={open} />
      </button>
      {open && (
        <div className="ml-3 mt-0.5 space-y-0.5">
          {item.children.map((child) => {
            const isActive = child.href
              ? pathname.startsWith(child.href)
              : false;
            return (
              <Link
                key={child.label}
                href={child.href ?? "#"}
                className={`block px-4 py-1.5 text-sm rounded-md transition-colors ${
                  isActive
                    ? "bg-sidebar-active text-foreground font-medium border-l-2 border-primary"
                    : "text-muted-foreground hover:bg-sidebar-hover hover:text-foreground"
                }`}
              >
                {child.label}
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

function WorkspaceSelector() {
  const { workspaces, activeWorkspaceId, switchWorkspace } = useAuth();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const active = workspaces.find((ws) => ws.id === activeWorkspaceId);

  // Close dropdown on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((prev) => !prev)}
        className="w-full flex items-center justify-between px-3 py-2 bg-card/60 rounded-lg border border-border text-sm text-foreground hover:bg-card/80 transition-colors"
      >
        <div className="flex items-center gap-2 min-w-0">
          <div className="w-5 h-5 rounded bg-primary/10 border border-primary/20 flex items-center justify-center text-xs font-medium text-primary shrink-0">
            {(active?.name ?? "W")[0].toUpperCase()}
          </div>
          <span className="truncate">{active?.name ?? "Select workspace"}</span>
        </div>
        <ChevronDown open={open} />
      </button>

      {open && workspaces.length > 1 && (
        <div className="absolute left-0 right-0 mt-1 bg-card rounded-lg border border-border shadow-lg z-50 py-1">
          {workspaces.map((ws) => (
            <button
              key={ws.id}
              onClick={() => {
                switchWorkspace(ws.id);
                setOpen(false);
              }}
              className={`w-full text-left px-3 py-2 text-sm transition-colors ${
                ws.id === activeWorkspaceId
                  ? "bg-primary/5 text-primary font-medium"
                  : "text-muted-foreground hover:bg-muted"
              }`}
            >
              <div className="flex items-center gap-2">
                <div className="w-5 h-5 rounded bg-primary/10 border border-primary/20 flex items-center justify-center text-xs font-medium text-primary">
                  {ws.name[0].toUpperCase()}
                </div>
                <span className="truncate">{ws.name}</span>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export function Sidebar() {
  const { user, logout } = useAuth();
  const router = useRouter();

  function handleLogout() {
    logout();
    router.push("/login");
  }

  return (
    <aside className="w-64 min-h-screen bg-sidebar border-r border-border flex flex-col">
      {/* Logo */}
      <div className="px-5 py-5 border-b border-border">
        <Link href="/agents" className="flex items-center gap-2">
          <img src="/macada.png" alt="Macada" className="w-7 h-7 rounded-lg" />
          <span className="font-semibold text-foreground text-lg">
            Macada
          </span>
        </Link>
      </div>

      {/* Workspace selector */}
      <div className="px-4 py-3 border-b border-border">
        <WorkspaceSelector />
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1">
        {NAV_ITEMS.map((item) => (
          <NavSection key={item.label} item={item} />
        ))}
      </nav>

      {/* User + Logout */}
      <div className="px-4 py-3 border-t border-border">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <div className="w-7 h-7 rounded-full bg-muted flex items-center justify-center text-xs font-medium text-muted-foreground shrink-0">
              {(user?.name ?? user?.email ?? "U")[0].toUpperCase()}
            </div>
            <div className="min-w-0">
              <p className="text-sm font-medium text-foreground truncate">
                {user?.name ?? "User"}
              </p>
              <p className="text-xs text-muted-foreground truncate">
                {user?.email}
              </p>
            </div>
          </div>
          <button
            onClick={handleLogout}
            title="Sign out"
            className="p-1.5 rounded-md text-muted-foreground hover:text-muted-foreground hover:bg-muted transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
              />
            </svg>
          </button>
        </div>
      </div>
    </aside>
  );
}
