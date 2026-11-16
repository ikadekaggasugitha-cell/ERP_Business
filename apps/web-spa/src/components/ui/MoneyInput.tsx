import * as React from 'react';
import { cn } from '../../lib/utils';

export interface MoneyInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  value: string;
  onValueChange: (nextValue: string) => void;
  currency?: string;
  precision?: number;
  allowNegative?: boolean;
  error?: boolean;
}

function sanitizeDecimalString(
  raw: string,
  precision: number,
  allowNegative: boolean
): string {
  // Accept dot or comma as decimal separator while stripping whitespace and group chars
  let cleaned = raw.replace(/\s+/g, '').replace(/,/g, '.');

  let sign = '';
  if (allowNegative && cleaned.startsWith('-')) {
    sign = '-';
  }
  cleaned = cleaned.replace(/[^0-9.]/g, '');

  const parts = cleaned.split('.');
  const intPart = parts[0] ?? '';
  if (parts.length <= 1 || precision === 0) {
    return `${sign}${intPart}`;
  }

  const fracPart = parts.slice(1).join('').slice(0, precision);
  return `${sign}${intPart}.${fracPart}`;
}

export const MoneyInput = React.forwardRef<HTMLInputElement, MoneyInputProps>(
  (
    {
      value,
      onValueChange,
      currency = 'IDR',
      precision = 2,
      allowNegative = false,
      error = false,
      className,
      onBlur,
      placeholder,
      disabled,
      ...props
    },
    ref
  ) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const sanitized = sanitizeDecimalString(e.target.value, precision, allowNegative);
      onValueChange(sanitized);
    };

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
      if (value && value !== '-') {
        const [intRaw = '0', fracRaw = ''] = value.replace(/^-/, '').split('.');
        const sign = allowNegative && value.startsWith('-') ? '-' : '';
        const cleanInt = intRaw.replace(/^0+(?=\d)/, '') || '0';
        const formatted =
          precision > 0
            ? `${sign}${cleanInt}.${fracRaw.padEnd(precision, '0').slice(0, precision)}`
            : `${sign}${cleanInt}`;
        if (formatted !== value) {
          onValueChange(formatted);
        }
      }
      onBlur?.(e);
    };

    return (
      <div className="relative flex items-center w-full">
        {currency ? (
          <span className="pointer-events-none absolute left-2 inline-flex items-center px-1.5 py-0.5 rounded border border-slate-200 bg-slate-100 text-[10.5px] font-semibold font-mono text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200">
            {currency}
          </span>
        ) : null}
        <input
          ref={ref}
          type="text"
          inputMode="decimal"
          value={value}
          disabled={disabled}
          placeholder={placeholder ?? (precision > 0 ? `0.${'0'.repeat(precision)}` : '0')}
          onChange={handleChange}
          onBlur={handleBlur}
          aria-invalid={error || undefined}
          className={cn(
            'w-full h-9 rounded-lg border bg-white pr-3 text-sm font-money text-right text-slate-900 placeholder:text-slate-400 shadow-2xs transition-colors',
            'dark:bg-slate-900 dark:text-slate-100 dark:placeholder:text-slate-500',
            currency ? 'pl-14' : 'pl-3',
            error
              ? 'border-rose-500 focus:border-rose-600 focus:outline-none focus:ring-2 focus:ring-rose-500/20'
              : 'border-slate-200 dark:border-slate-700 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20',
            'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500 dark:disabled:bg-slate-800/50',
            className
          )}
          {...props}
        />
      </div>
    );
  }
);

MoneyInput.displayName = 'MoneyInput';
