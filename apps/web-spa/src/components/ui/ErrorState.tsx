import * as React from 'react';
import { AlertTriangle, Check, Copy, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ApiClientError } from '../../lib/api';
import { cn } from '../../lib/utils';
import { Button } from './Button';

export interface ErrorStateProps {
  error?: unknown;
  title?: string;
  message?: string;
  code?: string;
  correlationId?: string;
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({
  error,
  title,
  message,
  code,
  correlationId,
  onRetry,
  className,
}: ErrorStateProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = React.useState(false);

  const apiErr = error instanceof ApiClientError ? error : null;
  const resolvedCode = code ?? apiErr?.code ?? 'REQUEST_FAILED';
  const resolvedMessage =
    message ??
    (error instanceof Error ? error.message : t('errors.boundaryDesc'));
  const resolvedCorrelation = correlationId ?? apiErr?.correlationId;

  const handleCopy = async () => {
    if (!resolvedCorrelation) return;
    try {
      await navigator.clipboard.writeText(resolvedCorrelation);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Ignore clipboard API failure in restricted environments
    }
  };

  return (
    <div
      role="alert"
      className={cn(
        'rounded-xl border border-rose-200 bg-rose-50/70 p-6 text-center dark:border-rose-900/60 dark:bg-rose-950/25',
        className
      )}
    >
      <div className="mx-auto mb-3 inline-flex h-11 w-11 items-center justify-center rounded-xl bg-rose-100 text-rose-600 dark:bg-rose-900/50 dark:text-rose-400">
        <AlertTriangle className="h-5 w-5" />
      </div>

      <h3 className="text-sm font-semibold text-rose-900 dark:text-rose-200">
        {title ?? t('errors.boundaryTitle')}
      </h3>
      <p className="mx-auto mt-1 max-w-lg text-xs text-rose-700 dark:text-rose-300/90 leading-relaxed">
        {resolvedMessage}
      </p>

      <div className="mt-3 flex flex-wrap items-center justify-center gap-2 text-[11px] font-mono">
        <span className="rounded bg-white/80 dark:bg-slate-900/70 border border-rose-200 dark:border-rose-800 px-2 py-0.5 text-rose-700 dark:text-rose-300">
          {t('errors.errorCode')}: {resolvedCode}
        </span>
        {resolvedCorrelation ? (
          <button
            type="button"
            onClick={handleCopy}
            className="inline-flex items-center gap-1 rounded bg-white/80 dark:bg-slate-900/70 border border-rose-200 dark:border-rose-800 px-2 py-0.5 text-slate-600 dark:text-slate-300 hover:bg-white cursor-pointer"
          >
            <span>CID: {resolvedCorrelation.slice(0, 12)}...</span>
            {copied ? (
              <Check className="h-3 w-3 text-emerald-600" />
            ) : (
              <Copy className="h-3 w-3" />
            )}
          </button>
        ) : null}
      </div>

      {onRetry ? (
        <div className="mt-4">
          <Button
            variant="outline"
            size="sm"
            leftIcon={<RefreshCw className="h-3.5 w-3.5" />}
            onClick={onRetry}
          >
            {t('actions.retry')}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
