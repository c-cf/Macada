"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { listAgents } from "@/lib/api";
import type { Agent } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { CreateAgentDialog } from "@/components/create-agent-dialog";
import { timeAgo, truncateId, getAgentStatus } from "@/lib/utils";

const COLUMNS: readonly Column<Agent>[] = [
  {
    key: "id",
    header: "ID",
    render: (agent) => (
      <span className="font-mono text-xs text-muted-foreground">
        {truncateId(agent.id)}
      </span>
    ),
  },
  {
    key: "name",
    header: "Name",
    render: (agent) => (
      <span className="font-medium text-foreground">{agent.name}</span>
    ),
  },
  {
    key: "model",
    header: "Model",
    render: (agent) => (
      <span className="text-muted-foreground">{agent.model.id}</span>
    ),
  },
  {
    key: "status",
    header: "Status",
    render: (agent) => <StatusBadge status={getAgentStatus(agent)} />,
  },
  {
    key: "created",
    header: "Created",
    render: (agent) => (
      <span className="text-muted-foreground">{timeAgo(agent.created_at)}</span>
    ),
  },
  {
    key: "updated",
    header: "Last updated",
    render: (agent) => (
      <span className="text-muted-foreground">{timeAgo(agent.updated_at)}</span>
    ),
  },
];

export default function AgentsPage() {
  const router = useRouter();
  const [agents, setAgents] = useState<readonly Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);
  const [showArchived, setShowArchived] = useState(false);
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  const fetchAgents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listAgents({
        limit: 20,
        page: currentPage,
        include_archived: showArchived,
      });
      setAgents(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load agents");
    } finally {
      setLoading(false);
    }
  }, [currentPage, showArchived]);

  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

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

  const filteredAgents = search
    ? agents.filter(
        (a) =>
          a.id.toLowerCase().includes(search.toLowerCase()) ||
          a.name.toLowerCase().includes(search.toLowerCase())
      )
    : agents;

  return (
    <div>
      <PageHeader
        title="Agents"
        subtitle="Create and manage autonomous agents"
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
            New agent
          </button>
        }
      />

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4">
        <div className="flex-1 max-w-xs">
          <input
            type="text"
            placeholder="Go to agent ID"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
          />
        </div>
        <select className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30">
          <option>Created All time</option>
          <option>Last 24 hours</option>
          <option>Last 7 days</option>
          <option>Last 30 days</option>
        </select>
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
        <ErrorMessage message={error} onRetry={fetchAgents} />
      ) : (
        <DataTable
          columns={COLUMNS}
          data={filteredAgents}
          keyFn={(a) => a.id}
          onRowClick={(agent) => router.push(`/agents/${agent.id}`)}
          loading={loading}
          emptyMessage="No agents found"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}

      <CreateAgentDialog
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreated={fetchAgents}
      />
    </div>
  );
}
