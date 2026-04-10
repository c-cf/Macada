"use client";

import { useCallback, useEffect, useState } from "react";
import { listFiles, deleteFile, downloadFile } from "@/lib/api";
import type { FileMetadata } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { DataTable, Column } from "@/components/data-table";
import { ErrorMessage } from "@/components/error-message";
import { UploadFileDialog } from "@/components/upload-file-dialog";
import { RowActionsMenu } from "@/components/row-actions-menu";
import { timeAgo, truncateId, formatBytes } from "@/lib/utils";

function mimeIcon(mime: string): string {
  if (mime.startsWith("image/")) return "IMG";
  if (mime === "application/pdf") return "PDF";
  if (mime.startsWith("text/")) return "TXT";
  if (mime.includes("json")) return "JSON";
  if (mime.includes("csv")) return "CSV";
  return "FILE";
}

export default function FilesPage() {
  const [files, setFiles] = useState<readonly FileMetadata[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPage, setNextPage] = useState<string | null>(null);
  const [pageStack, setPageStack] = useState<readonly string[]>([]);
  const [currentPage, setCurrentPage] = useState<string | undefined>(undefined);
  const [showUpload, setShowUpload] = useState(false);

  const fetchFiles = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listFiles({ limit: 20, page: currentPage });
      setFiles(res.data);
      setNextPage(res.next_page);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load files");
    } finally {
      setLoading(false);
    }
  }, [currentPage]);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

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

  const handleDelete = async (file: FileMetadata) => {
    try {
      await deleteFile(file.id);
      fetchFiles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete file");
    }
  };

  const handleDownload = async (file: FileMetadata) => {
    try {
      const blob = await downloadFile(file.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = file.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to download file");
    }
  };

  const columns: readonly Column<FileMetadata>[] = [
    {
      key: "id",
      header: "ID",
      render: (f) => (
        <span className="font-mono text-xs text-muted-foreground">
          {truncateId(f.id)}
        </span>
      ),
    },
    {
      key: "type_icon",
      header: "Type",
      render: (f) => (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-muted text-muted-foreground">
          {mimeIcon(f.mime_type)}
        </span>
      ),
    },
    {
      key: "filename",
      header: "Filename",
      render: (f) => (
        <span className="font-medium text-foreground">{f.filename}</span>
      ),
    },
    {
      key: "size",
      header: "Size",
      render: (f) => (
        <span className="text-muted-foreground">
          {formatBytes(f.size_bytes)}
        </span>
      ),
    },
    {
      key: "mime",
      header: "MIME Type",
      render: (f) => (
        <span className="text-muted-foreground text-xs font-mono">
          {f.mime_type}
        </span>
      ),
    },
    {
      key: "downloadable",
      header: "Downloadable",
      render: (f) => (
        <span
          className={`text-xs ${f.downloadable ? "text-green-500" : "text-muted-foreground"}`}
        >
          {f.downloadable ? "Yes" : "No"}
        </span>
      ),
    },
    {
      key: "created",
      header: "Created",
      render: (f) => (
        <span className="text-muted-foreground">{timeAgo(f.created_at)}</span>
      ),
    },
    {
      key: "actions",
      header: "",
      className: "w-10",
      render: (f) => (
        <RowActionsMenu
          actions={[
            ...(f.downloadable
              ? [
                  {
                    label: "Download",
                    onClick: () => handleDownload(f),
                  },
                ]
              : []),
            {
              label: "Delete",
              variant: "danger" as const,
              onClick: () => handleDelete(f),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Files"
        subtitle="Manage workspace files for agent sessions"
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
            Upload file
          </button>
        }
      />

      {error ? (
        <ErrorMessage message={error} onRetry={fetchFiles} />
      ) : (
        <DataTable
          columns={columns}
          data={files}
          keyFn={(f) => f.id}
          loading={loading}
          emptyMessage="No files uploaded yet"
          nextPage={nextPage}
          onNextPage={handleNextPage}
          onPrevPage={handlePrevPage}
          canPrevPage={pageStack.length > 0}
        />
      )}

      <UploadFileDialog
        open={showUpload}
        onClose={() => setShowUpload(false)}
        onUploaded={fetchFiles}
      />
    </div>
  );
}
