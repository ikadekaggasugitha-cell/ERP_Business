import { Link } from '@tanstack/react-router';
import { ChevronRight, Home } from 'lucide-react';
import { cn } from '../../lib/utils';

export interface BreadcrumbItem {
  label: string;
  to?: string;
}

export interface BreadcrumbsProps {
  items: BreadcrumbItem[];
  className?: string;
  showHome?: boolean;
}

export function Breadcrumbs({ items, className, showHome = true }: BreadcrumbsProps) {
  return (
    <nav aria-label="Breadcrumb" className={cn('flex items-center text-xs text-slate-500 dark:text-slate-400', className)}>
      <ol className="flex items-center flex-wrap gap-1.5">
        {showHome ? (
          <li className="inline-flex items-center gap-1.5">
            <Link
              to="/"
              className="inline-flex items-center text-slate-400 hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
              aria-label="Home"
            >
              <Home className="h-3.5 w-3.5" />
            </Link>
            {items.length > 0 ? <ChevronRight className="h-3.5 w-3.5 text-slate-300 dark:text-slate-600" /> : null}
          </li>
        ) : null}

        {items.map((item, idx) => {
          const isLast = idx === items.length - 1;
          return (
            <li key={`${item.label}-${idx}`} className="inline-flex items-center gap-1.5">
              {item.to && !isLast ? (
                <Link
                  to={item.to as '/'}
                  className="hover:text-brand-600 dark:hover:text-brand-400 transition-colors font-medium"
                >
                  {item.label}
                </Link>
              ) : (
                <span
                  aria-current={isLast ? 'page' : undefined}
                  className={cn(
                    isLast
                      ? 'font-semibold text-slate-900 dark:text-slate-100'
                      : 'text-slate-500 dark:text-slate-400'
                  )}
                >
                  {item.label}
                </span>
              )}
              {!isLast ? (
                <ChevronRight className="h-3.5 w-3.5 text-slate-300 dark:text-slate-600 shrink-0" />
              ) : null}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
