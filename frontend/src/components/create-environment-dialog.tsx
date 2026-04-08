"use client";

import { useState } from "react";

type NetworkingType = "unrestricted" | "limited";
type PackageManager = "apt" | "cargo" | "gem" | "go" | "npm" | "pip";

interface PackageEntry {
  readonly manager: PackageManager;
  readonly packages: string;
}

interface CreateEnvironmentForm {
  readonly name: string;
  readonly description: string;
  readonly networkingType: NetworkingType;
  readonly allowMcpServers: boolean;
  readonly allowPackageManagers: boolean;
  readonly allowedHosts: string;
  readonly packageEntries: readonly PackageEntry[];
}

interface CreateEnvironmentDialogProps {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onCreated: () => void;
}

const INITIAL_FORM: CreateEnvironmentForm = {
  name: "",
  description: "",
  networkingType: "limited",
  allowMcpServers: false,
  allowPackageManagers: false,
  allowedHosts: "",
  packageEntries: [],
};

const PACKAGE_MANAGERS: readonly PackageManager[] = [
  "apt",
  "cargo",
  "gem",
  "go",
  "npm",
  "pip",
];

function ChevronIcon({ open }: { readonly open: boolean }) {
  return (
    <svg
      className={`w-4 h-4 text-muted-foreground transition-transform ${open ? "rotate-90" : ""}`}
      fill="none"
      stroke="currentColor"
      viewBox="0 0 24 24"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 5l7 7-7 7"
      />
    </svg>
  );
}

