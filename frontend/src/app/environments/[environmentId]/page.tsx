"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getEnvironment, updateEnvironment } from "@/lib/api";
import type { Environment } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { timeAgo } from "@/lib/utils";

type Tab = "overview" | "networking" | "packages";
type NetworkingType = "unrestricted" | "limited";
type PackageManager = "apt" | "cargo" | "gem" | "go" | "npm" | "pip";

const PACKAGE_MANAGERS: readonly PackageManager[] = [
  "apt",
  "cargo",
  "gem",
  "go",
  "npm",
  "pip",
];

interface NetworkingForm {
  readonly type: NetworkingType;
  readonly allowMcpServers: boolean;
  readonly allowPackageManagers: boolean;
  readonly allowedHosts: string;
}

interface PackageEntry {
  readonly manager: PackageManager;
  readonly packages: string;
}

function buildNetworkingForm(env: Environment): NetworkingForm {
  const net = env.config?.networking;
  if (!net) {
    return {
      type: "limited",
      allowMcpServers: false,
      allowPackageManagers: false,
      allowedHosts: "",
    };
  }
  return {
    type: (net.type as NetworkingType) ?? "limited",
    allowMcpServers: net.allow_mcp_servers ?? false,
    allowPackageManagers: net.allow_package_managers ?? false,
    allowedHosts: (net.allowed_hosts ?? []).join(", "),
  };
}

function buildPackageEntries(env: Environment): readonly PackageEntry[] {
  const pkgs = env.config?.packages;
  if (!pkgs) return [];
  const entries: PackageEntry[] = [];
  for (const pm of PACKAGE_MANAGERS) {
    const list = pkgs[pm as keyof typeof pkgs];
    if (Array.isArray(list) && list.length > 0) {
      entries.push({ manager: pm, packages: list.join(" ") });
    }
  }
  return entries;
}

