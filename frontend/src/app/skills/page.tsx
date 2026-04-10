"use client";

import { useCallback, useEffect, useState } from "react";
import { listSkills, deleteSkill } from "@/lib/api";
import type { Skill } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { ErrorMessage } from "@/components/error-message";
import { UploadSkillDialog } from "@/components/upload-skill-dialog";
import { RowActionsMenu } from "@/components/row-actions-menu";
import { timeAgo, truncateId } from "@/lib/utils";

export default function SkillsPage() {
  const [skills, setSkills] = useState<readonly Skill[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);
  const [showUpload, setShowUpload] = useState(false);

  const fetchSkills = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listSkills({ limit: 20, page: currentPage });
      setSkills(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load skills");
    } finally {
      setLoading(false);
    }
  }, [currentPage]);

  useEffect(() => {
    fetchSkills();
  }, [fetchSkills]);

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

  const handleDelete = async (skill: Skill) => {
    try {
      await deleteSkill(skill.id);
      fetchSkills();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete skill");
    }
  };

  const columns: readonly Column<Skill>[] = [
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
      key: "name",
      header: "Name",
      render: (s) => (
        <span className="font-medium text-foreground">{s.name}</span>
      ),
    },
    {
      key: "description",
      header: "Description",
      render: (s) => (
        <span className="text-muted-foreground text-sm truncate max-w-xs block">
          {s.description || "—"}
        </span>
      ),
    },
    {
      key: "compatibility",
      header: "Compatibility",
      render: (s) => (
        <span className="text-xs text-muted-foreground font-mono">
          {s.compatibility || "—"}
        </span>
      ),
    },
    {
      key: "allowed_tools",
      header: "Tools",
      render: (s) => {
        if (!s.allowed_tools) return <span className="text-muted-foreground">—</span>;
        const tools = s.allowed_tools.split(",").map((t) => t.trim()).filter(Boolean);
        return (
          <div className="flex flex-wrap gap-1">
            {tools.slice(0, 3).map((tool) => (
              <span
                key={tool}
                className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-muted text-muted-foreground"
              >
                {tool}
              </span>
            ))}
            {tools.length > 3 && (
              <span className="text-xs text-muted-foreground">+{tools.length - 3}</span>
            )}
          </div>
        );
      },
    },
    {
      key: "updated",
      header: "Updated",
      render: (s) => (
        <span className="text-muted-foreground">{timeAgo(s.updated_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (s) => (
        <RowActionsMenu
          actions={[
            {
              label: "Delete",
              variant: "danger" as const,
              onClick: () => handleDelete(s),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Skills"
        subtitle="Manage reusable skills for your agents"
        actions={
          <button
            onClick={() => setShowUpload(true)}
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
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
              />
            </svg>
            Upload skill
          </button>
        }
      />

      {error ? (
        <ErrorMessage message={error} onRetry={fetchSkills} />
      ) : (
        <DataTable
          columns={columns}
          data={skills}
          keyFn={(s) => s.id}
          loading={loading}
          emptyMessage="No skills uploaded yet"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}

      <UploadSkillDialog
        open={showUpload}
        onClose={() => setShowUpload(false)}
        onUploaded={fetchSkills}
      />
    </div>
  );
}
