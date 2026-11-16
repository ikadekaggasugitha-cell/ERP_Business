import * as React from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';
import { Check, ChevronsUpDown, Search, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';

export interface ComboboxOption {
  value: string;
  label: string;
  subtitle?: string;
  keywords?: string[];
  disabled?: boolean;
}

export interface ComboboxProps {
  id?: string;
  value?: string;
  onValueChange: (value: string) => void;
  options: ComboboxOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  allowClear?: boolean;
  disabled?: boolean;
  error?: boolean;
  className?: string;
}

export function Combobox({
  id,
  value,
  onValueChange,
  options,
  placeholder,
  searchPlaceholder,
  emptyMessage,
  allowClear = false,
  disabled = false,
  error = false,
  className,
}: ComboboxProps) {
  const { t } = useTranslation();
  const resolvedPlaceholder = placeholder ?? t('combobox.placeholder');
  const resolvedSearchPlaceholder = searchPlaceholder ?? t('combobox.searchPlaceholder');
  const resolvedEmptyMessage = emptyMessage ?? t('combobox.empty');
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState('');
  const [activeIndex, setActiveIndex] = React.useState(0);

  const selectedOption = React.useMemo(
    () => options.find((o) => o.value === value),
    [options, value]
  );

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return options;
    return options.filter((o) => {
      if (o.label.toLowerCase().includes(q)) return true;
      if (o.subtitle && o.subtitle.toLowerCase().includes(q)) return true;
      if (o.value.toLowerCase().includes(q)) return true;
      return Boolean(o.keywords?.some((kw) => kw.toLowerCase().includes(q)));
    });
  }, [options, query]);

  React.useEffect(() => {
    setActiveIndex(0);
  }, [query, open]);

  const handleSelect = (opt: ComboboxOption) => {
    if (opt.disabled) return;
    onValueChange(opt.value);
    setOpen(false);
    setQuery('');
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (filtered.length === 0) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIndex((prev) => (prev + 1) % filtered.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((prev) => (prev - 1 + filtered.length) % filtered.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const target = filtered[activeIndex];
      if (target) handleSelect(target);
    }
  };

  const showClear = Boolean(allowClear && value && !disabled);

  return (
    <PopoverPrimitive.Root open={open} onOpenChange={setOpen}>
      <div className="relative flex items-center w-full">
        <PopoverPrimitive.Trigger asChild>
          <button
            id={id}
            type="button"
            role="combobox"
            aria-expanded={open}
            aria-invalid={error || undefined}
            disabled={disabled}
            className={cn(
              'flex h-9 w-full items-center justify-between gap-2 rounded-lg border bg-white pl-3 text-sm text-left shadow-2xs transition-colors cursor-pointer',
              showClear ? 'pr-8' : 'pr-3',
              'dark:bg-slate-900 dark:text-slate-100',
              error
                ? 'border-rose-500 focus:border-rose-600 focus:outline-none focus:ring-2 focus:ring-rose-500/20'
                : 'border-slate-200 dark:border-slate-700 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20',
              'disabled:cursor-not-allowed disabled:opacity-50',
              className
            )}
          >
            <span
              className={cn(
                'truncate',
                !selectedOption && 'text-slate-400 dark:text-slate-500',
                showClear && 'pr-6'
              )}
            >
              {selectedOption ? selectedOption.label : resolvedPlaceholder}
            </span>
            <ChevronsUpDown className="h-4 w-4 text-slate-400 shrink-0" />
          </button>
        </PopoverPrimitive.Trigger>
        {showClear ? (
          <button
            type="button"
            aria-label={t('actions.clearSelection')}
            onClick={() => onValueChange('')}
            className="absolute right-8 inline-flex h-6 w-6 items-center justify-center rounded hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 hover:text-slate-600 cursor-pointer"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        ) : null}
      </div>

      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="start"
          sideOffset={4}
          className={cn(
            'z-50 w-(--radix-popover-trigger-width) min-w-60 overflow-hidden rounded-xl border border-slate-200 bg-white text-slate-900 shadow-lg',
            'dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100'
          )}
        >
          <div className="flex items-center gap-2 border-b border-slate-100 dark:border-slate-800 px-3 py-2">
            <Search className="h-4 w-4 text-slate-400 shrink-0" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={resolvedSearchPlaceholder}
              autoFocus
              className="w-full bg-transparent text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none"
            />
          </div>

          <div role="listbox" className="max-h-60 overflow-y-auto p-1">
            {filtered.length === 0 ? (
              <div className="py-6 text-center text-xs text-slate-500 dark:text-slate-400">
                {resolvedEmptyMessage}
              </div>
            ) : (
              filtered.map((opt, idx) => {
                const isSelected = opt.value === value;
                const isHighlighted = idx === activeIndex;
                return (
                  <button
                    key={opt.value}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    disabled={opt.disabled}
                    onMouseEnter={() => setActiveIndex(idx)}
                    onClick={() => handleSelect(opt)}
                    className={cn(
                      'flex w-full items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition-colors cursor-pointer',
                      isHighlighted && 'bg-brand-50 dark:bg-slate-800',
                      isSelected && 'font-medium text-brand-700 dark:text-brand-400',
                      opt.disabled && 'opacity-40 cursor-not-allowed'
                    )}
                  >
                    <div className="min-w-0">
                      <div className="truncate">{opt.label}</div>
                      {opt.subtitle ? (
                        <div className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                          {opt.subtitle}
                        </div>
                      ) : null}
                    </div>
                    {isSelected ? (
                      <Check className="h-4 w-4 text-brand-600 dark:text-brand-400 shrink-0" />
                    ) : null}
                  </button>
                );
              })
            )}
          </div>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
