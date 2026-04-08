"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { listEnvironments } from "@/lib/api";
import type { Environment } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { CreateEnvironmentDialog } from "@/components/create-environment-dialog";
import { timeAgo, truncateId } from "@/lib/utils";

function getNetworkingType(env: Environment): string {
  const networking = env.config?.networking;
  if (!networking) return "-";
  if (typeof networking === "object" && "type" in networking) {
    return (networking as { type: string }).type;
  }
  return "-";
}

const COLUMNS: readonly Column<Environment>[] = [
  {
    key: "id",
    header: "ID",
    render: (env) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(env.id)}
      </span>
    ),
  },
  {
    key: "name",
    header: "Name",
    render: (env) => (
      <span className="font-medium text-foreground">{env.name || "-"}</span>
    ),
  },
  {
    key: "type",
    header: "Type",
    render: (env) => (
      <span className="text-muted-foreground">{env.config?.type ?? "-"}</span>
    ),
  },
  {
    key: "networking",
    header: "Networking",
    render: (env) => (
      <span className="text-muted-foreground capitalize">
        {getNetworkingType(env)}
      </span>
    ),
  },
  {
    key: "status",
    header: "Status",
    render: (env) => (
      <StatusBadge status={env.archived_at ? "Archived" : "Active"} />
    ),
  },
  {
    key: "created",
    header: "Created",
    render: (env) => (
      <span className="text-muted-foreground">{timeAgo(env.created_at)}</span>
    ),
  },
];

export default function EnvironmentsPage() {
  const router = useRouter();
  const [environments, setEnvironments] = useState<readonly Environment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);
  const [showCreate, setShowCreate] = useState(false);

  const fetchEnvironments = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listEnvironments({
        limit: 20,
        page: currentPage,
      });
      setEnvironments(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load environments"
      );
    } finally {
      setLoading(false);
    }
  }, [currentPage]);

  useEffect(() => {
    fetchEnvironments();
  }, [fetchEnvironments]);

  const handleNextPage = () => {
    if (nextPage) {
      setPageStack((prev) => [...prev, currentPage ?? ""]);
      setCurrentPage(nextPage);
    }
  };

  const handlePrevPage = () => {
    if (pageStack.length > 0) {
      const newStack = [...pageStack];
      const prev = newStack.pop();
      setPageStack(newStack);
      setCurrentPage(prev || undefined);
    }
  };

  return (
    <div>
      <PageHeader
        title="Environments"
        subtitle="Manage sandbox environments for agent execution"
        actions={
          <button
            onClick={() => setShowCreate(true)}
            className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors flex items-center gap-1.5"
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
                d="M12 4v16m8-8H4"
              />
            </svg>
            New environment
          </button>
        }
      />

      {error ? (
        <ErrorMessage message={error} onRetry={fetchEnvironments} />
      ) : (
        <DataTable
          columns={COLUMNS}
          data={environments}
          keyFn={(e) => e.id}
          onRowClick={(env) => router.push(`/environments/${env.id}`)}
          loading={loading}
          emptyMessage="No environments found"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}

      <CreateEnvironmentDialog
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreated={fetchEnvironments}
      />
    </div>
  );
}
