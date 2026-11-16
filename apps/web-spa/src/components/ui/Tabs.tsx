import * as React from 'react';
import * as TabsPrimitive from '@radix-ui/react-tabs';
import { cn } from '../../lib/utils';

export const Tabs = TabsPrimitive.Root;

export const TabsList = React.forwardRef<
  React.ComponentRef<typeof TabsPrimitive.List>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.List> & { variant?: 'pills' | 'underline' }
>(({ className, variant = 'pills', ...props }, ref) => (
  <TabsPrimitive.List
    ref={ref}
    className={cn(
      'inline-flex items-center gap-1',
      variant === 'pills'
        ? 'rounded-xl bg-slate-100 p-1 text-slate-600 border border-slate-200/70 dark:border-slate-700/80 dark:bg-slate-800/80 dark:text-slate-400'
        : 'border-b border-border w-full gap-4',
      className
    )}
    {...props}
  />
));
TabsList.displayName = TabsPrimitive.List.displayName;

export const TabsTrigger = React.forwardRef<
  React.ComponentRef<typeof TabsPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger> & { variant?: 'pills' | 'underline' }
>(({ className, variant = 'pills', ...props }, ref) => (
  <TabsPrimitive.Trigger
    ref={ref}
    className={cn(
      'inline-flex items-center justify-center whitespace-nowrap text-xs font-semibold transition-all cursor-pointer disabled:pointer-events-none disabled:opacity-50',
      variant === 'pills'
        ? 'rounded-lg px-3 py-1.5 data-[state=active]:bg-white data-[state=active]:text-brand-700 data-[state=active]:shadow-xs dark:data-[state=active]:bg-slate-900 dark:data-[state=active]:text-brand-400'
        : 'border-b-2 border-transparent px-3 py-2.5 text-slate-500 hover:text-slate-900 data-[state=active]:border-brand-600 data-[state=active]:text-brand-700 dark:text-slate-400 dark:hover:text-slate-200 dark:data-[state=active]:border-brand-400 dark:data-[state=active]:text-brand-400',
      className
    )}
    {...props}
  />
));
TabsTrigger.displayName = TabsPrimitive.Trigger.displayName;

export const TabsContent = React.forwardRef<
  React.ComponentRef<typeof TabsPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
>(({ className, ...props }, ref) => (
  <TabsPrimitive.Content
    ref={ref}
    className={cn('mt-3 focus-visible:outline-none', className)}
    {...props}
  />
));
TabsContent.displayName = TabsPrimitive.Content.displayName;
