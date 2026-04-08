interface StatusBadgeProps {
  readonly status: string;
}

const STATUS_STYLES: Record<string, string> = {
  active:
    "bg-success/10 text-success border border-success/20",
  running:
    "bg-info/10 text-info border border-info/20",
  idle:
    "bg-muted text-muted-foreground border border-border",
  terminated:
    "bg-destructive/10 text-destructive border border-destructive/20",
  rescheduling:
    "bg-warning/10 text-warning border border-warning/20",
  archived:
    "bg-muted text-muted-foreground border border-border",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  const key = status.toLowerCase();
  const style =
    STATUS_STYLES[key] ?? "bg-muted text-muted-foreground border border-border";

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${style}`}
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}
