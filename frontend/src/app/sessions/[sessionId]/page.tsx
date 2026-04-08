"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "next/navigation";
import { getSession, listSessionEvents } from "@/lib/api";
import type { Session, SessionEvent } from "@/lib/types";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { ErrorMessage } from "@/components/error-message";
import { timeAgo, formatDuration, formatTokens, truncateId } from "@/lib/utils";

/** Classify event types for display purposes. */
function getEventCategory(type: string): {
  label: string;
  color: string;
  bgColor: string;
} {
  if (type.startsWith("user.")) {
    return {
      label: "User",
      color: "text-info",
      bgColor: "bg-info/10 border-info/20",
    };
  }
  if (type === "agent.message") {
    return {
      label: "Agent",
      color: "text-success",
      bgColor: "bg-success/10 border-success/20",
    };
  }
  if (type === "agent.tool_use" || type === "agent.custom_tool_use") {
    return {
      label: "Tool Use",
      color: "text-violet-700",
      bgColor: "bg-violet-50 border-violet-200",
    };
  }
  if (type === "agent.tool_result") {
    return {
      label: "Tool Result",
      color: "text-warning",
      bgColor: "bg-warning/10 border-warning/20",
    };
  }
  if (type.startsWith("span.")) {
    return {
      label: "Span",
      color: "text-muted-foreground",
      bgColor: "bg-muted border-border",
    };
  }
  if (type.startsWith("session.")) {
    return {
      label: "Session",
      color: "text-muted-foreground",
      bgColor: "bg-muted border-border",
    };
  }
  return {
    label: "Event",
    color: "text-muted-foreground",
    bgColor: "bg-muted border-border",
  };
}

/** Extract display text from an event for the transcript view. */
function getEventDisplayText(event: SessionEvent): string {
  const content = event.content;
  if (typeof content === "string") return content;
  if (Array.isArray(content)) {
    return content
      .map((block: unknown) => {
        if (typeof block === "object" && block !== null) {
          const b = block as Record<string, unknown>;
          if (b.type === "text" && typeof b.text === "string") return b.text;
          if (b.type === "tool_use") return `[Tool: ${b.name ?? "unknown"}]`;
          if (b.type === "tool_result")
            return typeof b.content === "string" ? b.content : "[tool result]";
        }
        return "";
      })
      .filter(Boolean)
      .join("\n");
  }
  if (typeof event.name === "string") return `Tool: ${event.name}`;
  if (typeof event.result === "string") return event.result;
  return "";
}

/** Check if event is a transcript-visible event (not span/session). */
function isTranscriptEvent(event: SessionEvent): boolean {
  return (
    event.type.startsWith("user.") ||
    event.type.startsWith("agent.message") ||
    event.type.startsWith("agent.tool_use") ||
    event.type.startsWith("agent.tool_result") ||
    event.type.startsWith("agent.custom_tool_use")
  );
}

/** Format seconds as a human-friendly short duration (e.g., "9.7s", "1m 12s"). */
function formatShortDuration(seconds: number): string {
  if (seconds < 0.1) return "<0.1s";
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const mins = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  return `${mins}m ${secs}s`;
}

