import * as React from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';
import {
  AlertCircle,
  AlertTriangle,
  Bell,
  CheckCheck,
  CheckCircle2,
  Info,
  Trash2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn, formatDateTime } from '../../lib/utils';
import { useUiStore, type UiNotification } from '../../store/ui-store';
import { Button } from '../ui/Button';

const TYPE_ICON: Record<UiNotification['type'], React.ReactNode> = {
  success: (
    <span className="flex h-7 w-7 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-800/70 dark:bg-emerald-950/60 dark:text-emerald-400 shrink-0">
      <CheckCircle2 className="h-3.5 w-3.5" />
    </span>
  ),
  warning: (
    <span className="flex h-7 w-7 items-center justify-center rounded-full border border-amber-200 bg-amber-50 text-amber-600 dark:border-amber-800/70 dark:bg-amber-950/60 dark:text-amber-400 shrink-0">
      <AlertTriangle className="h-3.5 w-3.5" />
    </span>
  ),
  error: (
    <span className="flex h-7 w-7 items-center justify-center rounded-full border border-rose-200 bg-rose-50 text-rose-600 dark:border-rose-800/70 dark:bg-rose-950/60 dark:text-rose-400 shrink-0">
      <AlertCircle className="h-3.5 w-3.5" />
    </span>
  ),
  info: (
    <span className="flex h-7 w-7 items-center justify-center rounded-full border border-brand-200 bg-brand-50 text-brand-600 dark:border-brand-800/70 dark:bg-brand-950/60 dark:text-brand-400 shrink-0">
      <Info className="h-3.5 w-3.5" />
    </span>
  ),
};

export function NotificationPopover() {
  const { t } = useTranslation();
  const { notifications, locale, markAsRead, markAllAsRead, clearNotifications } =
    useUiStore();
  const unreadCount = notifications.filter((n) => !n.read).length;

  return (
    <PopoverPrimitive.Root>
      <PopoverPrimitive.Trigger asChild>
        <button
          type="button"
          aria-label={t('app.notifications')}
          className="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-600 hover:bg-surface-muted hover:text-slate-900 dark:text-slate-300 dark:hover:text-slate-100 cursor-pointer transition-colors"
        >
          <Bell className="h-4.5 w-4.5" />
          {unreadCount > 0 ? (
            <span className="absolute right-1 top-1 inline-flex min-w-4 h-4 items-center justify-center rounded-full bg-rose-600 px-1 text-[10px] font-bold text-white leading-none">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          ) : null}
        </button>
      </PopoverPrimitive.Trigger>

      <PopoverPrimitive.Portal>
        <PopoverPrimitive.Content
          align="end"
          sideOffset={8}
          className="z-50 w-80 sm:w-96 overflow-hidden rounded-2xl border border-border bg-surface text-ink shadow-[var(--shadow-popover)]"
        >
          <div className="flex items-center justify-between border-b border-border-subtle dark:border-slate-800 px-4 py-3">
            <div className="flex items-center gap-2">
              <span className="text-sm font-bold">{t('app.notifications')}</span>
              {unreadCount > 0 ? (
                <span className="rounded-full border border-rose-200 bg-rose-50 px-2 py-0.5 text-[10.5px] font-semibold text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/60 dark:text-rose-300">
                  {t('app.newCount', { count: unreadCount })}
                </span>
              ) : null}
            </div>

            <div className="flex items-center gap-1">
              {unreadCount > 0 ? (
                <Button
                  variant="ghost"
                  size="xs"
                  leftIcon={<CheckCheck className="h-3.5 w-3.5" />}
                  onClick={markAllAsRead}
                  className="text-brand-700 dark:text-brand-400"
                >
                  {t('app.markAllRead')}
                </Button>
              ) : null}
              {notifications.length > 0 ? (
                <Button
                  variant="ghost"
                  size="xs"
                  aria-label={t('app.clearAll')}
                  onClick={clearNotifications}
                >
                  <Trash2 className="h-3.5 w-3.5 text-slate-400 hover:text-rose-600" />
                </Button>
              ) : null}
            </div>
          </div>

          <div className="max-h-80 overflow-y-auto divide-y divide-border-subtle dark:divide-slate-800">
            {notifications.length === 0 ? (
              <div className="py-10 px-4 text-center text-xs text-slate-500 dark:text-slate-400">
                {t('app.noNotifications')}
              </div>
            ) : (
              notifications.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => markAsRead(item.id)}
                  className={cn(
                    'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors cursor-pointer hover:bg-surface-alt dark:hover:bg-slate-800/60',
                    !item.read && 'bg-brand-50/40 dark:bg-brand-950/20'
                  )}
                >
                  <div className="mt-0.5">{TYPE_ICON[item.type]}</div>
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs font-semibold text-slate-900 dark:text-slate-100 truncate">
                        {item.title}
                      </span>
                      <span className="flex items-center gap-1.5 shrink-0">
                        <span className="text-[10px] font-mono text-slate-400 dark:text-slate-500">
                          {formatDateTime(item.timestamp, locale)}
                        </span>
                        {!item.read ? (
                          <span className="h-1.5 w-1.5 rounded-full bg-brand-600 dark:bg-brand-400" />
                        ) : null}
                      </span>
                    </div>
                    <p className="text-xs text-slate-600 dark:text-slate-300 leading-snug">
                      {item.message}
                    </p>
                  </div>
                </button>
              ))
            )}
          </div>
        </PopoverPrimitive.Content>
      </PopoverPrimitive.Portal>
    </PopoverPrimitive.Root>
  );
}
