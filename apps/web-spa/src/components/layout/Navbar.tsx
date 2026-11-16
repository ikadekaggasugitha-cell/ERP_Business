import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { Link, useNavigate } from '@tanstack/react-router';
import {
  Building2,
  Check,
  ChevronDown,
  LogOut,
  Menu,
  Moon,
  Search,
  Settings,
  Sun,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import { BrandLogo } from '../ui/BrandLogo';
import { PortalSwitcher } from '../ui/PortalSwitcher';
import { NotificationPopover } from './NotificationPopover';

export function Navbar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user, activeOrg, organizations, setActiveOrg, isDemoSession, logout } =
    useAuthStore();
  const {
    portal,
    theme,
    toggleTheme,
    locale,
    setLocale,
    isRealtimeConnected,
    demoMode,
    setDemoMode,
    setMobileDrawerOpen,
    setCommandPaletteOpen,
  } = useUiStore();

  const handleLogout = () => {
    setDemoMode(false);
    logout();
    void navigate({ to: '/login' });
  };

  const isOffline = isDemoSession || demoMode;
  const userInitials = (user?.fullName || user?.username || 'U').slice(0, 2).toUpperCase();
  const visiblePermissions = (user?.permissions ?? []).slice(0, 4);

  return (
    <header className="sticky top-0 z-30 h-14 border-b border-border bg-chrome/95 backdrop-blur-md">
      <div className="flex h-full items-center justify-between gap-2 px-3 sm:px-5">
        {/* Left: Mobile Drawer Button + Brand + Portal Mode Switcher */}
        <div className="flex items-center gap-2 sm:gap-3 min-w-0">
          <button
            type="button"
            onClick={() => setMobileDrawerOpen(true)}
            aria-label={t('app.openMenu')}
            className="inline-flex h-10 w-10 items-center justify-center rounded-lg text-slate-600 hover:bg-surface-muted md:hidden dark:text-slate-300 dark:hover:bg-slate-800 cursor-pointer"
          >
            <Menu className="h-5 w-5" />
          </button>

          <Link to="/" className="flex items-center gap-2.5 shrink-0">
            <BrandLogo size="md" />
            <div className="hidden sm:flex items-center gap-2">
              <span className="text-sm font-bold tracking-tight text-slate-900 dark:text-slate-100">
                {t('app.title')}
              </span>
              <span className="inline-flex items-center rounded-md border border-brand-200 bg-brand-50 px-1.5 py-0.5 text-[10px] font-mono font-semibold text-brand-700 dark:border-brand-800/70 dark:bg-brand-950/60 dark:text-brand-300">
                v2.4
              </span>
            </div>
          </Link>

          <div className="hidden lg:block ml-2">
            <PortalSwitcher />
          </div>
        </div>

        {/* Center/Right Utilities */}
        <div className="flex items-center gap-1.5 sm:gap-2">
          {/* Command Palette Trigger */}
          <button
            type="button"
            onClick={() => setCommandPaletteOpen(true)}
            aria-label={t('app.searchTrigger')}
            className="inline-flex h-9 items-center gap-2 rounded-lg border border-border bg-surface-alt px-2.5 text-xs text-slate-500 hover:border-border-strong hover:bg-surface-muted dark:bg-slate-800/70 dark:text-slate-400 dark:hover:bg-slate-800 cursor-pointer transition-colors"
          >
            <Search className="h-3.5 w-3.5 shrink-0" />
            <span className="hidden xl:inline max-w-44 truncate">
              {t('app.searchTrigger')}
            </span>
            <kbd className="hidden md:inline-flex rounded border border-border bg-surface px-1.5 py-0.5 text-[10px] font-mono text-slate-500 dark:text-slate-400">
              ⌘K
            </kbd>
          </button>

          {/* Organization Switcher */}
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                aria-label={t('app.orgSwitch')}
                className="inline-flex h-9 max-w-40 sm:max-w-56 md:max-w-64 items-center gap-1.5 rounded-lg border border-border bg-surface px-2.5 text-xs font-medium text-slate-700 hover:bg-surface-alt dark:text-slate-200 dark:hover:bg-slate-800 cursor-pointer transition-colors"
              >
                <Building2 className="h-3.5 w-3.5 text-brand-600 dark:text-brand-400 shrink-0" />
                <span className="truncate font-semibold text-slate-900 dark:text-slate-100">
                  {activeOrg?.code ?? 'ORG'}
                </span>
                <span className="hidden md:inline truncate text-slate-500 dark:text-slate-400">
                  {activeOrg?.name ? `· ${activeOrg.name}` : ''}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400 shrink-0" />
              </button>
            </DropdownMenu.Trigger>

            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                sideOffset={8}
                className="z-50 w-80 rounded-2xl border border-border bg-surface p-2 text-ink shadow-[var(--shadow-popover)]"
              >
                <div className="px-2.5 pt-1.5 pb-2 border-b border-border-subtle dark:border-slate-800 mb-1.5">
                  <DropdownMenu.Label className="text-xs font-bold text-slate-900 dark:text-slate-100">
                    {t('app.orgSwitchTitle')}
                  </DropdownMenu.Label>
                  <p className="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400 leading-snug">
                    {t('app.orgSwitchSubtitle')}
                  </p>
                </div>
                <div className="space-y-1">
                  {organizations.map((org) => {
                    const selected = activeOrg?.id === org.id;
                    return (
                      <DropdownMenu.Item
                        key={org.id}
                        onSelect={() => setActiveOrg(org)}
                        className={cn(
                          'flex items-center justify-between gap-2.5 rounded-xl border px-2.5 py-2 text-xs outline-none cursor-pointer transition-colors',
                          selected
                            ? 'border-brand-500 bg-brand-50/75 dark:border-brand-500/70 dark:bg-brand-950/40'
                            : 'border-border-subtle bg-surface-alt/60 hover:bg-surface-muted focus:bg-brand-50/60 dark:border-slate-800 dark:bg-slate-800/40 dark:focus:bg-slate-800'
                        )}
                      >
                        <div className="flex items-center gap-2.5 min-w-0">
                          <span
                            className={cn(
                              'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border',
                              selected
                                ? 'border-brand-600 bg-brand-600 text-white'
                                : 'border-border bg-surface text-slate-500 dark:text-slate-400'
                            )}
                          >
                            <Building2 className="h-4 w-4" />
                          </span>
                          <div className="min-w-0">
                            <div className="truncate font-semibold text-slate-900 dark:text-slate-100">
                              {org.code} · {org.name}
                            </div>
                            <div className="mt-0.5 flex items-center gap-1.5 text-[10.5px] text-slate-500 dark:text-slate-400">
                              <span className="font-mono">{org.functionalCurrency}</span>
                              <span>·</span>
                              <span className="truncate">{org.type}</span>
                            </div>
                          </div>
                        </div>
                        {selected ? (
                          <Check className="h-4 w-4 text-brand-600 dark:text-brand-400 shrink-0" />
                        ) : null}
                      </DropdownMenu.Item>
                    );
                  })}
                </div>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>

          {/* Live SSE Connection Indicator */}
          <div
            aria-live="polite"
            title={isOffline ? t('app.demoBannerTitle') : isRealtimeConnected ? t('app.liveSse') : t('app.offlineSse')}
            className={cn(
              'hidden sm:inline-flex h-8 items-center gap-1.5 rounded-full border px-2.5 text-[11px] font-semibold',
              isOffline
                ? 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800/70 dark:bg-amber-950/50 dark:text-amber-300'
                : isRealtimeConnected
                  ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/70 dark:bg-emerald-950/50 dark:text-emerald-300'
                  : 'border-border bg-surface-muted text-slate-600 dark:text-slate-300'
            )}
          >
            <span
              className={cn(
                'h-2 w-2 rounded-full',
                isOffline
                  ? 'bg-amber-500'
                  : isRealtimeConnected
                    ? 'bg-emerald-500 animate-pulse'
                    : 'bg-slate-400'
              )}
            />
            <span className="hidden xl:inline">
              {isOffline
                ? t('app.demoShort')
                : isRealtimeConnected
                  ? t('app.liveSse')
                  : t('app.offlineSse')}
            </span>
          </div>

          {/* Language Toggle ID/EN */}
          <button
            type="button"
            onClick={() => setLocale(locale === 'id' ? 'en' : 'id')}
            aria-label={t('app.switchLanguage')}
            className="inline-flex h-9 items-center justify-center rounded-lg border border-border bg-surface px-2.5 text-xs font-mono font-semibold text-slate-700 hover:bg-surface-muted dark:text-slate-200 cursor-pointer transition-colors"
          >
            {locale.toUpperCase()}
          </button>

          {/* Theme Toggle Light/Dark */}
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? t('app.lightMode') : t('app.darkMode')}
            title={theme === 'dark' ? t('app.lightMode') : t('app.darkMode')}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-600 hover:bg-surface-muted hover:text-slate-900 dark:text-slate-300 dark:hover:text-slate-100 cursor-pointer transition-colors"
          >
            {theme === 'dark' ? <Sun className="h-4.5 w-4.5 text-amber-400" /> : <Moon className="h-4.5 w-4.5" />}
          </button>

          {/* Notification Bell Popover */}
          <NotificationPopover />

          {/* User Profile Menu */}
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                aria-label={t('app.profile')}
                className="inline-flex h-9 items-center gap-2 rounded-lg border border-border bg-surface pl-1.5 pr-2.5 text-xs hover:bg-surface-alt dark:hover:bg-slate-800 cursor-pointer transition-colors"
              >
                <span className="flex h-6 w-6 items-center justify-center rounded-md bg-brand-600 text-white text-[11px] font-bold">
                  {userInitials}
                </span>
                <span className="hidden md:inline max-w-32 truncate font-semibold text-slate-800 dark:text-slate-100">
                  {user?.fullName || user?.username || 'User'}
                </span>
                <ChevronDown className="h-3.5 w-3.5 text-slate-400 hidden sm:inline" />
              </button>
            </DropdownMenu.Trigger>

            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                sideOffset={8}
                className="z-50 w-80 rounded-2xl border border-border bg-surface p-3 text-ink shadow-[var(--shadow-popover)]"
              >
                <div className="flex items-center gap-3 pb-3 border-b border-border-subtle dark:border-slate-800">
                  <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-brand-600 text-white text-sm font-bold">
                    {userInitials}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm font-bold truncate text-slate-900 dark:text-slate-100">
                      {user?.fullName || user?.username}
                    </p>
                    <p className="text-[11px] text-slate-500 dark:text-slate-400 truncate">
                      {user?.email || user?.username}
                      {user?.roles?.length ? ` · ${user.roles[0]}` : ''}
                    </p>
                  </div>
                </div>

                <div className="my-2.5 rounded-xl border border-border-subtle bg-surface-alt/90 dark:border-slate-800 dark:bg-slate-800/50 p-2.5 space-y-2">
                  <p className="text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    {t('app.accessContextLabel')}
                  </p>
                  {visiblePermissions.length > 0 ? (
                    <div className="flex flex-wrap gap-1">
                      {visiblePermissions.map((perm) => (
                        <span
                          key={perm}
                          className="inline-flex items-center rounded-full border border-brand-200 bg-brand-50 px-2 py-0.5 text-[10px] font-mono font-semibold text-brand-700 dark:border-brand-800/70 dark:bg-brand-950/60 dark:text-brand-300"
                        >
                          {perm}
                        </span>
                      ))}
                      {(user?.permissions?.length ?? 0) > visiblePermissions.length ? (
                        <span className="inline-flex items-center rounded-full border border-border bg-surface px-2 py-0.5 text-[10px] font-mono text-slate-500">
                          +{(user?.permissions?.length ?? 0) - visiblePermissions.length}
                        </span>
                      ) : null}
                    </div>
                  ) : null}
                  <div className="space-y-0.5 text-[11px] text-slate-600 dark:text-slate-300">
                    <p className="truncate">
                      {t('app.activeEntity')}:{' '}
                      <span className="font-semibold text-slate-800 dark:text-slate-100">
                        {activeOrg ? `${activeOrg.code} · ${activeOrg.name}` : '—'}
                      </span>
                    </p>
                    <p className="truncate text-slate-500 dark:text-slate-400">
                      Portal: {t(`portals.${portal}`)} · {activeOrg?.functionalCurrency ?? 'IDR'}
                    </p>
                  </div>
                </div>

                <div className="flex items-center justify-between gap-2 pt-1">
                  <DropdownMenu.Item
                    onSelect={() => void navigate({ to: '/settings' })}
                    className="flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border bg-surface px-2.5 py-2 text-xs font-semibold text-slate-700 outline-none cursor-pointer hover:bg-surface-alt focus:bg-surface-muted dark:text-slate-200"
                  >
                    <Settings className="h-3.5 w-3.5 text-slate-400" />
                    <span>{t('app.preferencesAndSecurity')}</span>
                  </DropdownMenu.Item>

                  <DropdownMenu.Item
                    onSelect={handleLogout}
                    className="flex items-center justify-center gap-1.5 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-700 outline-none cursor-pointer hover:bg-rose-100 focus:bg-rose-100 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300 dark:focus:bg-rose-950/80"
                  >
                    <LogOut className="h-3.5 w-3.5" />
                    <span>{t('app.logout')}</span>
                  </DropdownMenu.Item>
                </div>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
      </div>
    </header>
  );
}
