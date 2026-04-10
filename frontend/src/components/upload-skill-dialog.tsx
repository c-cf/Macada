"use client";

import { useRef, useState } from "react";
import { createSkillFromMarkdown, createSkillFromZip } from "@/lib/api";

interface UploadSkillDialogProps {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onUploaded: () => void;
}

export function UploadSkillDialog({
  open,
  onClose,
  onUploaded,
}: UploadSkillDialogProps) {
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!open) return null;

  function handleClose() {
    setError(null);
    setDragOver(false);
    onClose();
  }

  async function processFile(file: File) {
    setUploading(true);
    setError(null);
    try {
      if (file.name.endsWith(".zip")) {
        await createSkillFromZip(file);
      } else if (file.name.endsWith(".md")) {
        const content = await file.text();
        await createSkillFromMarkdown(content);
      } else {
        setError("Only .md and .zip files are supported");
        setUploading(false);
        return;
      }
      onUploaded();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) processFile(file);
    // reset so same file can be selected again
    e.target.value = "";
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) processFile(file);
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(true);
  }

  function handleDragLeave() {
    setDragOver(false);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <div>
            <h2 className="text-base font-semibold text-foreground">
              Upload Skill
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              Upload a <code className="text-xs">.md</code> or{" "}
              <code className="text-xs">.zip</code> skill package
            </p>
          </div>
          <button
            onClick={handleClose}
            className="p-1.5 rounded-md text-muted-foreground hover:bg-muted transition-colors"
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
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Drop zone */}
        <div className="px-6 py-6">
          <div
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onClick={() => fileInputRef.current?.click()}
            className={`relative flex flex-col items-center justify-center gap-3 border-2 border-dashed rounded-lg py-10 cursor-pointer transition-colors ${
              dragOver
                ? "border-primary bg-primary/5"
                : "border-border hover:border-primary/50 hover:bg-muted/30"
            }`}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".md,.zip"
              onChange={handleFileChange}
              className="sr-only"
            />

            {uploading ? (
              <>
                <svg
                  className="w-8 h-8 text-primary animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                  />
                </svg>
                <p className="text-sm text-muted-foreground">Uploading...</p>
              </>
            ) : (
              <>
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                  <svg
                    className="w-6 h-6 text-muted-foreground"
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
                </div>
                <div className="text-center">
                  <p className="text-sm font-medium text-foreground">
                    Drop file here or{" "}
                    <span className="text-primary underline">browse</span>
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Accepts <strong>.md</strong> (skill markdown) or{" "}
                    <strong>.zip</strong> (skill package)
                  </p>
                </div>
              </>
            )}
          </div>

          {error && (
            <p className="mt-3 text-sm text-red-500 text-center">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-border flex justify-end">
          <button
            onClick={handleClose}
            disabled={uploading}
            className="px-4 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
