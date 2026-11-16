import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { Link, useRouterState } from '@tanstack/react-router';
import {
  BarChart3,
  BookOpen,
  CheckSquare,
  ChevronLeft,
  ChevronRight,
  Coins,
  Contact,
  CreditCard,
  Factory,
  FileSearch,
  FileSpreadsheet,
  FolderTree,
  Landmark,
  LayoutDashboard,
  Network,
  Package,
  Receipt,
  Settings,
  ShieldAlert,
  ShoppingCart,
  TrendingUp,
  UserCog,
  Users,
  Wallet,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  filterNavByPortalAndPermission,
  type PortalNavGroup,
  type PortalNavItem,
} from '../../lib/permissions';
import { cn } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import { BrandLogo } from '../ui/BrandLogo';
import { PortalSwitcher } from '../ui/PortalSwitcher';

const MODULE_ICONS: Record<string, typeof LayoutDashboard> = {
  dashboard: LayoutDashboard,
  journals: BookOpen,
  coa: FolderTree,
  ap: CreditCard,
  ar: Receipt,
  tax: FileSpreadsheet,
  treasury: Landmark,
  currency: Coins,
  procurement: ShoppingCart,
  inventory: Package,
  sales: TrendingUp,
  manufacturing: Factory,
  hr: Users,
  payroll: Wallet,
  workflow: CheckSquare,
  masterdata: Contact,
  organizations: Network,
  users: UserCog,
  security: ShieldAlert,
  audit: FileSearch,
  settings: Settings,
  reports: BarChart3,
};

const GROUP_ORDER: PortalNavGroup[] = [
  'overview',
  'finance',
  'supply_chain',
  'human_capital',
  'governance',
  'platform',
  'bi',
];

function groupNavItems(items: PortalNavItem[]): Array<{ group: PortalNavGroup; items: PortalNavItem[] }> {
  const map = new Map<PortalNavGroup, PortalNavItem[]>();
  for (const item of items) {
    const existing = map.get(item.group) ?? [];
    existing.push(item);
    map.set(item.group, existing);
  }
  return GROUP_ORDER.filter((g) => map.has(g)).map((g) => ({
    group: g,
    items: map.get(g) ?? [],
  }));
}

