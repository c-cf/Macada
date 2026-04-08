"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ReactNode } from "react";

const ANALYTICS_TABS = [
  { label: "Usage", href: "/analytics/usage" },
  { label: "Cost", href: "/analytics/cost" },
  { label: "Logs", href: "/analytics/logs" },
] as const;

export default function AnalyticsLayout({
  children,
}: {
  readonly children: ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-foreground">Analytics</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Monitor token usage, costs, and request logs
        </p>
      </div>

      <div className="flex items-center gap-1 mb-6 border-b border-border">
        {ANALYTICS_TABS.map((tab) => {
          const isActive = pathname === tab.href;
          return (
            <Link
              key={tab.href}
              href={tab.href}
              className={`px-4 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px ${
                isActive
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>

      {children}
    </div>
  );
}
