"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { listSessions } from "@/lib/api";
import type { Session } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { timeAgo, truncateId, formatDuration } from "@/lib/utils";

const COLUMNS: readonly Column<Session>[] = [
  {
    key: "id",
    header: "ID",
    render: (s) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(s.id)}
      </span>
    ),
  },
  {
    key: "agent",
    header: "Agent",
    render: (s) => (
      <span className="font-medium text-foreground">
        {s.agent?.name ?? truncateId(s.agent?.id ?? "-")}
      </span>
    ),
  },
  {
    key: "env",
    header: "Environment",
    render: (s) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(s.environment_id)}
      </span>
    ),
  },
  {
    key: "status",
    header: "Status",
    render: (s) => <StatusBadge status={s.status} />,
  },
  {
    key: "created",
    header: "Created",
    render: (s) => (
      <span className="text-muted-foreground">{timeAgo(s.created_at)}</span>
    ),
  },
  {
    key: "duration",
    header: "Duration",
    render: (s) => (
      <span className="text-muted-foreground">
        {formatDuration(s.stats.duration_seconds)}
      </span>
    ),
  },
];

export default function SessionsPage() {
  const router = useRouter();
  const [sessions, setSessions] = useState<readonly Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);

  const fetchSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listSessions({
        limit: 20,
        page: currentPage,
      });
      setSessions(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load sessions"
      );
    } finally {
      setLoading(false);
    }
  }, [currentPage]);

  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

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
        title="Sessions"
        subtitle="View and manage agent sessions"
      />

      {error ? (
        <ErrorMessage message={error} onRetry={fetchSessions} />
      ) : (
        <DataTable
          columns={COLUMNS}
          data={sessions}
          keyFn={(s) => s.id}
          onRowClick={(s) => router.push(`/sessions/${s.id}`)}
          loading={loading}
          emptyMessage="No sessions found"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}
    </div>
  );
}
