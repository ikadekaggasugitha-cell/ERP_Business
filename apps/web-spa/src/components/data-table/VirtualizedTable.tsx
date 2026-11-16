import * as React from 'react';
import {
  observeElementRect as defaultObserveElementRect,
  useVirtualizer,
} from '@tanstack/react-virtual';
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  ChevronLeft,
  ChevronRight,
  Download,
  Search,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import { EmptyState } from '../ui/EmptyState';
import { ErrorState } from '../ui/ErrorState';
import { Input } from '../ui/Input';
import { TableSkeleton } from '../ui/Skeleton';

export interface ColumnDef<T> {
  header: string;
  accessorKey?: keyof T;
  cell?: (row: T) => React.ReactNode;
  width?: string;
  className?: string;
  sortKey?: string;
  align?: 'left' | 'right' | 'center';
}

export interface TableSortState {
  key: string;
  direction: 'asc' | 'desc';
}

export interface TableToolbarConfig {
  searchValue?: string;
  onSearchChange?: (value: string) => void;
  searchPlaceholder?: string;
  filters?: React.ReactNode;
  actions?: React.ReactNode;
  onExportCsv?: () => void;
}

export interface TablePaginationConfig {
  page: number;
  size: number;
  totalElements: number;
  totalPages: number;
  pageBase?: 0 | 1;
  onPageChange: (nextPage: number) => void;
}

export interface VirtualizedTableProps<T> {
  data: T[];
  columns: ColumnDef<T>[];
  rowHeight?: number;
  tableHeight?: number | string;
  onRowClick?: (row: T) => void;
  isLoading?: boolean;
  error?: unknown;
  onRetry?: () => void;
  emptyTitle?: string;
  emptyDescription?: string;
  onResetFilters?: () => void;
  toolbar?: TableToolbarConfig;
  sort?: TableSortState | null;
  onSortChange?: (sort: TableSortState) => void;
  pagination?: TablePaginationConfig;
  getRowId?: (row: T, index: number) => string;
  selectedRowId?: string | null;
  rowClassName?: (row: T) => string;
  className?: string;
}

function useIsSmallViewport(): boolean {
  const [isSmall, setIsSmall] = React.useState(false);

  React.useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const mq = window.matchMedia('(max-width: 639px)');
    const sync = () => setIsSmall(mq.matches);
    sync();
    mq.addEventListener('change', sync);
    return () => mq.removeEventListener('change', sync);
  }, []);

  return isSmall;
}

