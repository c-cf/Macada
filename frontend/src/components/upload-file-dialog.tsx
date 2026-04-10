"use client";

import { useState, useRef } from "react";
import { uploadFile } from "@/lib/api";
import { formatBytes } from "@/lib/utils";

interface UploadFileDialogProps {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onUploaded: () => void;
}

const MAX_FILE_SIZE = 500 * 1024 * 1024; // 500MB

export function UploadFileDialog({
  open,
  onClose,
  onUploaded,
}: UploadFileDialogProps) {
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  if (!open) return null;

  function handleClose() {
    setFile(null);
    setError(null);
    setUploading(false);
    setDragOver(false);
    onClose();
  }

  function handleFileSelect(selectedFile: File) {
    if (selectedFile.size > MAX_FILE_SIZE) {
      setError(`File too large (${formatBytes(selectedFile.size)}). Maximum is 500 MB.`);
      return;
    }
    setError(null);
    setFile(selectedFile);
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault();
    setDragOver(false);
    const dropped = e.dataTransfer.files[0];
    if (dropped) {
      handleFileSelect(dropped);
    }
  }

  async function handleUpload() {
    if (!file) return;
    setUploading(true);
    setError(null);
    try {
      await uploadFile(file);
      handleClose();
      onUploaded();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={handleClose}
      />
      <div className="relative bg-card border border-border rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-lg font-semibold text-foreground mb-4">
          Upload file
        </h2>

        {/* Drop zone */}
        <div
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
          onClick={() => inputRef.current?.click()}
          className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
            dragOver
              ? "border-primary bg-primary/5"
              : "border-border hover:border-muted-foreground"
          }`}
        >
          <input
            ref={inputRef}
            type="file"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) handleFileSelect(f);
            }}
          />
          <svg
            className="w-8 h-8 mx-auto mb-2 text-muted-foreground"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
            />
          </svg>
          {file ? (
            <div>
              <p className="text-sm font-medium text-foreground">
                {file.name}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                {formatBytes(file.size)}
              </p>
            </div>
          ) : (
            <div>
              <p className="text-sm text-muted-foreground">
                Drop a file here or click to browse
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                Max 500 MB
              </p>
            </div>
          )}
        </div>

        {error && (
          <p className="mt-3 text-sm text-red-500">{error}</p>
        )}

        <div className="flex justify-end gap-3 mt-6">
          <button
            onClick={handleClose}
            className="px-4 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleUpload}
            disabled={!file || uploading}
            className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {uploading ? "Uploading..." : "Upload"}
          </button>
        </div>
      </div>
    </div>
  );
}