function ToggleSwitch({
  checked,
  onChange,
}: {
  readonly checked: boolean;
  readonly onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
        checked ? "bg-primary" : "bg-slate-300"
      }`}
    >
      <span
        className={`inline-block h-3.5 w-3.5 rounded-full bg-card transition-transform ${
          checked ? "translate-x-4.5" : "translate-x-0.5"
        }`}
      />
    </button>
  );
}

export default function EnvironmentDetailPage() {
  const params = useParams();
  const environmentId = params.environmentId as string;

  const [env, setEnv] = useState<Environment | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("overview");

  // Edit states
  const [editingName, setEditingName] = useState(false);
  const [nameValue, setNameValue] = useState("");
  const [editingDesc, setEditingDesc] = useState(false);
  const [descValue, setDescValue] = useState("");

  // Networking edit
  const [editingNetworking, setEditingNetworking] = useState(false);
  const [networkingForm, setNetworkingForm] = useState<NetworkingForm>({
    type: "limited",
    allowMcpServers: false,
    allowPackageManagers: false,
    allowedHosts: "",
  });

  // Packages edit
  const [editingPackages, setEditingPackages] = useState(false);
  const [packageEntries, setPackageEntries] = useState<readonly PackageEntry[]>([]);

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getEnvironment(environmentId);
      setEnv(data);
      setNameValue(data.name);
      setDescValue(data.description ?? "");
      setNetworkingForm(buildNetworkingForm(data));
      setPackageEntries(buildPackageEntries(data));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load environment"
      );
    } finally {
      setLoading(false);
    }
  }, [environmentId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const saveField = async (body: Parameters<typeof updateEnvironment>[1]) => {
    setSaving(true);
    setSaveError(null);
    try {
      const updated = await updateEnvironment(environmentId, body);
      setEnv(updated);
      setNameValue(updated.name);
      setDescValue(updated.description ?? "");
      setNetworkingForm(buildNetworkingForm(updated));
      setPackageEntries(buildPackageEntries(updated));
      return true;
    } catch (err) {
      setSaveError(
        err instanceof Error ? err.message : "Failed to save changes"
      );
      return false;
    } finally {
      setSaving(false);
    }
  };

  const handleSaveName = async () => {
    const ok = await saveField({ name: nameValue.trim() });
    if (ok) setEditingName(false);
  };

  const handleSaveDesc = async () => {
    const ok = await saveField({ description: descValue.trim() });
    if (ok) setEditingDesc(false);
  };

  const handleSaveNetworking = async () => {
    const networking =
      networkingForm.type === "unrestricted"
        ? { type: "unrestricted" as const }
        : {
            type: "limited" as const,
            allow_mcp_servers: networkingForm.allowMcpServers,
            allow_package_managers: networkingForm.allowPackageManagers,
            ...(networkingForm.allowedHosts.trim()
              ? {
                  allowed_hosts: networkingForm.allowedHosts
                    .split(",")
                    .map((h) => h.trim())
                    .filter(Boolean),
                }
              : { allowed_hosts: [] }),
          };

    const ok = await saveField({ config: { type: "cloud", networking } });
    if (ok) setEditingNetworking(false);
  };

  const addPackageEntry = () => {
    setPackageEntries((prev) => [
      ...prev,
      { manager: "pip" as PackageManager, packages: "" },
    ]);
  };

  const removePackageEntry = (index: number) => {
    setPackageEntries((prev) => prev.filter((_, i) => i !== index));
  };

  const updatePackageEntry = (
    index: number,
    field: keyof PackageEntry,
    value: string
  ) => {
    setPackageEntries((prev) =>
      prev.map((entry, i) =>
        i === index ? { ...entry, [field]: value } : entry
      )
    );
  };

  const handleSavePackages = async () => {
    const packages: Record<string, string[]> = {};
    for (const pm of PACKAGE_MANAGERS) {
      packages[pm] = [];
    }
    for (const entry of packageEntries) {
      const pkgs = entry.packages
        .split(/[\s,]+/)
        .map((p) => p.trim())
        .filter(Boolean);
      if (pkgs.length > 0) {
        packages[entry.manager] = [
          ...packages[entry.manager],
          ...pkgs,
        ];
      }
    }

    const ok = await saveField({ config: { type: "cloud", packages } });
    if (ok) setEditingPackages(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
      </div>
    );
  }

  if (error || !env) {
    return (
      <ErrorMessage
        message={error ?? "Environment not found"}
        onRetry={fetchData}
      />
    );
  }

  const status = env.archived_at ? "Archived" : "Active";
  const networking = env.config?.networking;
  const packages = env.config?.packages;

  const TABS: readonly { key: Tab; label: string }[] = [
    { key: "overview", label: "Overview" },
    { key: "networking", label: "Networking" },
    { key: "packages", label: "Packages" },
  ];

  return (
    <div>
      <PageHeader
        title={env.name || "Untitled"}
        subtitle={`${env.id} \u00B7 Last updated ${timeAgo(env.updated_at)}`}
        breadcrumbs={[
          { label: "Environments", href: "/environments" },
          { label: env.name || "Untitled" },
        ]}
      />

      <div className="flex items-center gap-2 mb-6">
        <StatusBadge status={status} />
        <span className="text-xs font-medium px-2.5 py-1 bg-muted text-muted-foreground rounded-full">
          {env.config?.type ?? "cloud"}
        </span>
        {env.description && (
          <span className="text-sm text-muted-foreground ml-2">
            {env.description}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="border-b border-border mb-6">
        <div className="flex gap-0">
          {TABS.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.key
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      {/* Save error */}
      {saveError && (
        <div className="mb-4 px-3 py-2 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
          {saveError}
        </div>
      )}

      {/* Overview tab */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          {/* Name */}
          <div className="bg-card rounded-xl border border-border p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">Name</h3>
              {!editingName && (
                <button
                  onClick={() => setEditingName(true)}
                  className="text-xs text-primary hover:text-primary/80 transition-colors"
                >
                  Edit
                </button>
              )}
            </div>
            {editingName ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={nameValue}
                  onChange={(e) => setNameValue(e.target.value)}
                  maxLength={50}
                  className="flex-1 px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50"
                />
                <button
                  onClick={handleSaveName}
                  disabled={saving || !nameValue.trim()}
                  className="px-3 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 disabled:opacity-50"
                >
                  {saving ? "Saving..." : "Save"}
                </button>
                <button
                  onClick={() => {
                    setEditingName(false);
                    setNameValue(env.name);
                  }}
                  className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground"
                >
                  Cancel
                </button>
              </div>
            ) : (
              <p className="text-sm text-foreground">{env.name || "-"}</p>
            )}
          </div>

          {/* Description */}
          <div className="bg-card rounded-xl border border-border p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-foreground">
                Description
              </h3>
              {!editingDesc && (
                <button
                  onClick={() => setEditingDesc(true)}
                  className="text-xs text-primary hover:text-primary/80 transition-colors"
                >
                  Edit
                </button>
              )}
            </div>
            {editingDesc ? (
              <div className="space-y-2">
                <textarea
                  value={descValue}
                  onChange={(e) => setDescValue(e.target.value)}
                  rows={3}
                  placeholder="Add a description for this environment (optional)"
                  className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 focus:border-primary/50 resize-none"
                />
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleSaveDesc}
                    disabled={saving}
                    className="px-3 py-2 text-sm font-medium text-white bg-primary rounded-lg hover:bg-primary/90 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save"}
                  </button>
                  <button
                    onClick={() => {
                      setEditingDesc(false);
                      setDescValue(env.description ?? "");
                    }}
                    className="px-3 py-2 text-sm text-muted-foreground hover:text-foreground"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {env.description || (
                  <span className="italic text-muted-foreground">No description</span>
                )}
              </p>
            )}
          </div>

          {/* Metadata */}
          <div className="bg-card rounded-xl border border-border p-5">
            <h3 className="text-sm font-medium text-foreground mb-3">
              Details
            </h3>
            <dl className="grid grid-cols-2 gap-y-3 gap-x-8 text-sm">
              <div>
                <dt className="text-muted-foreground">ID</dt>
                <dd className="font-mono text-xs text-foreground mt-0.5">
                  {env.id}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Type</dt>
                <dd className="text-foreground mt-0.5 capitalize">
                  {env.config?.type ?? "-"}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Created</dt>
                <dd className="text-foreground mt-0.5">
                  {new Date(env.created_at).toLocaleString()}
                </dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Updated</dt>
                <dd className="text-foreground mt-0.5">
                  {new Date(env.updated_at).toLocaleString()}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      )}

      {/* Networking tab */}
      {activeTab === "networking" && (
        <div className="space-y-6">
          <div className="bg-card rounded-xl border border-border p-5">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-foreground">
                  Networking
                </h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Configure network access policies for this environment.
                </p>
              </div>
              {!editingNetworking ? (
                <button
                  onClick={() => setEditingNetworking(true)}
                  className="px-3 py-1.5 text-xs font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
                >
                  Edit
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleSaveNetworking}
                    disabled={saving}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-primary rounded-lg hover:bg-primary/90 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save"}
                  </button>
                  <button
                    onClick={() => {
                      setEditingNetworking(false);
                      setNetworkingForm(buildNetworkingForm(env));
                    }}
                    className="px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>

            {editingNetworking ? (
              <div className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                    Type
                  </label>
                  <select
                    value={networkingForm.type}
                    onChange={(e) =>
                      setNetworkingForm({
                        ...networkingForm,
                        type: e.target.value as NetworkingType,
                      })
                    }
                    className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30"
                  >
                    <option value="limited">Limited</option>
                    <option value="unrestricted">Unrestricted</option>
                  </select>
                </div>

                {networkingForm.type === "limited" && (
                  <>
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-foreground">
                        Allow MCP server network access
                      </span>
                      <ToggleSwitch
                        checked={networkingForm.allowMcpServers}
                        onChange={(v) =>
                          setNetworkingForm({
                            ...networkingForm,
                            allowMcpServers: v,
                          })
                        }
                      />
                    </div>

                    <div className="flex items-center justify-between">
                      <span className="text-sm text-foreground">
                        Allow package manager network access
                      </span>
                      <ToggleSwitch
                        checked={networkingForm.allowPackageManagers}
                        onChange={(v) =>
                          setNetworkingForm({
                            ...networkingForm,
                            allowPackageManagers: v,
                          })
                        }
                      />
                    </div>

                    <div>
                      <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                        Allowed Hosts
                      </label>
                      <textarea
                        value={networkingForm.allowedHosts}
                        onChange={(e) =>
                          setNetworkingForm({
                            ...networkingForm,
                            allowedHosts: e.target.value,
                          })
                        }
                        placeholder="www.example1.com, www.example2.com"
                        rows={3}
                        className="w-full px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 resize-none"
                      />
                    </div>
                  </>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center justify-between py-2 border-b border-border">
                  <span className="text-sm text-muted-foreground">Type</span>
                  <span className="text-sm font-medium text-foreground capitalize">
                    {networking?.type ?? "limited"}
                  </span>
                </div>
                {networking?.type === "limited" && (
                  <>
                    <div className="flex items-center justify-between py-2 border-b border-border">
                      <span className="text-sm text-muted-foreground">
                        Allow MCP server network access
                      </span>
                      <span className="text-sm text-foreground">
                        {networking.allow_mcp_servers ? "Yes" : "No"}
                      </span>
                    </div>
                    <div className="flex items-center justify-between py-2 border-b border-border">
                      <span className="text-sm text-muted-foreground">
                        Allow package manager network access
                      </span>
                      <span className="text-sm text-foreground">
                        {networking.allow_package_managers ? "Yes" : "No"}
                      </span>
                    </div>
                    <div className="py-2">
                      <span className="text-sm text-muted-foreground block mb-2">
                        Allowed Hosts
                      </span>
                      {networking.allowed_hosts &&
                      networking.allowed_hosts.length > 0 ? (
                        <div className="flex flex-wrap gap-1.5">
                          {networking.allowed_hosts.map((host) => (
                            <span
                              key={host}
                              className="px-2 py-0.5 text-xs font-mono bg-muted text-muted-foreground border border-border rounded"
                            >
                              {host}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-sm text-muted-foreground italic">
                          None
                        </span>
                      )}
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Packages tab */}
      {activeTab === "packages" && (
        <div className="space-y-6">
          <div className="bg-card rounded-xl border border-border p-5">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-semibold text-foreground">
                  Packages
                </h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Specify packages and their versions available in this
                  environment.
                </p>
              </div>
              {!editingPackages ? (
                <button
                  onClick={() => setEditingPackages(true)}
                  className="px-3 py-1.5 text-xs font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted transition-colors"
                >
                  Edit
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <button
                    onClick={handleSavePackages}
                    disabled={saving}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-primary rounded-lg hover:bg-primary/90 disabled:opacity-50"
                  >
                    {saving ? "Saving..." : "Save"}
                  </button>
                  <button
                    onClick={() => {
                      setEditingPackages(false);
                      setPackageEntries(buildPackageEntries(env));
                    }}
                    className="px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>

            {editingPackages ? (
              <div className="space-y-3">
                {packageEntries.length === 0 && (
                  <p className="text-xs text-muted-foreground italic">
                    No packages configured. Click + to add.
                  </p>
                )}

                {packageEntries.map((entry, index) => (
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
                      className="flex-1 px-3 py-2 text-sm bg-card border border-border rounded-lg placeholder-muted-foreground text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30 font-mono"
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

                <button
                  type="button"
                  onClick={addPackageEntry}
                  className="flex items-center gap-1.5 text-sm text-primary hover:text-primary/80 transition-colors"
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
                  Add package row
                </button>
              </div>
            ) : (
              <div className="space-y-4">
                {PACKAGE_MANAGERS.some((pm) => {
                  const list = packages?.[pm as keyof typeof packages];
                  return Array.isArray(list) && list.length > 0;
                }) ? (
                  PACKAGE_MANAGERS.map((pm) => {
                    const list = packages?.[pm as keyof typeof packages];
                    if (!Array.isArray(list) || list.length === 0) return null;
                    return (
                      <div key={pm}>
                        <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                          {pm}
                        </h4>
                        <div className="flex flex-wrap gap-1.5">
                          {list.map((pkg) => (
                            <span
                              key={pkg}
                              className="px-2 py-0.5 text-xs font-mono bg-muted text-muted-foreground border border-border rounded"
                            >
                              {String(pkg)}
                            </span>
                          ))}
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <p className="text-sm text-muted-foreground italic">
                    No packages configured
                  </p>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