export function CreateEnvironmentDialog({
  open,
  onClose,
  onCreated,
}: CreateEnvironmentDialogProps) {
  const [form, setForm] = useState<CreateEnvironmentForm>(INITIAL_FORM);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!open) return null;

  const updateField = <K extends keyof CreateEnvironmentForm>(
    key: K,
    value: CreateEnvironmentForm[K]
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const addPackageEntry = () => {
    updateField("packageEntries", [
      ...form.packageEntries,
      { manager: "pip" as PackageManager, packages: "" },
    ]);
  };

  const removePackageEntry = (index: number) => {
    updateField(
      "packageEntries",
      form.packageEntries.filter((_, i) => i !== index)
    );
  };

  const updatePackageEntry = (
    index: number,
    field: keyof PackageEntry,
    value: string
  ) => {
    updateField(
      "packageEntries",
      form.packageEntries.map((entry, i) =>
        i === index ? { ...entry, [field]: value } : entry
      )
    );
  };

  const handleSubmit = async () => {
    if (!form.name.trim()) return;

    setSubmitting(true);
    setError(null);

    try {
      const { createEnvironment } = await import("@/lib/api");

      const networking =
        form.networkingType === "unrestricted"
          ? { type: "unrestricted" as const }
          : {
              type: "limited" as const,
              allow_mcp_servers: form.allowMcpServers,
              allow_package_managers: form.allowPackageManagers,
              ...(form.allowedHosts.trim()
                ? {
                    allowed_hosts: form.allowedHosts
                      .split(",")
                      .map((h) => h.trim())
                      .filter(Boolean),
                  }
                : {}),
            };

      const packages: Record<string, string[]> = {};
      for (const entry of form.packageEntries) {
        const pkgs = entry.packages
          .split(/[\s,]+/)
          .map((p) => p.trim())
          .filter(Boolean);
        if (pkgs.length > 0) {
          packages[entry.manager] = [
            ...(packages[entry.manager] ?? []),
            ...pkgs,
          ];
        }
      }

      const hasAdvanced =
        showAdvanced &&
        (form.networkingType !== "limited" ||
          form.allowMcpServers ||
          form.allowPackageManagers ||
          form.allowedHosts.trim() ||
          form.packageEntries.length > 0);

      await createEnvironment({
        name: form.name.trim(),
        ...(form.description.trim()
          ? { description: form.description.trim() }
          : {}),
        ...(hasAdvanced
          ? {
              config: {
                type: "cloud",
                networking,
                ...(Object.keys(packages).length > 0 ? { packages } : {}),
              },
            }
          : {}),
      });

      setForm(INITIAL_FORM);
      setShowAdvanced(false);
      onCreated();
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create environment"
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

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={handleBackdropClick}
    >
      <div className="bg-card rounded-xl shadow-xl w-full max-w-lg max-h-[85vh] overflow-y-auto mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 pt-6 pb-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              Add environment
            </h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              Create a sandbox environment for agent execution
            </p>
          </div>
          <button
            onClick={onClose}
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

        <div className="px-6 pb-6 space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">
              Name
            </label>
            <input
              type="text"
              placeholder="E.g. My Environment"
              value={form.name}
              onChange={(e) => updateField("name", e.target.value)}
              maxLength={50}
              className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
            />
            <p className="text-xs text-muted-foreground mt-1">
              50 characters or fewer.
            </p>
          </div>

          {/* Hosting Type - fixed to Cloud */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">
              Hosting Type
            </label>
            <div className="px-3 py-2 text-sm bg-muted border border-border rounded-lg text-muted-foreground">
              Cloud
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              This cannot be changed after creation.
            </p>
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-foreground mb-1.5">
              Description
            </label>
            <textarea
              placeholder="Optional description for this environment"
              value={form.description}
              onChange={(e) => updateField("description", e.target.value)}
              rows={2}
              className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors resize-none"
            />
          </div>

          {/* Advanced Options Toggle */}
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
          >
            <ChevronIcon open={showAdvanced} />
            Advanced options
          </button>

          {/* Advanced Options Content */}
          {showAdvanced && (
            <div className="space-y-5 pl-1">
              {/* Networking */}
              <div className="border border-border rounded-xl p-4 space-y-4">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">
                    Networking
                  </h3>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Configure network access policies for this environment.
                  </p>
                </div>

                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                    Type
                  </label>
                  <select
                    value={form.networkingType}
                    onChange={(e) =>
                      updateField(
                        "networkingType",
                        e.target.value as NetworkingType
                      )
                    }
                    className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                  >
                    <option value="limited">Limited</option>
                    <option value="unrestricted">Unrestricted</option>
                  </select>
                </div>

                {form.networkingType === "limited" && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-foreground">
                        Allow MCP server network access
                      </span>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={form.allowMcpServers}
                        onClick={() =>
                          updateField("allowMcpServers", !form.allowMcpServers)
                        }
                        className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                          form.allowMcpServers ? "bg-primary" : "bg-slate-300"
                        }`}
                      >
                        <span
                          className={`inline-block h-3.5 w-3.5 rounded-full bg-card transition-transform ${
                            form.allowMcpServers
                              ? "translate-x-4.5"
                              : "translate-x-0.5"
                          }`}
                        />
                      </button>
                    </div>

                    <div className="flex items-center justify-between">
                      <span className="text-sm text-foreground">
                        Allow package manager network access
                      </span>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={form.allowPackageManagers}
                        onClick={() =>
                          updateField(
                            "allowPackageManagers",
                            !form.allowPackageManagers
                          )
                        }
                        className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                          form.allowPackageManagers
                            ? "bg-primary"
                            : "bg-slate-300"
                        }`}
                      >
                        <span
                          className={`inline-block h-3.5 w-3.5 rounded-full bg-card transition-transform ${
                            form.allowPackageManagers
                              ? "translate-x-4.5"
                              : "translate-x-0.5"
                          }`}
                        />
                      </button>
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                        Allowed Hosts
                      </label>
                      <textarea
                        placeholder="www.example1.com, www.example2.com"
                        value={form.allowedHosts}
                        onChange={(e) =>
                          updateField("allowedHosts", e.target.value)
                        }
                        rows={2}
                        className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors resize-none"
                      />
                    </div>
                  </>
                )}
              </div>

              {/* Packages */}
              <div className="border border-border rounded-xl p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-semibold text-foreground">
                      Packages
                    </h3>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Specify packages and their versions available in this
                      environment. Separate multiple values with spaces.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={addPackageEntry}
                    className="p-1 text-muted-foreground hover:text-foreground hover:bg-muted rounded transition-colors"
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
                        d="M12 4v16m8-8H4"
                      />
                    </svg>
                  </button>
                </div>

                {form.packageEntries.length === 0 && (
                  <p className="text-xs text-muted-foreground italic">
                    No packages configured. Click + to add.
                  </p>
                )}

                {form.packageEntries.map((entry, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <select
                      value={entry.manager}
                      onChange={(e) =>
                        updatePackageEntry(index, "manager", e.target.value)
                      }
                      className="px-2 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 w-24 shrink-0"
                    >
                      {PACKAGE_MANAGERS.map((pm) => (
                        <option key={pm} value={pm}>
                          {pm}
                        </option>
                      ))}
                    </select>
                    <input
                      type="text"
                      placeholder="package package==1.0.0"
                      value={entry.packages}
                      onChange={(e) =>
                        updatePackageEntry(index, "packages", e.target.value)
                      }
                      className="flex-1 px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 transition-colors"
                    />
                    <button
                      type="button"
                      onClick={() => removePackageEntry(index)}
                      className="p-1.5 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded transition-colors shrink-0"
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
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Error */}
          {error && (
            <div className="px-3 py-2 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={!form.name.trim() || submitting}
              className="px-4 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {submitting ? "Creating..." : "Create"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
