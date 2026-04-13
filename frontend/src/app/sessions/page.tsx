"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { listSessions, archiveSession } from "@/lib/api";
import type { Session } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { RowActionsMenu } from "@/components/row-actions-menu";
import { timeAgo, truncateId, formatDuration } from "@/lib/utils";

export default function SessionsPage() {
  const router = useRouter();
  const [sessions, setSessions] = useState<readonly Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);
  const [showArchived, setShowArchived] = useState(false);
  const [search, setSearch] = useState("");

  const fetchSessions = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setError(null);
      try {
        const res = await listSessions(
          {
            limit: 20,
            page: currentPage,
            include_archived: showArchived,
          },
          signal
        );
        setSessions(res.data);
        setNextPage(res.next_page);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setError(
          err instanceof Error ? err.message : "Failed to load sessions"
        );
      } finally {
        setLoading(false);
      }
    },
    [currentPage, showArchived]
  );

  useEffect(() => {
    const controller = new AbortController();
    fetchSessions(controller.signal);
    return () => controller.abort();
  }, [fetchSessions]);

  const handleArchive = async (session: Session) => {
    try {
      await archiveSession(session.id);
      fetchSessions();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to archive session"
      );
    }
  };

  const columns: readonly Column<Session>[] = [
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
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (s) =>
        s.archived_at ? null : (
          <RowActionsMenu
            actions={[
              { label: "Archive", onClick: () => handleArchive(s) },
            ]}
          />
        ),
    },
  ];

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

  const filteredSessions = search
    ? sessions.filter(
        (s) =>
          s.id.toLowerCase().includes(search.toLowerCase()) ||
          (s.title ?? "").toLowerCase().includes(search.toLowerCase()) ||
          (s.agent?.name ?? "").toLowerCase().includes(search.toLowerCase())
      )
    : sessions;

  return (
    <div>
      <PageHeader
        title="Sessions"
        subtitle="View and manage agent sessions"
      />

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4">
        <div className="flex-1 max-w-xs">
          <input
            type="text"
            placeholder="Search by ID, title, or agent"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
          />
        </div>
        <label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer">
          <input
            type="checkbox"
            checked={showArchived}
            onChange={(e) => setShowArchived(e.target.checked)}
            className="rounded border-border text-primary focus:ring-ring/30"
          />
          Show archived
        </label>
      </div>

      {error ? (
        <ErrorMessage message={error} onRetry={fetchSessions} />
      ) : (
        <DataTable
          columns={columns}
          data={filteredSessions}
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
