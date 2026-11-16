import * as React from 'react';
import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-react';
import { cn } from '../../lib/utils';
import { KpiSkeleton } from './Skeleton';
import type { StatusTone } from './StatusBadge';

export interface KpiCardProps {
  label: string;
  value: string | React.ReactNode;
  subtitle?: string;
  delta?: string;
  deltaDirection?: 'up' | 'down' | 'flat';
  badge?: string;
  badgeTone?: StatusTone;
  freshness?: string;
  icon?: React.ReactNode;
  accent?: 'brand' | 'emerald' | 'amber' | 'rose' | 'sky';
  isLoading?: boolean;
  onClick?: () => void;
  className?: string;
}

const ACCENT_ICON_BG = {
  brand:
    'border-brand-200 bg-brand-50 text-brand-700 dark:border-brand-900 dark:bg-brand-950/70 dark:text-brand-400',
  emerald:
    'border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-900 dark:bg-emerald-950/70 dark:text-emerald-400',
  amber:
    'border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-900 dark:bg-amber-950/70 dark:text-amber-400',
  rose:
    'border-rose-200 bg-rose-50 text-rose-600 dark:border-rose-900 dark:bg-rose-950/70 dark:text-rose-400',
  sky:
    'border-sky-200 bg-sky-50 text-sky-600 dark:border-sky-900 dark:bg-sky-950/70 dark:text-sky-400',
};

const BADGE_TONE_CLASSES: Record<StatusTone, string> = {
  neutral:
    'border-slate-200 bg-slate-100 text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300',
  success:
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/70 dark:bg-emerald-950/60 dark:text-emerald-300',
  warning:
    'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800/70 dark:bg-amber-950/60 dark:text-amber-300',
  danger:
    'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/60 dark:text-rose-300',
  info:
    'border-brand-200 bg-brand-50 text-brand-700 dark:border-brand-800/70 dark:bg-brand-950/60 dark:text-brand-300',
  brand:
    'border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-indigo-800/70 dark:bg-indigo-950/60 dark:text-indigo-300',
};

export function KpiCard({
  label,
  value,
  subtitle,
  delta,
  deltaDirection = 'flat',
  badge,
  badgeTone = 'neutral',
  freshness,
  icon,
  accent = 'brand',
  isLoading = false,
  onClick,
  className,
}: KpiCardProps) {
  if (isLoading) {
    return <KpiSkeleton className={className} />;
  }

  const Wrapper = onClick ? 'button' : 'div';

  return (
    <Wrapper
      type={onClick ? 'button' : undefined}
      onClick={onClick}
      className={cn(
        'w-full text-left rounded-xl border border-border bg-surface p-4.5 sm:p-5 shadow-[var(--shadow-card)] transition-all',
        onClick &&
          'cursor-pointer hover:border-brand-300 hover:shadow-sm dark:hover:border-brand-700',
        className
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <span className="text-[11px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400 leading-snug">
          {label}
        </span>
        {icon ? (
          <div className={cn('p-1.5 rounded-lg border shrink-0', ACCENT_ICON_BG[accent])}>
            {icon}
          </div>
        ) : null}
      </div>

      <div className="mt-2 text-xl sm:text-[22px] font-bold tracking-tight text-slate-900 dark:text-slate-100 font-money truncate">
        {value}
      </div>

      {(delta || badge || subtitle || freshness) ? (
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-[11px]">
          <div className="flex items-center gap-1.5 min-w-0">
            {badge ? (
              <span
                className={cn(
                  'inline-flex items-center rounded-full border px-2 py-0.5 text-[10.5px] font-semibold tracking-tight whitespace-nowrap shrink-0',
                  BADGE_TONE_CLASSES[badgeTone]
                )}
              >
                {badge}
              </span>
            ) : null}
            {delta ? (
              <span
                className={cn(
                  'inline-flex items-center gap-0.5 font-semibold font-money',
                  deltaDirection === 'up' && 'text-emerald-600 dark:text-emerald-400',
                  deltaDirection === 'down' && 'text-rose-600 dark:text-rose-400',
                  deltaDirection === 'flat' && 'text-slate-500 dark:text-slate-400'
                )}
              >
                {deltaDirection === 'up' ? (
                  <ArrowUpRight className="h-3.5 w-3.5" />
                ) : deltaDirection === 'down' ? (
                  <ArrowDownRight className="h-3.5 w-3.5" />
                ) : (
                  <Minus className="h-3 w-3" />
                )}
                {delta}
              </span>
            ) : null}
            {subtitle ? (
              <span className="text-slate-500 dark:text-slate-400 truncate">{subtitle}</span>
            ) : null}
          </div>

          {freshness ? (
            <span className="text-[10.5px] text-slate-500 dark:text-slate-400 shrink-0 font-mono">
              {freshness}
            </span>
          ) : null}
        </div>
      ) : null}
    </Wrapper>
  );
}
