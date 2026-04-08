import { ReactNode } from "react";

interface PageHeaderProps {
  readonly title: string;
  readonly subtitle?: string;
  readonly actions?: ReactNode;
  readonly breadcrumbs?: readonly { label: string; href?: string }[];
}

export function PageHeader({
  title,
  subtitle,
  actions,
  breadcrumbs,
}: PageHeaderProps) {
  return (
    <div className="mb-6">
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="flex items-center gap-1.5 text-sm text-muted-foreground mb-3">
          {breadcrumbs.map((crumb, i) => (
            <span key={crumb.label} className="flex items-center gap-1.5">
              {i > 0 && <span>/</span>}
              {crumb.href ? (
                <a
                  href={crumb.href}
                  className="hover:text-foreground transition-colors"
                >
                  {crumb.label}
                </a>
              ) : (
                <span className="text-foreground">{crumb.label}</span>
              )}
            </span>
          ))}
        </nav>
      )}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
          {subtitle && (
            <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
          )}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}
