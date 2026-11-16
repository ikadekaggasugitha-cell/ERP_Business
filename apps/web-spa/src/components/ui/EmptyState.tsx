import * as React from 'react';
import { FolderOpen, RotateCcw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';
import { Button } from './Button';

export interface EmptyStateProps {
  title?: string;
  description?: string;
  action?: React.ReactNode;
  onResetFilters?: () => void;
  imageSrc?: string;
  className?: string;
}

export function EmptyState({
  title,
  description,
  action,
  onResetFilters,
  imageSrc = '/empty-state.png',
  className,
}: EmptyStateProps) {
  const { t } = useTranslation();
  const [imgFailed, setImgFailed] = React.useState(false);

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center py-12 px-6',
        className
      )}
    >
      {!imgFailed && imageSrc ? (
        <img
          src={imageSrc}
          alt=""
          onError={() => setImgFailed(true)}
          className="h-24 w-24 object-contain mb-3 opacity-90"
        />
      ) : (
        <div className="mb-3 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-100 text-slate-400 dark:bg-slate-800 dark:text-slate-500">
          <FolderOpen className="h-7 w-7" />
        </div>
      )}

      <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
        {title ?? t('table.emptyTitle')}
      </h3>
      <p className="mt-1 max-w-md text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
        {description ?? t('table.emptyDescription')}
      </p>

      {action || onResetFilters ? (
        <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
          {onResetFilters ? (
            <Button
              variant="outline"
              size="sm"
              leftIcon={<RotateCcw className="h-3.5 w-3.5" />}
              onClick={onResetFilters}
            >
              {t('actions.resetFilters')}
            </Button>
          ) : null}
          {action}
        </div>
      ) : null}
    </div>
  );
}
