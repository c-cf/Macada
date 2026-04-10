/**
 * Format an ISO date string as a relative time string (e.g., "3 hours ago").
 */
export function timeAgo(dateStr: string): string {
  const now = Date.now();
  const past = new Date(dateStr).getTime();
  const diffMs = now - past;

  if (diffMs < 0) return "just now";

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const weeks = Math.floor(days / 7);
  const months = Math.floor(days / 30);

  if (seconds < 60) return "just now";
  if (minutes === 1) return "1 minute ago";
  if (minutes < 60) return `${minutes} minutes ago`;
  if (hours === 1) return "1 hour ago";
  if (hours < 24) return `${hours} hours ago`;
  if (days === 1) return "1 day ago";
  if (days < 7) return `${days} days ago`;
  if (weeks === 1) return "1 week ago";
  if (weeks < 5) return `${weeks} weeks ago`;
  if (months === 1) return "1 month ago";
  return `${months} months ago`;
}

/**
 * Truncate a string with a prefix and ellipsis.
 * E.g., "agent_01JX..." for long agent IDs.
 */
export function truncateId(id: string, maxLen: number = 16): string {
  if (id.length <= maxLen) return id;
  return `${id.slice(0, maxLen)}...`;
}

/**
 * Format seconds into a human-readable duration string.
 */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds == null || seconds <= 0) return "-";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const mins = Math.floor(seconds / 60);
  const secs = Math.round(seconds % 60);
  if (mins < 60) return `${mins}m ${secs}s`;
  const hrs = Math.floor(mins / 60);
  const remainingMins = mins % 60;
  return `${hrs}h ${remainingMins}m`;
}

/**
 * Format a token count for display (e.g., 1234 -> "1,234").
 */
export function formatTokens(count: number | null | undefined): string {
  if (count == null) return "0";
  return count.toLocaleString();
}

/**
 * Format bytes into a human-readable string (e.g., "1.5 MB").
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, i);
  return `${i === 0 ? value : value.toFixed(1)} ${units[i]}`;
}

/**
 * Determine the status label for an agent.
 */
export function getAgentStatus(agent: {
  archived_at: string | null;
}): "Active" | "Archived" {
  return agent.archived_at ? "Archived" : "Active";
}