export function VirtualizedTable<T>({
  data,
  columns,
  rowHeight = 46,
  tableHeight = 'calc(100vh - 300px)',
  onRowClick,
  isLoading = false,
  error,
  onRetry,
  emptyTitle,
  emptyDescription,
  onResetFilters,
  toolbar,
  sort,
  onSortChange,
  pagination,
  getRowId,
  selectedRowId,
  rowClassName,
  className,
}: VirtualizedTableProps<T>) {
  const { t } = useTranslation();
  const parentRef = React.useRef<HTMLDivElement>(null);
  const [focusedIndex, setFocusedIndex] = React.useState<number>(-1);
  const isMobile = useIsSmallViewport();

  const rowVirtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: 8,
    initialRect: { width: 960, height: 480 },
    observeElementRect: (instance, cb) =>
      defaultObserveElementRect(instance, (rect) => {
        cb({
          width: rect.width > 0 ? rect.width : 960,
          height: rect.height > 0 ? rect.height : 480,
        });
      }),
  });

  const handleSortClick = (col: ColumnDef<T>) => {
    if (!col.sortKey || !onSortChange) return;
    const isCurrent = sort?.key === col.sortKey;
    const nextDir: 'asc' | 'desc' = isCurrent && sort.direction === 'asc' ? 'desc' : 'asc';
    onSortChange({ key: col.sortKey, direction: nextDir });
  };

  const handleGridKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (data.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      const next = focusedIndex < data.length - 1 ? focusedIndex + 1 : 0;
      setFocusedIndex(next);
      rowVirtualizer.scrollToIndex(next);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const prev = focusedIndex > 0 ? focusedIndex - 1 : data.length - 1;
      setFocusedIndex(prev);
      rowVirtualizer.scrollToIndex(prev);
    } else if (e.key === 'Enter' && focusedIndex >= 0 && focusedIndex < data.length) {
      e.preventDefault();
      onRowClick?.(data[focusedIndex]);
    }
  };

  const resolveRowKey = (row: T, index: number): string | number => {
    if (getRowId) return getRowId(row, index);
    const maybeId = (row as { id?: string | number })?.id;
    return maybeId ?? index;
  };

  const renderCellContent = (row: T, col: ColumnDef<T>): React.ReactNode => {
    if (col.cell) return col.cell(row);
    if (col.accessorKey) {
      const val = row[col.accessorKey];
      return val !== undefined && val !== null ? String(val) : '-';
    }
    return '-';
  };

  return (
    <div
      className={cn(
        'w-full rounded-xl border border-border bg-surface overflow-hidden shadow-[var(--shadow-card)]',
        className
      )}
    >
      {toolbar ? (
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border bg-surface-alt/80 dark:bg-slate-900/60 px-4 py-3">
          <div className="flex flex-wrap items-center gap-2 flex-1 min-w-0">
            {toolbar.onSearchChange ? (
              <div className="w-full sm:w-72">
                <Input
                  value={toolbar.searchValue ?? ''}
                  onChange={(e) => toolbar.onSearchChange?.(e.target.value)}
                  placeholder={toolbar.searchPlaceholder ?? t('table.searchPlaceholder')}
                  leftIcon={<Search className="h-3.5 w-3.5" />}
                />
              </div>
            ) : null}
            {toolbar.filters}
          </div>

          <div className="flex flex-wrap items-center gap-2 shrink-0">
            {toolbar.onExportCsv ? (
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Download className="h-3.5 w-3.5" />}
                onClick={toolbar.onExportCsv}
              >
                {t('actions.exportCsv')}
              </Button>
            ) : null}
            {toolbar.actions}
          </div>
        </div>
      ) : null}

      {isLoading && data.length === 0 ? (
        <TableSkeleton rows={6} columns={Math.min(columns.length, 6)} />
      ) : error && data.length === 0 ? (
        <div className="p-6">
          <ErrorState error={error} onRetry={onRetry} />
        </div>
      ) : data.length === 0 ? (
        <EmptyState
          title={emptyTitle}
          description={emptyDescription}
          onResetFilters={onResetFilters}
        />
      ) : isMobile ? (
        <div className="divide-y divide-slate-100 dark:divide-slate-800 max-h-[70vh] overflow-y-auto">
          {data.map((row, idx) => (
            <div
              key={resolveRowKey(row, idx)}
              onClick={() => onRowClick?.(row)}
              role={onRowClick ? 'button' : undefined}
              tabIndex={onRowClick ? 0 : undefined}
              onKeyDown={(e) => {
                if (onRowClick && (e.key === 'Enter' || e.key === ' ')) {
                  e.preventDefault();
                  onRowClick(row);
                }
              }}
              className={cn(
                'p-4 space-y-2 text-sm text-slate-700 dark:text-slate-200',
                onRowClick && 'cursor-pointer active:bg-brand-50/50 dark:active:bg-slate-800',
                rowClassName?.(row)
              )}
            >
              {columns.map((col, colIdx) => (
                <div key={colIdx} className="flex items-start justify-between gap-3 text-xs">
                  <span className="font-semibold text-slate-500 dark:text-slate-400 shrink-0">
                    {col.header}
                  </span>
                  <span className={cn('text-right truncate', col.className)}>
                    {renderCellContent(row, col)}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      ) : (
        <div
          role="grid"
          aria-rowcount={data.length}
          aria-colcount={columns.length}
          tabIndex={0}
          onKeyDown={handleGridKeyDown}
          className="focus:outline-none"
        >
          {/* Sticky Table Header */}
          <div
            role="row"
            className="flex bg-surface-muted/80 dark:bg-slate-800/75 border-b border-border text-[10.5px] font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider py-2.5 px-4"
          >
            {columns.map((col, index) => {
              const isSorted = Boolean(col.sortKey && sort?.key === col.sortKey);
              const alignClass =
                col.align === 'right'
                  ? 'justify-end text-right'
                  : col.align === 'center'
                    ? 'justify-center text-center'
                    : 'justify-start text-left';

              return (
                <div
                  key={index}
                  role="columnheader"
                  aria-sort={
                    isSorted
                      ? sort?.direction === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                  className={cn('shrink-0 px-2 flex items-center truncate', alignClass, col.className)}
                  style={{ width: col.width || `${100 / columns.length}%` }}
                >
                  {col.sortKey && onSortChange ? (
                    <button
                      type="button"
                      onClick={() => handleSortClick(col)}
                      className="inline-flex items-center gap-1 hover:text-brand-700 dark:hover:text-brand-400 cursor-pointer transition-colors"
                    >
                      <span className="truncate">{col.header}</span>
                      {isSorted ? (
                        sort?.direction === 'asc' ? (
                          <ArrowUp className="h-3.5 w-3.5 text-brand-600 shrink-0" />
                        ) : (
                          <ArrowDown className="h-3.5 w-3.5 text-brand-600 shrink-0" />
                        )
                      ) : (
                        <ArrowUpDown className="h-3 w-3 opacity-45 shrink-0" />
                      )}
                    </button>
                  ) : (
                    <span className="truncate">{col.header}</span>
                  )}
                </div>
              );
            })}
          </div>

          {/* Virtual Scroll Body */}
          <div
            ref={parentRef}
            className="overflow-auto relative"
            style={{ height: tableHeight }}
          >
            <div
              style={{
                height: `${rowVirtualizer.getTotalSize()}px`,
                width: '100%',
                position: 'relative',
              }}
            >
              {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                const row = data[virtualRow.index];
                if (!row) return null;
                const rowKey = resolveRowKey(row, virtualRow.index);
                const isFocused = focusedIndex === virtualRow.index;
                const isSelected =
                  selectedRowId !== null &&
                  selectedRowId !== undefined &&
                  String(rowKey) === selectedRowId;
                return (
                  <div
                    key={rowKey}
                    role="row"
                    aria-rowindex={virtualRow.index + 1}
                    aria-selected={isSelected || undefined}
                    onClick={() => {
                      setFocusedIndex(virtualRow.index);
                      onRowClick?.(row);
                    }}
                    className={cn(
                      'flex items-center px-4 border-b border-border-subtle dark:border-slate-800/80 text-sm text-slate-700 dark:text-slate-200 transition-colors',
                      onRowClick &&
                        'cursor-pointer hover:bg-brand-50/60 dark:hover:bg-slate-800/70',
                      isFocused && 'bg-brand-50/70 dark:bg-slate-800/90',
                      isSelected &&
                        'bg-brand-50/75 border-l-2 border-l-brand-600 dark:bg-brand-950/35 dark:border-l-brand-400',
                      rowClassName?.(row)
                    )}
                    style={{
                      position: 'absolute',
                      top: 0,
                      left: 0,
                      width: '100%',
                      height: `${virtualRow.size}px`,
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                  >
                    {columns.map((col, colIndex) => (
                      <div
                        key={colIndex}
                        role="gridcell"
                        className={cn(
                          'shrink-0 px-2 truncate',
                          col.align === 'right' && 'text-right',
                          col.align === 'center' && 'text-center',
                          col.className
                        )}
                        style={{ width: col.width || `${100 / columns.length}%` }}
                      >
                        {renderCellContent(row, col)}
                      </div>
                    ))}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {pagination
        ? (() => {
            const pageBase = pagination.pageBase ?? 1;
            const displayPage =
              pageBase === 0 ? pagination.page + 1 : Math.max(pagination.page, 1);
            const safeTotalPages = Math.max(pagination.totalPages, 1);
            const isFirstPage = pageBase === 0 ? pagination.page <= 0 : pagination.page <= 1;
            const isLastPage = displayPage >= safeTotalPages;
            return (
              <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-200 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-900/60 px-4 py-2.5 text-xs text-slate-500 dark:text-slate-400">
                <span>
                  {t('table.rowsCount', {
                    shown: data.length,
                    total: pagination.totalElements,
                  })}
                </span>
                <div className="flex items-center gap-2">
                  <span>
                    {t('table.pageInfo', {
                      page: displayPage,
                      totalPages: safeTotalPages,
                    })}
                  </span>
                  <Button
                    variant="outline"
                    size="xs"
                    disabled={isFirstPage}
                    onClick={() => pagination.onPageChange(pagination.page - 1)}
                    aria-label={t('actions.previous')}
                  >
                    <ChevronLeft className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="xs"
                    disabled={isLastPage}
                    onClick={() => pagination.onPageChange(pagination.page + 1)}
                    aria-label={t('actions.next')}
                  >
                    <ChevronRight className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            );
          })()
        : null}
    </div>
  );
}
