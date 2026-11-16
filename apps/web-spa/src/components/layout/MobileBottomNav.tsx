import { Link, useRouterState } from '@tanstack/react-router';
import {
  BarChart3,
  BookOpen,
  CheckSquare,
  LayoutDashboard,
  Menu,
  Network,
  Receipt,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { hasUserPermission } from '../../lib/permissions';
import { cn } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';

export function MobileBottomNav() {
  const { t } = useTranslation();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const user = useAuthStore((s) => s.user);
  const { portal, setMobileDrawerOpen } = useUiStore();

  const rawLinks =
    portal === 'admin'
      ? [
          {
            to: '/organizations',
            label: t('nav.organizations'),
            icon: Network,
            permissions: ['org:read', 'org.read'],
          },
          {
            to: '/audit',
            label: t('nav.audit'),
            icon: BarChart3,
            permissions: ['audit:read', 'audit.read'],
          },
          {
            to: '/settings',
            label: t('nav.settings'),
            icon: CheckSquare,
            permissions: [],
          },
        ]
      : portal === 'reporting'
        ? [
            {
              to: '/reports',
              label: t('nav.reports'),
              icon: BarChart3,
              permissions: ['reporting.read', 'reporting:read'],
            },
            {
              to: '/',
              label: t('nav.dashboard'),
              icon: LayoutDashboard,
              permissions: [],
            },
          ]
        : [
            {
              to: '/',
              label: t('nav.dashboard'),
              icon: LayoutDashboard,
              permissions: [],
            },
            {
              to: '/journals',
              label: t('nav.journalsShort'),
              icon: BookOpen,
              permissions: ['gl:read', 'gl.read', 'gl.*'],
            },
            {
              to: '/ap',
              label: t('nav.apArShort'),
              icon: Receipt,
              permissions: ['ap:read', 'ap.read'],
            },
            {
              to: '/workflow',
              label: t('nav.approvalShort'),
              icon: CheckSquare,
              permissions: ['workflow:read', 'workflow.read'],
            },
          ];

  const quickLinks = rawLinks.filter(
    (item) => item.permissions.length === 0 || hasUserPermission(user, item.permissions)
  );

  return (
    <nav
      aria-label={t('nav.mobileQuickLabel')}
      className="fixed bottom-0 inset-x-0 z-30 flex items-center justify-around border-t border-border bg-chrome/95 backdrop-blur-md px-1 py-1 md:hidden"
    >
      {quickLinks.map((item) => {
        const Icon = item.icon;
        const active = item.to === '/' ? pathname === '/' : pathname.startsWith(item.to);
        return (
          <Link
            key={item.to}
            to={item.to as '/'}
            aria-current={active ? 'page' : undefined}
            className={cn(
              'flex min-h-11 min-w-11 flex-1 flex-col items-center justify-center gap-0.5 rounded-lg px-2 py-1 text-[10px] font-medium transition-colors',
              active
                ? 'text-brand-700 dark:text-brand-400 font-semibold'
                : 'text-slate-500 dark:text-slate-400'
            )}
          >
            <Icon className="h-4 w-4" />
            <span className="truncate max-w-16">{item.label}</span>
          </Link>
        );
      })}

      <button
        type="button"
        onClick={() => setMobileDrawerOpen(true)}
        aria-label={t('app.openMenu')}
        className="flex min-h-11 min-w-11 flex-1 flex-col items-center justify-center gap-0.5 rounded-lg px-2 py-1 text-[10px] font-medium text-slate-500 dark:text-slate-400 cursor-pointer"
      >
        <Menu className="h-4 w-4" />
        <span>{t('app.menu')}</span>
      </button>
    </nav>
  );
}
