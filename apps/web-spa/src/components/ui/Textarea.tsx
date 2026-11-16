import * as React from 'react';
import { cn } from '../../lib/utils';

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  error?: boolean;
}

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, error, ...props }, ref) => {
    return (
      <textarea
        ref={ref}
        aria-invalid={error || undefined}
        className={cn(
          'w-full min-h-20 rounded-lg border bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 shadow-2xs transition-colors',
          'dark:bg-slate-900 dark:text-slate-100 dark:placeholder:text-slate-500',
          error
            ? 'border-rose-500 focus:border-rose-600 focus:ring-2 focus:ring-rose-500/20'
            : 'border-slate-300 dark:border-slate-700 focus:border-brand-600 focus:ring-2 focus:ring-brand-500/20',
          'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500 dark:disabled:bg-slate-800/50',
          className
        )}
        {...props}
      />
    );
  }
);

Textarea.displayName = 'Textarea';