/** Format seconds as an absolute offset timestamp (e.g., "0:00:45"). */
function formatOffsetTime(seconds: number): string {
  const hrs = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);
  return `${hrs}:${String(mins).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
}

/** Compute timing metadata for events relative to a session start time. */
function computeEventTimings(
  events: readonly SessionEvent[],
  sessionStartIso: string
): ReadonlyArray<{
  readonly eventId: string;
  readonly offsetSeconds: number;
  readonly durationSeconds: number;
}> {
  const sessionStart = new Date(sessionStartIso).getTime();
  return events.map((event, index) => {
    const eventTime = new Date(event.processed_at).getTime();
    const offsetSeconds = Math.max(0, (eventTime - sessionStart) / 1000);
    const prevTime =
      index > 0
        ? new Date(events[index - 1].processed_at).getTime()
        : sessionStart;
    const durationSeconds = Math.max(0, (eventTime - prevTime) / 1000);
    return { eventId: event.id, offsetSeconds, durationSeconds };
  });
}

/** Determine timeline segment color for transcript mode. */
function getTranscriptSegmentColor(type: string): string {
  if (type.startsWith("span.")) return "#5694af";
  if (type === "agent.tool_use" || type === "agent.custom_tool_use" || type === "agent.tool_result")
    return "#a4aad4";
  if (type.startsWith("user.")) return "#3B82F6";
  if (type === "agent.message") return "#10b981";
  return "#dfe3ea";
}

/** Determine timeline marker color for debug mode. */
function getDebugMarkerColor(type: string): string {
  if (type.startsWith("span.")) return "#5694af";
  if (type === "agent.tool_use" || type === "agent.custom_tool_use" || type === "agent.tool_result")
    return "#7675be";
  if (type.startsWith("session.")) return "#94a0b2";
  if (type === "agent.message") return "#10b981";
  if (type.startsWith("user.")) return "#3B82F6";
  return "#94a0b2";
}

/** Available event type filter options for debug mode. */
const DEBUG_FILTER_OPTIONS = [
  { value: "all", label: "All events" },
  { value: "span", label: "Span events" },
  { value: "agent", label: "Agent events" },
  { value: "user", label: "User events" },
  { value: "session", label: "Session events" },
  { value: "tool", label: "Tool events" },
] as const;

type DebugFilterValue = (typeof DEBUG_FILTER_OPTIONS)[number]["value"];

/** Filter events by debug filter type. */
function filterByDebugType(
  events: readonly SessionEvent[],
  filter: DebugFilterValue
): readonly SessionEvent[] {
  if (filter === "all") return events;
  if (filter === "span") return events.filter((e) => e.type.startsWith("span."));
  if (filter === "agent")
    return events.filter(
      (e) =>
        e.type === "agent.message" ||
        e.type === "agent.tool_use" ||
        e.type === "agent.custom_tool_use" ||
        e.type === "agent.tool_result"
    );
  if (filter === "user") return events.filter((e) => e.type.startsWith("user."));
  if (filter === "session") return events.filter((e) => e.type.startsWith("session."));
  if (filter === "tool")
    return events.filter(
      (e) => e.type === "agent.tool_use" || e.type === "agent.custom_tool_use" || e.type === "agent.tool_result"
    );
  return events;
}

/* ─── Timeline Bar ─────────────────────────────────────────────────────── */

function TimelineBar({
  events,
  timings,
  sessionStartIso,
  isTranscript,
  selectedId,
  onSelect,
}: {
  readonly events: readonly SessionEvent[];
  readonly timings: ReadonlyArray<{
    readonly eventId: string;
    readonly offsetSeconds: number;
    readonly durationSeconds: number;
  }>;
  readonly sessionStartIso: string;
  readonly isTranscript: boolean;
  readonly selectedId: string | null;
  readonly onSelect: (id: string) => void;
}) {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const barRef = useRef<HTMLDivElement>(null);

  const displayEvents = isTranscript ? events.filter(isTranscriptEvent) : events;

  // Map timings by event id for quick lookup
  const timingMap = useMemo(() => {
    const m = new Map<string, (typeof timings)[number]>();
    for (const t of timings) {
      m.set(t.eventId, t);
    }
    return m;
  }, [timings]);

  // Compute total span duration for proportional sizing
  const sessionStart = new Date(sessionStartIso).getTime();
  const totalDuration = useMemo(() => {
    if (displayEvents.length === 0) return 1;
    const lastEvent = displayEvents[displayEvents.length - 1];
    const lastTime = new Date(lastEvent.processed_at).getTime();
    return Math.max(1, (lastTime - sessionStart) / 1000);
  }, [displayEvents, sessionStart]);

  if (displayEvents.length === 0) return null;

  if (isTranscript) {
    // Chunky proportional segments
    return (
      <div className="relative px-4 py-2">
        <div
          ref={barRef}
          className="flex w-full h-6 rounded-md overflow-hidden bg-muted gap-px"
        >
          {displayEvents.map((event, idx) => {
            const timing = timingMap.get(event.id);
            const offset = timing?.offsetSeconds ?? 0;
            const duration = timing?.durationSeconds ?? 0;
            // Width proportional to duration, with a minimum for visibility
            const widthPct = Math.max(0.5, (duration / totalDuration) * 100);
            const isSelected = event.id === selectedId;
            const isHovered = hoveredIndex === idx;
            return (
              <div
                key={event.id}
                className="relative h-full cursor-pointer transition-opacity"
                style={{
                  width: `${widthPct}%`,
                  minWidth: "3px",
                  backgroundColor: getTranscriptSegmentColor(event.type),
                  opacity: isSelected ? 1 : isHovered ? 0.85 : 0.7,
                  outline: isSelected ? "2px solid var(--color-primary)" : "none",
                  outlineOffset: "-1px",
                  borderRadius: "2px",
                }}
                onClick={() => onSelect(event.id)}
                onMouseEnter={() => setHoveredIndex(idx)}
                onMouseLeave={() => setHoveredIndex(null)}
              >
                {isHovered && (
                  <div
                    className="absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-2 whitespace-nowrap bg-slate-800 text-white text-[11px] rounded-md px-2.5 py-1.5 shadow-lg pointer-events-none"
                    style={{ minWidth: "140px" }}
                  >
                    <span
                      className="inline-block w-2 h-2 rounded-full mr-1.5"
                      style={{ backgroundColor: getTranscriptSegmentColor(event.type) }}
                    />
                    <span className="font-medium">{getEventCategory(event.type).label}</span>
                    <span className="text-muted-foreground ml-1.5 truncate max-w-[160px] inline-block align-bottom">
                      {(getEventDisplayText(event) || event.type).slice(0, 40)}
                    </span>
                    <div className="text-muted-foreground mt-0.5">
                      {formatShortDuration(duration)} &middot; {formatOffsetTime(offset)}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  // Debug mode: thin vertical markers
  return (
    <div className="relative px-4 py-2">
      <div
        ref={barRef}
        className="relative w-full h-6 rounded-md bg-muted"
      >
        {displayEvents.map((event, idx) => {
          const timing = timingMap.get(event.id);
          const offset = timing?.offsetSeconds ?? 0;
          const duration = timing?.durationSeconds ?? 0;
          const leftPct = (offset / totalDuration) * 100;
          const isSelected = event.id === selectedId;
          const isHovered = hoveredIndex === idx;
          return (
            <div
              key={event.id}
              className="absolute top-0 h-full cursor-pointer"
              style={{
                left: `${Math.min(leftPct, 99.5)}%`,
                width: "3px",
                backgroundColor: getDebugMarkerColor(event.type),
                opacity: isSelected ? 1 : isHovered ? 0.9 : 0.6,
                outline: isSelected ? "1px solid var(--color-primary)" : "none",
                borderRadius: "1px",
              }}
              onClick={() => onSelect(event.id)}
              onMouseEnter={() => setHoveredIndex(idx)}
              onMouseLeave={() => setHoveredIndex(null)}
            >
              {isHovered && (
                <div
                  className="absolute z-50 bottom-full left-1/2 -translate-x-1/2 mb-2 whitespace-nowrap bg-slate-800 text-white text-[11px] rounded-md px-2.5 py-1.5 shadow-lg pointer-events-none"
                  style={{ minWidth: "140px" }}
                >
                  <span
                    className="inline-block w-2 h-2 rounded-full mr-1.5"
                    style={{ backgroundColor: getDebugMarkerColor(event.type) }}
                  />
                  <span className="font-medium">{getEventCategory(event.type).label}</span>
                  <span className="text-muted-foreground ml-1.5">{event.type}</span>
                  <div className="text-muted-foreground mt-0.5">
                    {formatShortDuration(duration)} &middot; {formatOffsetTime(offset)}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

/* ─── Toolbar ──────────────────────────────────────────────────────────── */

function SessionToolbar({
  isDebug,
  debugFilter,
  onDebugFilterChange,
  events,
}: {
  readonly isDebug: boolean;
  readonly debugFilter: DebugFilterValue;
  readonly onDebugFilterChange: (value: DebugFilterValue) => void;
  readonly events: readonly SessionEvent[];
}) {
  const handleCopyAll = useCallback(() => {
    const text = events
      .map(
        (e) =>
          `[${new Date(e.processed_at).toISOString()}] ${e.type}: ${getEventDisplayText(e) || JSON.stringify(e)}`
      )
      .join("\n");
    navigator.clipboard.writeText(text);
  }, [events]);

  const handleDownload = useCallback(() => {
    const json = JSON.stringify(events, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "session-events.json";
    a.click();
    URL.revokeObjectURL(url);
  }, [events]);

  return (
    <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-muted">
      <div>
        {isDebug && (
          <select
            value={debugFilter}
            onChange={(e) => onDebugFilterChange(e.target.value as DebugFilterValue)}
            className="text-xs border border-border rounded-md px-2 py-1 bg-card text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring/40"
          >
            {DEBUG_FILTER_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        )}
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={handleCopyAll}
          className="text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded-md hover:bg-muted transition-colors"
          title="Copy all events to clipboard"
        >
          Copy all
        </button>
        <button
          onClick={handleDownload}
          className="text-xs text-muted-foreground hover:text-foreground px-2 py-1 rounded-md hover:bg-muted transition-colors"
          title="Download events as JSON"
        >
          Download
        </button>
      </div>
    </div>
  );
}

/* ─── Event Timeline (left panel list) ─────────────────────────────────── */

function EventTimeline({
  events,
  selectedId,
  onSelect,
  filterTranscript,
  timings,
}: {
  readonly events: readonly SessionEvent[];
  readonly selectedId: string | null;
  readonly onSelect: (id: string) => void;
  readonly filterTranscript: boolean;
  readonly timings: ReadonlyArray<{
    readonly eventId: string;
    readonly offsetSeconds: number;
    readonly durationSeconds: number;
  }>;
}) {
  const filtered = filterTranscript
    ? events.filter(isTranscriptEvent)
    : events;

  const timingMap = useMemo(() => {
    const m = new Map<string, (typeof timings)[number]>();
    for (const t of timings) {
      m.set(t.eventId, t);
    }
    return m;
  }, [timings]);

  return (
    <div className="space-y-1 p-3 overflow-y-auto max-h-[calc(100vh-420px)]">
      {filtered.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-8">
          No events found
        </p>
      )}
      {filtered.map((event) => {
        const cat = getEventCategory(event.type);
        const isSelected = event.id === selectedId;
        const timing = timingMap.get(event.id);
        return (
          <button
            key={event.id}
            onClick={() => onSelect(event.id)}
            className={`w-full text-left px-3 py-2.5 rounded-lg border transition-colors ${
              isSelected
                ? "bg-primary/10 border-primary/30"
                : `${cat.bgColor} hover:opacity-80`
            }`}
          >
            <div className="flex items-center justify-between mb-1">
              <span
                className={`text-xs font-medium ${isSelected ? "text-primary" : cat.color}`}
              >
                {cat.label}
              </span>
              <div className="flex items-center gap-2">
                {timing && (
                  <>
                    <span className="text-[10px] font-mono text-muted-foreground bg-muted px-1 rounded">
                      {formatShortDuration(timing.durationSeconds)}
                    </span>
                    <span className="text-[10px] font-mono text-muted-foreground">
                      {formatOffsetTime(timing.offsetSeconds)}
                    </span>
                  </>
                )}
                <span className="text-[10px] text-muted-foreground">
                  {new Date(event.processed_at).toLocaleTimeString()}
                </span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground line-clamp-2">
              {filterTranscript
                ? getEventDisplayText(event) || event.type
                : event.type}
            </p>
          </button>
        );
      })}
    </div>
  );
}

function EventDetail({
  event,
  isDebug,
}: {
  readonly event: SessionEvent | null;
  readonly isDebug: boolean;
}) {
  if (!event) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
        Select an event to view details
      </div>
    );
  }

  if (isDebug) {
    return (
      <div className="p-4 overflow-auto max-h-[calc(100vh-320px)]">
        <div className="mb-3">
          <span className="text-xs font-medium text-muted-foreground">
            {event.type}
          </span>
          <span className="text-xs text-muted-foreground ml-2">
            {new Date(event.processed_at).toISOString()}
          </span>
        </div>
        <pre className="text-xs font-mono text-muted-foreground bg-muted rounded-lg p-4 whitespace-pre-wrap overflow-x-auto">
          {JSON.stringify(event, null, 2)}
        </pre>
      </div>
    );
  }

  const displayText = getEventDisplayText(event);
  const cat = getEventCategory(event.type);

  return (
    <div className="p-5 overflow-auto max-h-[calc(100vh-320px)]">
      <div className="flex items-center gap-2 mb-4">
        <span className={`text-xs font-medium ${cat.color}`}>{cat.label}</span>
        <span className="text-xs text-muted-foreground">
          {new Date(event.processed_at).toLocaleTimeString()}
        </span>
      </div>
      {displayText ? (
        <div className="prose prose-sm prose-stone max-w-none">
          <pre className="whitespace-pre-wrap text-sm text-foreground bg-muted rounded-lg p-4 font-sans">
            {displayText}
          </pre>
        </div>
      ) : (
        <pre className="text-xs font-mono text-muted-foreground bg-muted rounded-lg p-4 whitespace-pre-wrap overflow-x-auto">
          {JSON.stringify(event, null, 2)}
        </pre>
      )}
    </div>
  );
}

export default function SessionDetailPage() {
  const params = useParams();
  const sessionId = params.sessionId as string;

  const [session, setSession] = useState<Session | null>(null);
  const [events, setEvents] = useState<readonly SessionEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"transcript" | "debug">(
    "transcript"
  );
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const [debugFilter, setDebugFilter] = useState<DebugFilterValue>("all");

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [sessionData, eventsData] = await Promise.all([
        getSession(sessionId),
        listSessionEvents(sessionId, { limit: 100, order: "asc" }),
      ]);
      setSession(sessionData);
      setEvents(eventsData.data);
      if (eventsData.data.length > 0) {
        setSelectedEventId(eventsData.data[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load session");
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const timings = useMemo(
    () =>
      session ? computeEventTimings(events, session.created_at) : [],
    [events, session]
  );

  const displayEvents = useMemo(
    () =>
      activeTab === "debug" ? filterByDebugType(events, debugFilter) : events,
    [events, activeTab, debugFilter]
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center p-12">
        <div className="inline-block w-6 h-6 border-2 border-slate-300 border-t-slate-600 rounded-full animate-spin" />
      </div>
    );
  }

  if (error || !session) {
    return (
      <ErrorMessage
        message={error ?? "Session not found"}
        onRetry={fetchData}
      />
    );
  }

  const selectedEvent =
    events.find((e) => e.id === selectedEventId) ?? null;

  return (
    <div>
      <PageHeader
        title={session.title || truncateId(session.id, 24)}
        breadcrumbs={[
          { label: "Sessions", href: "/sessions" },
          { label: truncateId(session.id, 20) },
        ]}
      />

      {/* Session metadata */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        <StatusBadge status={session.status} />
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <span>Agent:</span>
          <a
            href={`/agents/${session.agent?.id}`}
            className="text-primary hover:underline font-medium"
          >
            {session.agent?.name ?? truncateId(session.agent?.id ?? "-")}
          </a>
        </div>
        <span className="text-xs text-muted-foreground">
          Env: {truncateId(session.environment_id)}
        </span>
        <span className="text-xs text-muted-foreground">
          Duration: {formatDuration(session.stats.duration_seconds)}
        </span>
      </div>

      {/* Token usage */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-card rounded-xl border border-border p-4">
          <p className="text-xs text-muted-foreground mb-1">Input tokens</p>
          <p className="text-lg font-semibold text-foreground">
            {formatTokens(session.usage.input_tokens)}
          </p>
        </div>
        <div className="bg-card rounded-xl border border-border p-4">
          <p className="text-xs text-muted-foreground mb-1">Output tokens</p>
          <p className="text-lg font-semibold text-foreground">
            {formatTokens(session.usage.output_tokens)}
          </p>
        </div>
        <div className="bg-card rounded-xl border border-border p-4">
          <p className="text-xs text-muted-foreground mb-1">Cache read tokens</p>
          <p className="text-lg font-semibold text-foreground">
            {formatTokens(session.usage.cache_read_input_tokens)}
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-border">
        <div className="flex gap-0">
          <button
            onClick={() => setActiveTab("transcript")}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "transcript"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Transcript
          </button>
          <button
            onClick={() => setActiveTab("debug")}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              activeTab === "debug"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Debug
          </button>
        </div>
      </div>

      {/* Toolbar */}
      <SessionToolbar
        isDebug={activeTab === "debug"}
        debugFilter={debugFilter}
        onDebugFilterChange={setDebugFilter}
        events={displayEvents}
      />

      {/* Timeline Bar */}
      <TimelineBar
        events={displayEvents}
        timings={timings}
        sessionStartIso={session.created_at}
        isTranscript={activeTab === "transcript"}
        selectedId={selectedEventId}
        onSelect={setSelectedEventId}
      />

      {/* Split view */}
      <div className="grid grid-cols-5 gap-4 mt-2">
        {/* Left panel: event timeline */}
        <div className="col-span-2 bg-card rounded-xl border border-border overflow-hidden">
          <div className="px-4 py-3 border-b border-border">
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {activeTab === "transcript" ? "Events" : "Raw Events"}
            </h3>
          </div>
          <EventTimeline
            events={displayEvents}
            selectedId={selectedEventId}
            onSelect={setSelectedEventId}
            filterTranscript={activeTab === "transcript"}
            timings={timings}
          />
        </div>

        {/* Right panel: event detail */}
        <div className="col-span-3 bg-card rounded-xl border border-border overflow-hidden">
          <div className="px-4 py-3 border-b border-border">
            <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {activeTab === "transcript" ? "Content" : "Raw JSON"}
            </h3>
          </div>
          <EventDetail
            event={selectedEvent}
            isDebug={activeTab === "debug"}
          />
        </div>
      </div>
    </div>
  );
}