export function Sidebar() {
  const { t } = useTranslation();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const { user, activeOrg, isDemoSession } = useAuthStore();
  const {
    portal,
    sidebarCollapsed,
    toggleSidebar,
    mobileDrawerOpen,
    setMobileDrawerOpen,
    isRealtimeConnected,
    demoMode,
  } = useUiStore();

  const navItems = React.useMemo(
    () => filterNavByPortalAndPermission(portal, user),
    [portal, user]
  );
  const grouped = React.useMemo(() => groupNavItems(navItems), [navItems]);

  React.useEffect(() => {
    setMobileDrawerOpen(false);
  }, [pathname, setMobileDrawerOpen]);

  React.useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const mq = window.matchMedia('(min-width: 768px)');
    const onChange = (e: MediaQueryListEvent) => {
      if (e.matches) {
        setMobileDrawerOpen(false);
      }
    };
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [setMobileDrawerOpen]);

  const isRouteActive = (to: string) => {
    if (to === '/') return pathname === '/';
    return pathname === to || pathname.startsWith(`${to}/`);
  };

  const renderLinks = (collapsed: boolean, onSelect?: () => void) => (
    <nav aria-label="Main Navigation" className="flex-1 overflow-y-auto px-2.5 py-3 space-y-5">
      {grouped.map(({ group, items }) => (
        <div key={group} className="space-y-1">
          {!collapsed ? (
            <div className="px-2.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
              {t(`groups.${group}`)}
            </div>
          ) : (
            <div className="mx-2 h-px bg-slate-100 dark:bg-slate-800" />
          )}

          {items.map((item) => {
            const Icon = MODULE_ICONS[item.id] ?? LayoutDashboard;
            const active = isRouteActive(item.to);
            const label = t(item.labelKey, item.defaultLabel);

            return (
              <Link
                key={item.id}
                to={item.to as '/'}
                onClick={onSelect}
                aria-current={active ? 'page' : undefined}
                title={collapsed ? `${label} (${item.serviceId})` : undefined}
                className={cn(
                  'group relative flex items-center gap-3 rounded-lg border px-2.5 py-2 text-xs font-medium transition-colors',
                  collapsed && 'justify-center px-0',
                  active
                    ? 'border-brand-200/80 bg-brand-50/90 text-brand-700 font-semibold dark:border-brand-800/70 dark:bg-brand-950/60 dark:text-brand-300 before:absolute before:left-1 before:top-1.5 before:bottom-1.5 before:w-[3px] before:rounded-full before:bg-brand-600 dark:before:bg-brand-400'
                    : 'border-transparent text-slate-600 hover:bg-surface-muted hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800/80 dark:hover:text-slate-100'
                )}
              >
                <Icon
                  className={cn(
                    'h-4 w-4 shrink-0 transition-colors',
                    active
                      ? 'text-brand-700 dark:text-brand-300'
                      : 'text-slate-400 group-hover:text-slate-700 dark:text-slate-500 dark:group-hover:text-slate-200'
                  )}
                />
                {!collapsed ? (
                  <span className="flex-1 truncate">{label}</span>
                ) : null}
              </Link>
            );
          })}
        </div>
      ))}
    </nav>
  );

  const isOffline = isDemoSession || demoMode;

  return (
    <>
      {/* Desktop Collapsible Rail */}
      <aside
        className={cn(
          'hidden md:flex flex-col fixed left-0 top-14 bottom-0 z-20 border-r border-border bg-chrome transition-all duration-200',
          sidebarCollapsed ? 'w-16' : 'w-64'
        )}
      >
        {!sidebarCollapsed ? (
          <div className="border-b border-border-subtle dark:border-slate-800 p-2.5 lg:hidden">
            <PortalSwitcher compact />
          </div>
        ) : null}

        {renderLinks(sidebarCollapsed)}

        <div className="border-t border-border-subtle dark:border-slate-800 p-2.5 space-y-2">
          {!sidebarCollapsed ? (
            <div className="rounded-xl border border-border-subtle bg-surface-alt dark:border-slate-800 dark:bg-slate-900/70 px-3 py-2.5 text-[11px]">
              <div className="flex items-center gap-1.5 font-semibold text-slate-800 dark:text-slate-200">
                <span
                  className={cn(
                    'h-2 w-2 rounded-full shrink-0',
                    isOffline
                      ? 'bg-amber-500'
                      : isRealtimeConnected
                        ? 'bg-emerald-500'
                        : 'bg-slate-400'
                  )}
                />
                <span className="truncate">{t('app.sidebarComplianceTitle')}</span>
              </div>
              <p className="mt-1 truncate text-slate-600 dark:text-slate-400">
                {t('app.functionalShort')}:{' '}
                <span className="font-mono font-semibold text-slate-700 dark:text-slate-300">
                  {activeOrg?.functionalCurrency ?? 'IDR'}
                </span>{' '}
                · {activeOrg?.code ?? 'ORG'}
              </p>
            </div>
          ) : null}

          <button
            type="button"
            onClick={toggleSidebar}
            aria-label={
              sidebarCollapsed ? t('app.expandSidebar') : t('app.collapseSidebar')
            }
            className="flex w-full items-center justify-center gap-2 rounded-lg py-1.5 text-xs font-medium text-slate-500 hover:bg-surface-muted hover:text-slate-800 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200 cursor-pointer transition-colors"
          >
            {sidebarCollapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4" />
                <span>{t('app.collapseSidebar')}</span>
              </>
            )}
          </button>
        </div>
      </aside>

      {/* Mobile Slide-Over Navigation Drawer (Radix Dialog with Focus Trap + Escape) */}
      <DialogPrimitive.Root open={mobileDrawerOpen} onOpenChange={setMobileDrawerOpen}>
        <DialogPrimitive.Portal>
          <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/50 backdrop-blur-xs md:hidden" />
          <DialogPrimitive.Content
            aria-label={t('nav.mobileMenuLabel')}
            className="fixed inset-y-0 left-0 z-50 flex w-72 max-w-[85vw] flex-col border-r border-border bg-chrome shadow-2xl focus:outline-none md:hidden"
          >
            <DialogPrimitive.Description className="sr-only">
              {t('app.subtitle')}
            </DialogPrimitive.Description>
            <div className="flex h-14 items-center justify-between border-b border-border-subtle dark:border-slate-800 px-4">
              <div className="flex items-center gap-2">
                <BrandLogo size="sm" />
                <DialogPrimitive.Title className="text-sm font-bold text-slate-900 dark:text-slate-100">
                  {t('app.title')}
                </DialogPrimitive.Title>
              </div>
              <DialogPrimitive.Close
                aria-label={t('app.closeMenu')}
                className="inline-flex h-10 w-10 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 cursor-pointer"
              >
                <X className="h-5 w-5" />
              </DialogPrimitive.Close>
            </div>

            <div className="border-b border-slate-100 dark:border-slate-800 p-3">
              <PortalSwitcher compact />
            </div>

            {renderLinks(false, () => setMobileDrawerOpen(false))}
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      </DialogPrimitive.Root>
    </>
  );
}
