"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/lib/auth-context";
import { Sidebar } from "./sidebar";

const PUBLIC_PATHS = ["/login", "/register"];

export function AuthGuard({
  children,
}: {
  readonly children: React.ReactNode;
}) {
  const { token, isLoading } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  const isPublic = PUBLIC_PATHS.includes(pathname);

  useEffect(() => {
    if (isLoading) return;
    if (!token && !isPublic) {
      router.replace("/login");
    }
    if (token && isPublic) {
      router.replace("/agents");
    }
  }, [token, isLoading, isPublic, router]);

  // Still loading auth state from localStorage
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    );
  }

  // Public pages (login/register) — no sidebar
  if (isPublic) {
    return <>{children}</>;
  }

  // Not authenticated — will redirect, show nothing
  if (!token) {
    return null;
  }

  // Authenticated — show sidebar + content
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <main className="flex-1 p-8 overflow-auto">{children}</main>
    </div>
  );
}
