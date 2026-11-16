import * as React from 'react';
import { cn } from '../../lib/utils';
import { Label } from './Label';

export interface FormFieldProps {
  label?: string;
  htmlFor?: string;
  required?: boolean;
  hint?: string;
  description?: string;
  error?: string;
  className?: string;
  children: React.ReactNode;
}

export function FormField({
  label,
  htmlFor,
  required,
  hint,
  description,
  error,
  className,
  children,
}: FormFieldProps) {
  return (
    <div className={cn('space-y-1.5', className)}>
      {label ? (
        <Label htmlFor={htmlFor} required={required} hint={hint}>
          {label}
        </Label>
      ) : null}
      {children}
      {description && !error ? (
        <p className="text-[11px] text-slate-500 dark:text-slate-400 leading-snug">{description}</p>
      ) : null}
      {error ? (
        <p role="alert" className="text-[11px] font-medium text-rose-600 dark:text-rose-400 leading-snug">
          {error}
        </p>
      ) : null}
    </div>
  );
}
