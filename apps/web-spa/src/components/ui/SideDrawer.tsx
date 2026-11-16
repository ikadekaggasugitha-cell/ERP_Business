import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { motion, useReducedMotion } from 'motion/react';
import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';

export type DrawerSize = 'md' | 'lg' | 'xl';

export interface SideDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  titleBadge?: React.ReactNode;
  description?: string;
  size?: DrawerSize;
  footer?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

const DRAWER_WIDTH: Record<DrawerSize, string> = {
  md: 'sm:max-w-md',
  lg: 'sm:max-w-lg',
  xl: 'sm:max-w-2xl',
};

export function SideDrawer({
  open,
  onOpenChange,
  title,
  titleBadge,
  description,
  size = 'lg',
  footer,
  children,
  className,
}: SideDrawerProps) {
  const { t } = useTranslation();
  const prefersReduced = useReducedMotion();
  const hasDescription = Boolean(description?.trim());

  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/50 backdrop-blur-xs" />
        <DialogPrimitive.Content
          asChild
          {...(!hasDescription ? { 'aria-describedby': undefined } : {})}
        >
          <motion.div
            initial={prefersReduced ? false : { opacity: 0, x: 24 }}
            animate={{ opacity: 1, x: 0 }}
            transition={
              prefersReduced ? { duration: 0 } : { duration: 0.2, ease: 'easeOut' }
            }
            className={cn(
              'fixed z-50 flex flex-col border border-border bg-surface text-ink shadow-[var(--shadow-drawer)] focus:outline-none',
              // Mobile bottom sheet
              'inset-x-0 bottom-0 max-h-[90vh] rounded-t-2xl',
              // Desktop right slide-over drawer
              'sm:inset-y-0 sm:right-0 sm:left-auto sm:max-h-none sm:h-full sm:w-full sm:rounded-none sm:rounded-l-2xl',
              DRAWER_WIDTH[size],
              className
            )}
          >
            <div className="mx-auto mt-2 h-1 w-10 rounded-full bg-slate-300 dark:bg-slate-700 sm:hidden" />

            <div className="flex items-start justify-between gap-4 border-b border-border-subtle dark:border-slate-800 px-6 py-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2.5">
                  <DialogPrimitive.Title className="text-base font-bold text-slate-900 dark:text-slate-100">
                    {title}
                  </DialogPrimitive.Title>
                  {titleBadge}
                </div>
                {hasDescription ? (
                  <DialogPrimitive.Description className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {description}
                  </DialogPrimitive.Description>
                ) : null}
              </div>

              <DialogPrimitive.Close
                aria-label={t('app.closePanel')}
                className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800 dark:hover:text-slate-200 cursor-pointer transition-colors"
              >
                <X className="h-4 w-4" />
              </DialogPrimitive.Close>
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-5">{children}</div>

            {footer ? (
              <div className="flex flex-wrap items-center justify-end gap-2.5 border-t border-slate-100 dark:border-slate-800 bg-slate-50/70 dark:bg-slate-950/40 px-6 py-4">
                {footer}
              </div>
            ) : null}
          </motion.div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
