"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getAgent, listSessions } from "@/lib/api";
import type { Agent, Session } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { DataTable, Column } from "@/components/data-table";
import { ErrorMessage } from "@/components/error-message";
import { EditAgentDialog } from "@/components/edit-agent-dialog";
import {
  timeAgo,
  truncateId,
  getAgentStatus,
  formatDuration,
} from "@/lib/utils";

const SESSION_COLUMNS: readonly Column<Session>[] = [
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
    key: "status",
    header: "Status",
    render: (s) => <StatusBadge status={s.status} />,
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
    key: "duration",
    header: "Duration",
    render: (s) => (
      <span className="text-muted-foreground">
        {formatDuration(s.stats.duration_seconds)}
      </span>
    ),
  },
  {
    key: "created",
    header: "Created",
    render: (s) => (
      <span className="text-muted-foreground">{timeAgo(s.created_at)}</span>
    ),
  },
];

export default function AgentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const agentId = params.agentId as string;

  const [agent, setAgent] = useState<Agent | null>(null);
  const [sessions, setSessions] = useState<readonly Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"agent" | "sessions">("agent");
  const [editOpen, setEditOpen] = useState(false);

  const fetchData = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setError(null);
      try {
        const [agentData, sessionsData] = await Promise.all([
          getAgent(agentId, signal),
          listSessions({ agent_id: agentId, limit: 20 }, signal),
        ]);
        setAgent(agentData);
        setSessions(sessionsData.data);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setError(err instanceof Error ? err.message : "Failed to load agent");
      } finally {
        setLoading(false);
      }
    },
    [agentId]
  );

  useEffect(() => {
    const controller = new AbortController();
    fetchData(controller.signal);
    return () => controller.abort();
  }, [fetchData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
      </div>
    );
  }

  if (error || !agent) {
    return (
      <ErrorMessage message={error ?? "Agent not found"} onRetry={fetchData} />
    );
  }

  const status = getAgentStatus(agent);
  const tools = Array.isArray(agent.tools) ? agent.tools : [];
  const mcpServers = Array.isArray(agent.mcp_servers) ? agent.mcp_servers : [];
  const skills = Array.isArray(agent.skills) ? agent.skills : [];

  return (
    <div>
      <PageHeader
        title={agent.name}
        subtitle={`${agent.id} \u00B7 Last updated ${timeAgo(agent.updated_at)}`}
        breadcrumbs={[
          { label: "Agents", href: "/agents" },
          { label: agent.name },
        ]}
        actions={
          <button
            onClick={() => setEditOpen(true)}
            className="px-4 py-2 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
          >
            Edit
          </button>
        }
      />

      <div className="flex items-center gap-2 mb-6">
        <StatusBadge status={status} />
        {agent.description && (
          <span className="text-sm text-muted-foreground ml-2">
            {agent.description}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="border-b border-border mb-6">
        <div className="flex gap-0">
          <button
            onClick={() => setActiveTab("agent")}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "agent"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Agent
          </button>
          <button
            onClick={() => setActiveTab("sessions")}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "sessions"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Sessions
          </button>
        </div>
      </div>

      {activeTab === "agent" ? (
        <div className="space-y-6">
          {/* Version */}
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium px-2.5 py-1 bg-muted text-muted-foreground rounded-full">
              Version {agent.version}
            </span>
            <span className="text-xs text-muted-foreground">
              Model: {agent.model.id}
            </span>
          </div>

          {/* System Prompt */}
          <div className="bg-card rounded-xl border border-border p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">
              System Prompt
            </h3>
            {agent.system ? (
              <pre className="text-sm text-muted-foreground bg-muted rounded-lg p-4 font-mono whitespace-pre-wrap overflow-x-auto max-h-96 overflow-y-auto">
                {agent.system}
              </pre>
            ) : (
              <p className="text-sm text-muted-foreground italic">
                No system prompt configured
              </p>
            )}
          </div>

          {/* MCPs and Tools */}
          <div className="bg-card rounded-xl border border-border p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">
              MCPs and Tools
            </h3>
            {mcpServers.length > 0 || tools.length > 0 ? (
              <div className="space-y-3">
                {mcpServers.length > 0 && (
                  <div>
                    <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                      MCP Servers ({mcpServers.length})
                    </h4>
                    <pre className="text-xs text-muted-foreground bg-muted rounded-lg p-3 font-mono whitespace-pre-wrap overflow-x-auto">
                      {JSON.stringify(mcpServers, null, 2)}
                    </pre>
                  </div>
                )}
                {tools.length > 0 && (
                  <div>
                    <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                      Tools ({tools.length})
                    </h4>
                    <pre className="text-xs text-muted-foreground bg-muted rounded-lg p-3 font-mono whitespace-pre-wrap overflow-x-auto">
                      {JSON.stringify(tools, null, 2)}
                    </pre>
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground italic">
                No MCPs or tools configured
              </p>
            )}
          </div>

          {/* Skills */}
          <div className="bg-card rounded-xl border border-border p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">Skills</h3>
            {skills.length > 0 ? (
              <pre className="text-xs text-muted-foreground bg-muted rounded-lg p-3 font-mono whitespace-pre-wrap overflow-x-auto">
                {JSON.stringify(skills, null, 2)}
              </pre>
            ) : (
              <p className="text-sm text-muted-foreground italic">
                No skills configured
              </p>
            )}
          </div>
        </div>
      ) : (
        <DataTable
          columns={SESSION_COLUMNS}
          data={sessions}
          keyFn={(s) => s.id}
          onRowClick={(s) => router.push(`/sessions/${s.id}`)}
          emptyMessage="No sessions found for this agent"
        />
      )}

      <EditAgentDialog
        open={editOpen}
        agent={agent}
        onClose={() => setEditOpen(false)}
        onUpdated={(updated) => setAgent(updated)}
      />
    </div>
  );
}
