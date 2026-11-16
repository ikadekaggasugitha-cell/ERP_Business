import * as React from 'react';
import { cn } from '../../lib/utils';

export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  required?: boolean;
  hint?: string;
}

export const Label = React.forwardRef<HTMLLabelElement, LabelProps>(
  ({ className, children, required, hint, ...props }, ref) => {
    return (
      <label
        ref={ref}
        className={cn(
          'inline-flex items-center gap-1.5 text-xs font-semibold text-slate-700 dark:text-slate-200',
          className
        )}
        {...props}
      >
        <span>{children}</span>
        {required ? (
          <span className="text-rose-500 font-bold" aria-hidden="true">
            *
          </span>
        ) : null}
        {hint ? (
          <span className="font-normal text-slate-400 dark:text-slate-500">({hint})</span>
        ) : null}
      </label>
    );
  }
);

Label.displayName = 'Label';
