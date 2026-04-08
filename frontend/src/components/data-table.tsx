"use client";

import { ReactNode } from "react";

export interface Column<T> {
  readonly key: string;
  readonly header: string;
  readonly render: (item: T) => ReactNode;
  readonly className?: string;
}

interface DataTableProps<T> {
  readonly columns: readonly Column<T>[];
  readonly data: readonly T[];
  readonly keyFn: (item: T) => string;
  readonly onRowClick?: (item: T) => void;
  readonly loading?: boolean;
  readonly emptyMessage?: string;
  readonly nextPage?: string | null;
  readonly onNextPage?: () => void;
  readonly onPrevPage?: () => void;
  readonly canPrevPage?: boolean;
}

export function DataTable<T>({
  columns,
  data,
  keyFn,
  onRowClick,
  loading,
  emptyMessage = "No data found",
  nextPage,
  onNextPage,
  onPrevPage,
  canPrevPage,
}: DataTableProps<T>) {
  if (loading) {
    return (
      <div className="bg-card rounded-xl border border-border p-12 text-center">
        <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
        <p className="mt-3 text-sm text-muted-foreground">Loading...</p>
      </div>
    );
  }

  return (
    <div className="bg-card rounded-xl border border-border overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-border">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-5 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider ${col.className ?? ""}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-5 py-12 text-center text-sm text-muted-foreground"
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              data.map((item) => (
                <tr
                  key={keyFn(item)}
                  onClick={() => onRowClick?.(item)}
                  className={`border-b border-border last:border-0 ${
                    onRowClick
                      ? "cursor-pointer hover:bg-muted transition-colors"
                      : ""
                  }`}
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      className={`px-5 py-3.5 text-sm text-foreground ${col.className ?? ""}`}
                    >
                      {col.render(item)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {(canPrevPage || nextPage) && (
        <div className="flex items-center justify-between px-5 py-3 border-t border-border">
          <span className="text-xs text-muted-foreground">
            {data.length} item{data.length !== 1 ? "s" : ""}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={onPrevPage}
              disabled={!canPrevPage}
              className="p-1.5 rounded-md border border-border text-muted-foreground hover:bg-muted disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
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
            <button
              onClick={onNextPage}
              disabled={!nextPage}
              className="p-1.5 rounded-md border border-border text-muted-foreground hover:bg-muted disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
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
      )}
    </div>
  );
}
