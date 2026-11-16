import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';

export type StatusTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand';

export interface StatusBadgeProps {
  status: string;
  label?: string;
  tone?: StatusTone;
  dot?: boolean;
  className?: string;
}

const STATUS_TONE_MAP: Record<string, StatusTone> = {
  DRAFT: 'neutral',
  PENDING: 'warning',
  SOFT_CLOSED: 'warning',
  PARTIAL: 'warning',
  ON_LEAVE: 'warning',
  SUBMITTED: 'info',
  RELEASED: 'info',
  IN_PROGRESS: 'info',
  OPEN: 'info',
  CALCULATED: 'info',
  POSTED: 'success',
  APPROVED: 'success',
  SETTLED: 'success',
  PAID: 'success',
  ACTIVE: 'success',
  CONFIRMED: 'success',
  COMPLETED: 'success',
  MATCHED: 'success',
  ISSUED: 'brand',
  REPORTED: 'brand',
  CONVERTED: 'brand',
  REVERSED: 'danger',
  REJECTED: 'danger',
  VOIDED: 'danger',
  CANCELLED: 'danger',
  FAILED: 'danger',
  HARD_CLOSED: 'neutral',
  INACTIVE: 'neutral',
  LOCKED: 'danger',
  DISABLED: 'neutral',
  TERMINATED: 'danger',
};

const TONE_CLASSES: Record<StatusTone, string> = {
  neutral:
    'bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700',
  success:
    'bg-emerald-50 text-emerald-800 border-emerald-200 dark:bg-emerald-950/60 dark:text-emerald-300 dark:border-emerald-800/70',
  warning:
    'bg-amber-50 text-amber-800 border-amber-200 dark:bg-amber-950/60 dark:text-amber-300 dark:border-amber-800/70',
  danger:
    'bg-rose-50 text-rose-800 border-rose-200 dark:bg-rose-950/60 dark:text-rose-300 dark:border-rose-800/70',
  info:
    'bg-brand-50 text-brand-800 border-brand-200 dark:bg-brand-950/60 dark:text-brand-300 dark:border-brand-800/70',
  brand:
    'bg-indigo-50 text-indigo-800 border-indigo-200 dark:bg-indigo-950/60 dark:text-indigo-300 dark:border-indigo-800/70',
};

const DOT_CLASSES: Record<StatusTone, string> = {
  neutral: 'bg-slate-500 dark:bg-slate-400',
  success: 'bg-emerald-500 dark:bg-emerald-400',
  warning: 'bg-amber-500 dark:bg-amber-400',
  danger: 'bg-rose-500 dark:bg-rose-400',
  info: 'bg-brand-500 dark:bg-brand-400',
  brand: 'bg-indigo-500 dark:bg-indigo-400',
};

export function StatusBadge({ status, label, tone, dot = true, className }: StatusBadgeProps) {
  const { t } = useTranslation();
  const normalized = (status || 'UNKNOWN').trim().toUpperCase();
  const resolvedTone = tone ?? STATUS_TONE_MAP[normalized] ?? 'neutral';
  const translated = label ?? t(`statuses.${normalized}`, { defaultValue: status });

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold tracking-tight whitespace-nowrap',
        TONE_CLASSES[resolvedTone],
        className
      )}
    >
      {dot ? (
        <span className={cn('h-1.5 w-1.5 rounded-full shrink-0', DOT_CLASSES[resolvedTone])} aria-hidden="true" />
      ) : null}
      <span>{translated}</span>
    </span>
  );
}
