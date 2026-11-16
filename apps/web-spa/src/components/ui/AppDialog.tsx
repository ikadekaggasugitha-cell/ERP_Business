import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { motion, useReducedMotion } from 'motion/react';
import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';

export type DialogSize = 'sm' | 'md' | 'lg' | 'xl' | '2xl';

export interface AppDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  size?: DialogSize;
  footer?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

const SIZE_MAP: Record<DialogSize, string> = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-2xl',
  '2xl': 'max-w-4xl',
};

export function AppDialog({
  open,
  onOpenChange,
  title,
  description,
  size = 'lg',
  footer,
  children,
  className,
}: AppDialogProps) {
  const { t } = useTranslation();
  const prefersReduced = useReducedMotion();
  const hasDescription = Boolean(description?.trim());

  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/50 backdrop-blur-xs" />
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 overflow-y-auto">
          <DialogPrimitive.Content
            asChild
            {...(!hasDescription ? { 'aria-describedby': undefined } : {})}
          >
            <motion.div
              initial={prefersReduced ? false : { opacity: 0, scale: 0.97, y: 6 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              transition={
                prefersReduced
                  ? { duration: 0 }
                  : { duration: 0.16, ease: 'easeOut' }
              }
              className={cn(
                'relative w-full rounded-2xl border border-border bg-surface text-ink shadow-[var(--shadow-dialog)] focus:outline-none',
                'flex flex-col max-h-[88vh]',
                SIZE_MAP[size],
                className
              )}
            >
              <div className="flex items-start justify-between gap-4 border-b border-slate-100 dark:border-slate-800 px-6 py-4">
                <div className="min-w-0">
                  <DialogPrimitive.Title className="text-base font-bold text-slate-900 dark:text-slate-100">
                    {title}
                  </DialogPrimitive.Title>
                  {hasDescription ? (
                    <DialogPrimitive.Description className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      {description}
                    </DialogPrimitive.Description>
                  ) : null}
                </div>

                <DialogPrimitive.Close
                  aria-label={t('app.closeDialog')}
                  className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-200 cursor-pointer transition-colors"
                >
                  <X className="h-4 w-4" />
                </DialogPrimitive.Close>
              </div>

              <div className="flex-1 overflow-y-auto px-6 py-5">{children}</div>

              {footer ? (
                <div className="flex flex-wrap items-center justify-end gap-2.5 border-t border-slate-100 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-950/40 px-6 py-3.5 rounded-b-2xl">
                  {footer}
                </div>
              ) : null}
            </motion.div>
          </DialogPrimitive.Content>
        </div>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
