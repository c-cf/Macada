"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getUsage } from "@/lib/api";
import type { UsageDayData, UsageSummary } from "@/lib/types";
import { ErrorMessage } from "@/components/error-message";
import * as XLSX from "xlsx";

function formatMonth(year: number, month: number): string {
  const date = new Date(year, month);
  return date.toLocaleDateString("en-US", { month: "long", year: "numeric" });
}

function formatDateRange(year: number, month: number) {
  const from = `${year}-${String(month + 1).padStart(2, "0")}-01`;
  const lastDay = new Date(year, month + 1, 0).getDate();
  const to = `${year}-${String(month + 1).padStart(2, "0")}-${String(lastDay).padStart(2, "0")}`;
  return { from, to };
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(0)}k`;
  return n.toLocaleString();
}

function formatAxisLabel(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return n.toLocaleString();
  return String(n);
}

function formatDayLabel(dateStr: string): string {
  const date = new Date(dateStr + "T00:00:00");
  return date.toLocaleDateString("en-US", { month: "short", day: "2-digit" });
}

interface AggregatedDay {
  readonly day: string;
  readonly totalTokens: number;
}

function aggregateByDay(data: readonly UsageDayData[]): readonly AggregatedDay[] {
  const map = new Map<string, number>();
  for (const row of data) {
    const existing = map.get(row.day) ?? 0;
    map.set(row.day, existing + row.input_tokens + row.output_tokens);
  }
  return Array.from(map.entries())
    .map(([day, totalTokens]) => ({ day, totalTokens }))
    .sort((a, b) => a.day.localeCompare(b.day));
}

function buildExportRows(data: readonly UsageDayData[]) {
  return data.map((row) => ({
    Day: row.day,
    Model: row.model,
    "Input Tokens": row.input_tokens,
    "Output Tokens": row.output_tokens,
    "Cache Read Tokens": row.cache_read_tokens,
    "Cache Creation Tokens": row.cache_creation_tokens,
    Requests: row.request_count,
  }));
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function exportCSV(data: readonly UsageDayData[], filename: string) {
  const rows = buildExportRows(data);
  if (rows.length === 0) return;
  const headers = Object.keys(rows[0]);
  const lines = [
    headers.join(","),
    ...rows.map((row) =>
      headers.map((h) => String(row[h as keyof typeof row])).join(",")
    ),
  ];
  const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8" });
  downloadBlob(blob, filename);
}

function exportExcel(data: readonly UsageDayData[], filename: string) {
  const rows = buildExportRows(data);
  const ws = XLSX.utils.json_to_sheet(rows);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, "Usage");
  const buf = XLSX.write(wb, { type: "array", bookType: "xlsx" });
  const blob = new Blob([buf], {
    type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  });
  downloadBlob(blob, filename);
}

function ExportDropdown({
  data,
  year,
  month,
}: {
  readonly data: readonly UsageDayData[];
  readonly year: number;
  readonly month: number;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  const prefix = `usage-${year}-${String(month + 1).padStart(2, "0")}`;

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="px-4 py-2 text-sm bg-card border border-border rounded-lg text-foreground hover:bg-muted transition-colors"
      >
        Export
      </button>
      {open && (
        <div className="absolute right-0 mt-1 w-40 bg-card border border-border rounded-lg shadow-lg z-50 overflow-hidden">
          <button
            onClick={() => {
              exportCSV(data, `${prefix}.csv`);
              setOpen(false);
            }}
            className="w-full px-4 py-2.5 text-sm text-left text-foreground hover:bg-muted transition-colors"
          >
            Export as CSV
          </button>
          <button
            onClick={() => {
              exportExcel(data, `${prefix}.xlsx`);
              setOpen(false);
            }}
            className="w-full px-4 py-2.5 text-sm text-left text-foreground hover:bg-muted transition-colors"
          >
            Export as Excel
          </button>
        </div>
      )}
    </div>
  );
}

function BarChart({ data }: { readonly data: readonly AggregatedDay[] }) {
  const maxVal = useMemo(
    () => Math.max(...data.map((d) => d.totalTokens), 1),
    [data]
  );

  // Generate y-axis ticks
  const yTicks = useMemo(() => {
    const step = Math.ceil(maxVal / 4 / 1000) * 1000;
    const ticks: number[] = [0];
    let current = step;
    while (current <= maxVal * 1.1) {
      ticks.push(current);
      current += step;
    }
    if (ticks.length < 2) ticks.push(maxVal);
    return ticks;
  }, [maxVal]);

  const chartMax = yTicks[yTicks.length - 1] || 1;

  if (data.length === 0) {
    return (
      <div className="h-64 flex items-center justify-center text-sm text-muted-foreground">
        No usage data for this period
      </div>
    );
  }

  return (
    <div className="flex h-64">
      {/* Y-axis labels */}
      <div className="flex flex-col justify-between pr-3 py-1 text-right w-20 shrink-0">
        {[...yTicks].reverse().map((tick) => (
          <span key={tick} className="text-xs text-muted-foreground leading-none">
            {formatAxisLabel(tick)}
          </span>
        ))}
      </div>

      {/* Chart area */}
      <div className="flex-1 flex flex-col">
        <div className="flex-1 relative border-l border-b border-border">
          {/* Horizontal grid lines */}
          {yTicks.slice(1).map((tick) => (
            <div
              key={tick}
              className="absolute left-0 right-0 border-t border-border"
              style={{ bottom: `${(tick / chartMax) * 100}%` }}
            />
          ))}

          {/* Bars */}
          <div className="absolute inset-0 flex gap-px px-1">
            {data.map((d) => {
              const height = (d.totalTokens / chartMax) * 100;
              return (
                <div
                  key={d.day}
                  className="flex-1 flex flex-col items-center justify-end group relative"
                >
                  <div
                    className="w-full max-w-8 rounded-t bg-primary/70 hover:bg-primary/85 transition-colors"
                    style={{ height: `${Math.max(height, 0.5)}%` }}
                    title={`${formatDayLabel(d.day)}: ${d.totalTokens.toLocaleString()} tokens`}
                  />
                </div>
              );
            })}
          </div>
        </div>

        {/* X-axis labels */}
        <div className="flex gap-px px-1 mt-2">
          {data.map((d, i) => {
            // Show every Nth label to avoid overcrowding
            const showLabel =
              data.length <= 15 || i % Math.ceil(data.length / 10) === 0;
            return (
              <div key={d.day} className="flex-1 text-center">
                {showLabel && (
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {formatDayLabel(d.day)}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default function UsagePage() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth());
  const [model, setModel] = useState("");
  const [data, setData] = useState<readonly UsageDayData[]>([]);
  const [summary, setSummary] = useState<UsageSummary>({
    total_input: 0,
    total_output: 0,
    total_requests: 0,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchUsage = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { from, to } = formatDateRange(year, month);
      const res = await getUsage({ from, to, model: model || undefined });
      setData(res.data);
      setSummary(res.summary);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load usage data");
    } finally {
      setLoading(false);
    }
  }, [year, month, model]);

  useEffect(() => {
    fetchUsage();
  }, [fetchUsage]);

  const aggregated = useMemo(() => aggregateByDay(data), [data]);

  const handlePrevMonth = () => {
    if (month === 0) {
      setMonth(11);
      setYear((y) => y - 1);
    } else {
      setMonth((m) => m - 1);
    }
  };

  const handleNextMonth = () => {
    if (month === 11) {
      setMonth(0);
      setYear((y) => y + 1);
    } else {
      setMonth((m) => m + 1);
    }
  };

  return (
    <div className="space-y-6">
      {/* Filter row */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-3">
          <select className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30">
            <option>All Workspaces</option>
          </select>

          {/* Month navigator */}
          <div className="flex items-center gap-1">
            <button
              onClick={handlePrevMonth}
              className="p-1.5 rounded-md border border-border text-muted-foreground hover:bg-muted transition-colors"
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
                  d="M15 19l-7-7 7-7"
                />
              </svg>
            </button>
            <span className="px-3 py-1.5 text-sm font-medium text-foreground min-w-36 text-center">
              {formatMonth(year, month)}
            </span>
            <button
              onClick={handleNextMonth}
              className="p-1.5 rounded-md border border-border text-muted-foreground hover:bg-muted transition-colors"
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
                  d="M9 5l7 7-7 7"
                />
              </svg>
            </button>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <select className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30">
            <option>All API keys</option>
          </select>

          <select
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30"
          >
            <option value="">All Models</option>
            <option value="claude-sonnet-4-6">claude-sonnet-4-6</option>
            <option value="claude-opus-4-6">claude-opus-4-6</option>
            <option value="claude-haiku-4-5">claude-haiku-4-5</option>
          </select>

          <select className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30">
            <option>Group by: Day</option>
            <option>Group by: Week</option>
          </select>

          <ExportDropdown data={data} year={year} month={month} />
        </div>
      </div>

      {error ? (
        <ErrorMessage message={error} onRetry={fetchUsage} />
      ) : loading ? (
        <div className="bg-card rounded-xl border border-border p-12 text-center">
          <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
          <p className="mt-3 text-sm text-muted-foreground">Loading usage data...</p>
        </div>
      ) : (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="bg-card rounded-xl border border-border p-5">
              <p className="text-sm text-muted-foreground mb-1">Total tokens in</p>
              <p className="text-3xl font-semibold text-foreground">
                {formatNumber(summary.total_input)}
              </p>
            </div>
            <div className="bg-card rounded-xl border border-border p-5">
              <p className="text-sm text-muted-foreground mb-1">Total tokens out</p>
              <p className="text-3xl font-semibold text-foreground">
                {formatNumber(summary.total_output)}
              </p>
            </div>
            <div className="bg-card rounded-xl border border-border p-5">
              <p className="text-sm text-muted-foreground mb-1">Total web searches</p>
              <p className="text-3xl font-semibold text-foreground">0</p>
            </div>
          </div>

          {/* Bar chart */}
          <div className="bg-card rounded-xl border border-border p-6">
            <div className="mb-4">
              <h2 className="text-lg font-semibold text-foreground">
                Token usage
              </h2>
              <p className="text-sm text-muted-foreground">
                Includes usage from both API and Console
              </p>
            </div>
            <BarChart data={aggregated} />
          </div>
        </>
      )}
    </div>
  );
}
