"use client";

import { useCallback, useEffect, useState } from "react";
import { getLogs } from "@/lib/api";
import type { LogEntry } from "@/lib/types";
import { DataTable, Column } from "@/components/data-table";
import { ErrorMessage } from "@/components/error-message";
import { truncateId, formatTokens } from "@/lib/utils";

function formatTimestamp(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString("en-US", {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

const COLUMNS: readonly Column<LogEntry>[] = [
  {
    key: "time",
    header: "Time",
    render: (row) => (
      <span className="text-muted-foreground whitespace-nowrap text-xs">
        {formatTimestamp(row.created_at)}
      </span>
    ),
  },
  {
    key: "session",
    header: "Session",
    render: (row) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(row.session_id)}
      </span>
    ),
  },
  {
    key: "agent",
    header: "Agent",
    render: (row) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(row.agent_id)}
      </span>
    ),
  },
  {
    key: "model",
    header: "Model",
    render: (row) => (
      <span className="text-foreground text-xs font-medium">{row.model}</span>
    ),
  },
  {
    key: "input",
    header: "Input tokens",
    render: (row) => (
      <span className="text-foreground tabular-nums">
        {formatTokens(row.input_tokens)}
      </span>
    ),
    className: "text-right",
  },
  {
    key: "output",
    header: "Output tokens",
    render: (row) => (
      <span className="text-foreground tabular-nums">
        {formatTokens(row.output_tokens)}
      </span>
    ),
    className: "text-right",
  },
  {
    key: "cache_read",
    header: "Cache read",
    render: (row) => (
      <span className="text-muted-foreground tabular-nums">
        {formatTokens(row.cache_read_tokens)}
      </span>
    ),
    className: "text-right",
  },
  {
    key: "latency",
    header: "Latency",
    render: (row) => (
      <span className="text-muted-foreground tabular-nums">
        {row.latency_ms > 0 ? `${row.latency_ms}ms` : "-"}
      </span>
    ),
    className: "text-right",
  },
  {
    key: "error",
    header: "Status",
    render: (row) =>
      row.is_error ? (
        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-destructive/10 text-destructive">
          Error
        </span>
      ) : (
        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-success/10 text-success">
          OK
        </span>
      ),
  },
];

export default function LogsPage() {
  const [logs, setLogs] = useState<readonly LogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await getLogs({ limit: 50, page: currentPage });
      setLogs(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load logs");
    } finally {
      setLoading(false);
    }
  }, [currentPage]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

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
    <div className="space-y-4">
      {error ? (
        <ErrorMessage message={error} onRetry={fetchLogs} />
      ) : (
        <DataTable
          columns={COLUMNS}
          data={logs}
          keyFn={(row) => row.id}
          loading={loading}
          emptyMessage="No request logs found"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}
    </div>
  );
}
