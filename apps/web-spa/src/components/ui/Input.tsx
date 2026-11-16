import * as React from 'react';
import { cn } from '../../lib/utils';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: boolean;
  numeric?: boolean;
  leftIcon?: React.ReactNode;
  rightAdornment?: React.ReactNode;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, error, numeric, leftIcon, rightAdornment, type = 'text', ...props }, ref) => {
    const baseInputClass = cn(
      'w-full h-9 rounded-lg border bg-white px-3 text-sm text-slate-900 placeholder:text-slate-400 shadow-2xs transition-colors',
      'dark:bg-slate-900 dark:text-slate-100 dark:placeholder:text-slate-500',
      error
        ? 'border-rose-500 focus:border-rose-600 focus:outline-none focus:ring-2 focus:ring-rose-500/20'
        : 'border-slate-200 dark:border-slate-700 focus:border-brand-500 focus:outline-none focus:ring-2 focus:ring-brand-500/20',
      'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500 dark:disabled:bg-slate-800/50',
      numeric && 'font-money text-right',
      leftIcon ? 'pl-9' : '',
      rightAdornment ? 'pr-10' : '',
      className
    );

    if (!leftIcon && !rightAdornment) {
      return <input ref={ref} type={type} className={baseInputClass} aria-invalid={error || undefined} {...props} />;
    }

    return (
      <div className="relative flex items-center w-full">
        {leftIcon ? (
          <span className="pointer-events-none absolute left-3 text-slate-400 dark:text-slate-500 flex items-center">
            {leftIcon}
          </span>
        ) : null}
        <input ref={ref} type={type} className={baseInputClass} aria-invalid={error || undefined} {...props} />
        {rightAdornment ? (
          <span className="absolute right-2.5 flex items-center text-slate-500 dark:text-slate-400">
            {rightAdornment}
          </span>
        ) : null}
      </div>
    );
  }
);

Input.displayName = 'Input';
