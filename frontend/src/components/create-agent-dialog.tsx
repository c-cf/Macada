"use client";

import { useState } from "react";
import yaml from "js-yaml";

type ConfigFormat = "yaml" | "json";

const BLANK_TEMPLATE = {
  name: "Untitled agent",
  description: "A blank starting point with the core toolset.",
  model: "claude-sonnet-4-6",
  system:
    "You are a general-purpose agent that can research, write code, run commands, and use connected tools to complete the user's task end to end.",
  mcp_servers: [],
  tools: [{ type: "agent_toolset_20260401" }],
  skills: [],
};

function toYaml(obj: unknown): string {
  return yaml.dump(obj, {
    indent: 2,
    lineWidth: -1,
    noRefs: true,
    quotingType: '"',
  });
}

function toJson(obj: unknown): string {
  return JSON.stringify(obj, null, 2);
}

function parseConfig(text: string, format: ConfigFormat): unknown {
  if (format === "yaml") {
    return yaml.load(text);
  }
  return JSON.parse(text);
}

interface CreateAgentDialogProps {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onCreated: () => void;
}

export function CreateAgentDialog({
  open,
  onClose,
  onCreated,
}: CreateAgentDialogProps) {
  const [format, setFormat] = useState<ConfigFormat>("yaml");
  const [configText, setConfigText] = useState(toYaml(BLANK_TEMPLATE));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const switchFormat = (newFormat: ConfigFormat) => {
    if (newFormat === format) return;

    try {
      const parsed = parseConfig(configText, format);
      const converted =
        newFormat === "yaml" ? toYaml(parsed) : toJson(parsed);
      setConfigText(converted);
      setFormat(newFormat);
      setError(null);
    } catch {
      setError(
        `Cannot convert: current ${format.toUpperCase()} is invalid. Fix the syntax first.`
      );
    }
  };

  const handleCopy = async () => {
    await navigator.clipboard.writeText(configText);
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);

    try {
      const parsed = parseConfig(configText, format);
      if (!parsed || typeof parsed !== "object") {
        throw new Error("Config must be an object");
      }

      const body = parsed as Record<string, unknown>;
      if (!body.name || typeof body.name !== "string" || !body.name.trim()) {
        throw new Error("Agent name is required");
      }
      if (!body.model) {
        throw new Error("Model is required");
      }

      const { createAgent } = await import("@/lib/api");
      await createAgent(body);

      setConfigText(toYaml(BLANK_TEMPLATE));
      setFormat("yaml");
      onCreated();
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create agent"
      );
    } finally {
      setSubmitting(false);
    }
  };

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const handleClose = () => {
    setError(null);
    onClose();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={handleBackdropClick}
    >
      <div className="bg-card rounded-xl shadow-xl w-full max-w-2xl max-h-[90vh] flex flex-col mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 pt-6 pb-4 shrink-0">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              Create agent
            </h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              Start from a template or describe what you need.
            </p>
          </div>
          <button
            onClick={handleClose}
            className="p-1.5 text-muted-foreground hover:text-muted-foreground hover:bg-muted rounded-lg transition-colors"
          >
            <svg
              className="w-5 h-5"
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

        <div className="px-6 pb-6 flex flex-col gap-4 overflow-hidden flex-1">
          {/* Starting point label */}
          <div className="flex items-center gap-2 text-sm">
            <span className="font-medium text-foreground">Starting point</span>
            <span className="text-muted-foreground">&middot;</span>
            <span className="text-muted-foreground">Blank agent</span>
          </div>

          {/* Agent config */}
          <div>
            <h3 className="text-sm font-medium text-foreground mb-2">
              Agent config
            </h3>
            <div className="border border-border rounded-xl overflow-hidden flex flex-col">
              {/* Format toggle + copy */}
              <div className="flex items-center justify-between bg-muted border-b border-border px-3 py-1.5">
                <div className="flex">
                  <button
                    onClick={() => switchFormat("yaml")}
                    className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                      format === "yaml"
                        ? "bg-card text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    YAML
                  </button>
                  <button
                    onClick={() => switchFormat("json")}
                    className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                      format === "json"
                        ? "bg-card text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    JSON
                  </button>
                </div>
                <button
                  onClick={handleCopy}
                  className="p-1 text-muted-foreground hover:text-muted-foreground transition-colors"
                  title="Copy to clipboard"
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
                      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                    />
                  </svg>
                </button>
              </div>

              {/* Editor */}
              <textarea
                value={configText}
                onChange={(e) => {
                  setConfigText(e.target.value);
                  setError(null);
                }}
                spellCheck={false}
                className="w-full px-4 py-3 text-sm font-mono text-foreground bg-card focus:outline-none resize-none min-h-[320px] max-h-[50vh]"
              />
            </div>
          </div>

          {/* Error */}
          {error && (
            <div className="px-3 py-2 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-1 shrink-0">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={submitting}
              className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "Creating..." : "Create agent"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
