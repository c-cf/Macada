"use client";

import { useCallback, useEffect, useState } from "react";
import { getUsage } from "@/lib/api";
import type { UsageDayData } from "@/lib/types";
import { ErrorMessage } from "@/components/error-message";

// Approximate cost per million tokens (placeholder rates)
const COST_PER_MILLION_INPUT = 3.0;
const COST_PER_MILLION_OUTPUT = 15.0;
const COST_PER_MILLION_CACHE_READ = 0.3;
const COST_PER_MILLION_CACHE_CREATION = 3.75;

function formatDateRange(year: number, month: number) {
  const from = `${year}-${String(month + 1).padStart(2, "0")}-01`;
  const lastDay = new Date(year, month + 1, 0).getDate();
  const to = `${year}-${String(month + 1).padStart(2, "0")}-${String(lastDay).padStart(2, "0")}`;
  return { from, to };
}

function formatMonth(year: number, month: number): string {
  const date = new Date(year, month);
  return date.toLocaleDateString("en-US", { month: "long", year: "numeric" });
}

function formatCost(cents: number): string {
  if (cents < 0.01) return "$0.00";
  return `$${cents.toFixed(2)}`;
}

function calculateCost(data: readonly UsageDayData[]): {
  inputCost: number;
  outputCost: number;
  cacheReadCost: number;
  cacheCreationCost: number;
  totalCost: number;
} {
  let totalInput = 0;
  let totalOutput = 0;
  let totalCacheRead = 0;
  let totalCacheCreation = 0;

  for (const row of data) {
    totalInput += row.input_tokens;
    totalOutput += row.output_tokens;
    totalCacheRead += row.cache_read_tokens;
    totalCacheCreation += row.cache_creation_tokens;
  }

  const inputCost = (totalInput / 1_000_000) * COST_PER_MILLION_INPUT;
  const outputCost = (totalOutput / 1_000_000) * COST_PER_MILLION_OUTPUT;
  const cacheReadCost =
    (totalCacheRead / 1_000_000) * COST_PER_MILLION_CACHE_READ;
  const cacheCreationCost =
    (totalCacheCreation / 1_000_000) * COST_PER_MILLION_CACHE_CREATION;
  const totalCost = inputCost + outputCost + cacheReadCost + cacheCreationCost;

  return { inputCost, outputCost, cacheReadCost, cacheCreationCost, totalCost };
}

export default function CostPage() {
  const now = new Date();
  const [year, setYear] = useState(now.getFullYear());
  const [month, setMonth] = useState(now.getMonth());
  const [data, setData] = useState<readonly UsageDayData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { from, to } = formatDateRange(year, month);
      const res = await getUsage({ from, to });
      setData(res.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load cost data");
    } finally {
      setLoading(false);
    }
  }, [year, month]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const costs = calculateCost(data);

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
      {/* Month navigator */}
      <div className="flex items-center gap-3">
        <select className="px-3 py-2 text-sm bg-card border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-ring/30">
          <option>All Workspaces</option>
        </select>

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

      {error ? (
        <ErrorMessage message={error} onRetry={fetchData} />
      ) : loading ? (
        <div className="bg-card rounded-xl border border-border p-12 text-center">
          <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
          <p className="mt-3 text-sm text-muted-foreground">Loading cost data...</p>
        </div>
      ) : (
        <>
          {/* Total cost card */}
          <div className="bg-card rounded-xl border border-border p-6">
            <p className="text-sm text-muted-foreground mb-1">
              Estimated total cost
            </p>
            <p className="text-4xl font-semibold text-foreground">
              {formatCost(costs.totalCost)}
            </p>
            <p className="text-xs text-muted-foreground mt-2">
              Based on approximate token pricing. Actual billing may differ.
            </p>
          </div>

          {/* Cost breakdown */}
          <div className="bg-card rounded-xl border border-border overflow-hidden">
            <div className="px-6 py-4 border-b border-border">
              <h2 className="text-lg font-semibold text-foreground">
                Cost breakdown
              </h2>
            </div>
            <table className="w-full">
              <thead>
                <tr className="border-b border-border">
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Category
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Rate (per 1M tokens)
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Estimated cost
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-border">
                  <td className="px-6 py-3.5 text-sm text-foreground">
                    Input tokens
                  </td>
                  <td className="px-6 py-3.5 text-sm text-muted-foreground text-right">
                    ${COST_PER_MILLION_INPUT.toFixed(2)}
                  </td>
                  <td className="px-6 py-3.5 text-sm text-foreground text-right font-medium">
                    {formatCost(costs.inputCost)}
                  </td>
                </tr>
                <tr className="border-b border-border">
                  <td className="px-6 py-3.5 text-sm text-foreground">
                    Output tokens
                  </td>
                  <td className="px-6 py-3.5 text-sm text-muted-foreground text-right">
                    ${COST_PER_MILLION_OUTPUT.toFixed(2)}
                  </td>
                  <td className="px-6 py-3.5 text-sm text-foreground text-right font-medium">
                    {formatCost(costs.outputCost)}
                  </td>
                </tr>
                <tr className="border-b border-border">
                  <td className="px-6 py-3.5 text-sm text-foreground">
                    Cache read tokens
                  </td>
                  <td className="px-6 py-3.5 text-sm text-muted-foreground text-right">
                    ${COST_PER_MILLION_CACHE_READ.toFixed(2)}
                  </td>
                  <td className="px-6 py-3.5 text-sm text-foreground text-right font-medium">
                    {formatCost(costs.cacheReadCost)}
                  </td>
                </tr>
                <tr>
                  <td className="px-6 py-3.5 text-sm text-foreground">
                    Cache creation tokens
                  </td>
                  <td className="px-6 py-3.5 text-sm text-muted-foreground text-right">
                    ${COST_PER_MILLION_CACHE_CREATION.toFixed(2)}
                  </td>
                  <td className="px-6 py-3.5 text-sm text-foreground text-right font-medium">
                    {formatCost(costs.cacheCreationCost)}
                  </td>
                </tr>
              </tbody>
              <tfoot>
                <tr className="border-t border-border bg-muted">
                  <td className="px-6 py-3.5 text-sm font-semibold text-foreground">
                    Total
                  </td>
                  <td />
                  <td className="px-6 py-3.5 text-sm font-semibold text-foreground text-right">
                    {formatCost(costs.totalCost)}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>

          {/* Coming soon note */}
          <div className="bg-muted rounded-xl border border-border p-5 text-center">
            <p className="text-sm text-muted-foreground">
              Detailed per-model and per-agent cost breakdowns are coming soon.
            </p>
          </div>
        </>
      )}
    </div>
  );
}
