import * as SelectPrimitive from '@radix-ui/react-select';
import { Check, ChevronDown, ChevronUp } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';

export interface SelectOption {
  value: string;
  label: string;
  description?: string;
  disabled?: boolean;
}

export interface SelectProps {
  id?: string;
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
  error?: boolean;
  className?: string;
  'aria-label'?: string;
}

export function Select({
  id,
  value,
  defaultValue,
  onValueChange,
  options,
  placeholder,
  disabled,
  error,
  className,
  'aria-label': ariaLabel,
}: SelectProps) {
  const { t } = useTranslation();
  const resolvedPlaceholder = placeholder ?? t('select.placeholder');

  return (
    <SelectPrimitive.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
      disabled={disabled}
    >
      <SelectPrimitive.Trigger
        id={id}
        aria-label={ariaLabel}
        aria-invalid={error || undefined}
        className={cn(
          'flex h-9 w-full items-center justify-between gap-2 rounded-lg border bg-white px-3 text-sm text-slate-900 shadow-2xs transition-colors cursor-pointer',
          'dark:bg-slate-900 dark:text-slate-100',
          'data-placeholder:text-slate-400 dark:data-placeholder:text-slate-500',
          error
            ? 'border-rose-500 focus:border-rose-600 focus:outline-none focus:ring-2 focus:ring-rose-500/20'
            : 'border-slate-200 dark:border-slate-700 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20',
          'disabled:cursor-not-allowed disabled:opacity-50',
          className
        )}
      >
        <SelectPrimitive.Value placeholder={resolvedPlaceholder} />
        <SelectPrimitive.Icon asChild>
          <ChevronDown className="h-4 w-4 text-slate-400 shrink-0" />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>

      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          position="popper"
          sideOffset={4}
          className={cn(
            'z-50 max-h-72 min-w-(--radix-select-trigger-width) overflow-hidden rounded-xl border border-slate-200 bg-white text-slate-900 shadow-lg',
            'dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100'
          )}
        >
          <SelectPrimitive.ScrollUpButton className="flex items-center justify-center py-1 text-slate-400">
            <ChevronUp className="h-4 w-4" />
          </SelectPrimitive.ScrollUpButton>

          <SelectPrimitive.Viewport className="p-1">
            {options.map((opt) => (
              <SelectPrimitive.Item
                key={opt.value}
                value={opt.value}
                disabled={opt.disabled}
                className={cn(
                  'relative flex w-full cursor-pointer select-none items-center justify-between rounded-lg py-2 pl-3 pr-8 text-sm outline-none transition-colors',
                  'data-highlighted:bg-brand-50 data-highlighted:text-brand-900 dark:data-highlighted:bg-slate-800 dark:data-highlighted:text-slate-100',
                  'data-disabled:pointer-events-none data-disabled:opacity-40'
                )}
              >
                <div className="min-w-0">
                  <SelectPrimitive.ItemText>{opt.label}</SelectPrimitive.ItemText>
                  {opt.description ? (
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {opt.description}
                    </p>
                  ) : null}
                </div>
                <SelectPrimitive.ItemIndicator className="absolute right-2.5 inline-flex items-center text-brand-600 dark:text-brand-400">
                  <Check className="h-4 w-4" />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>

          <SelectPrimitive.ScrollDownButton className="flex items-center justify-center py-1 text-slate-400">
            <ChevronDown className="h-4 w-4" />
          </SelectPrimitive.ScrollDownButton>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
