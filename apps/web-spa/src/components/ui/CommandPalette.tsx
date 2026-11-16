import * as React from 'react';
import * as DialogPrimitive from '@radix-ui/react-dialog';
import { Command } from 'cmdk';
import { useNavigate } from '@tanstack/react-router';
import {
  Search,
  Building2,
  Sun,
  Moon,
  Languages,
  ArrowRight,
  Compass,
  Layers,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  PORTAL_MODULES,
  filterNavByPortalAndPermission,
  hasUserPermission,
} from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { PortalMode } from '../../types';

export function CommandPalette() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { user, activeOrg, organizations, setActiveOrg, isAuthenticated } = useAuthStore();
  const {
    portal,
    commandPaletteOpen,
    setCommandPaletteOpen,
    theme,
    setTheme,
    locale,
    setLocale,
    setPortal,
  } = useUiStore();

  React.useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(!useUiStore.getState().commandPaletteOpen);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [setCommandPaletteOpen]);

  const permittedModules = React.useMemo(
    () =>
      PORTAL_MODULES.filter(
        (m) => m.permissions.length === 0 || hasUserPermission(user, m.permissions)
      ),
    [user]
  );

  const runAndClose = (fn: () => void) => {
    setCommandPaletteOpen(false);
    fn();
  };

  const navigateModule = (to: string, targetPortal: PortalMode) => {
    runAndClose(() => {
      setPortal(targetPortal);
      void navigate({ to: to as '/' });
    });
  };

  if (!isAuthenticated) return null;

  return (
    <DialogPrimitive.Root open={commandPaletteOpen} onOpenChange={setCommandPaletteOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-slate-950/50 backdrop-blur-xs" />
        <DialogPrimitive.Content
          aria-label={t('commandPalette.title')}
          className="fixed left-1/2 top-[14%] z-50 w-[92vw] max-w-xl -translate-x-1/2 overflow-hidden rounded-2xl border border-border bg-surface shadow-[var(--shadow-dialog)]"
        >
          <DialogPrimitive.Title className="sr-only">
            {t('commandPalette.title')}
          </DialogPrimitive.Title>
          <DialogPrimitive.Description className="sr-only">
            {t('commandPalette.placeholder')}
          </DialogPrimitive.Description>
          <Command
            label={t('commandPalette.title')}
            className="flex flex-col text-ink"
          >
            <div className="flex items-center gap-3 border-b border-border px-4 py-3">
              <Search className="h-4 w-4 text-brand-600 dark:text-brand-400 shrink-0" />
              <Command.Input
                autoFocus
                placeholder={t('commandPalette.placeholder')}
                className="w-full bg-transparent text-sm text-ink placeholder:text-slate-400 focus:outline-none"
              />
              <kbd className="hidden sm:inline-flex items-center rounded border border-border bg-surface-alt px-1.5 py-0.5 text-[10px] font-mono text-slate-500">
                ESC
              </kbd>
            </div>

            <Command.List className="max-h-96 overflow-y-auto p-2.5">
              <Command.Empty className="py-10 text-center text-xs text-slate-500 dark:text-slate-400">
                {t('commandPalette.noResults')}
              </Command.Empty>

              <Command.Group
                heading={t('commandPalette.groupNavigation')}
                className="px-2 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400"
              >
                {permittedModules.map((mod) => (
                  <Command.Item
                    key={mod.id}
                    value={`${t(mod.labelKey, mod.defaultLabel)} ${mod.serviceId} ${mod.keywords.join(' ')}`}
                    onSelect={() => navigateModule(mod.to, mod.portal)}
                    className="flex items-center justify-between rounded-xl border border-transparent px-2.5 py-2 text-sm font-normal text-slate-700 dark:text-slate-200 cursor-pointer aria-selected:border-brand-200 aria-selected:bg-brand-50/80 aria-selected:text-brand-800 dark:aria-selected:border-brand-800/70 dark:aria-selected:bg-slate-800 dark:aria-selected:text-brand-300"
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-border bg-surface-alt text-slate-500 dark:text-slate-400">
                        <Compass className="h-3.5 w-3.5" />
                      </span>
                      <span className="truncate font-medium">
                        {t(mod.labelKey, mod.defaultLabel)}
                      </span>
                      <span className="rounded border border-border-subtle bg-surface-muted px-1.5 py-0.5 text-[10px] font-mono text-slate-500 dark:text-slate-400">
                        {mod.serviceId}
                      </span>
                    </div>
                    <ArrowRight className="h-3.5 w-3.5 text-slate-400 shrink-0" />
                  </Command.Item>
                ))}
              </Command.Group>

              <Command.Separator className="my-1.5 h-px bg-slate-100 dark:bg-slate-800" />

              <Command.Group
                heading={t('commandPalette.groupQuickActions')}
                className="px-2 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400"
              >
                <Command.Item
                  value="toggle theme light dark mode tema terang gelap"
                  onSelect={() =>
                    runAndClose(() => setTheme(theme === 'dark' ? 'light' : 'dark'))
                  }
                  className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-slate-700 dark:text-slate-200 cursor-pointer aria-selected:bg-brand-50 dark:aria-selected:bg-slate-800"
                >
                  {theme === 'dark' ? (
                    <Sun className="h-4 w-4 text-amber-500" />
                  ) : (
                    <Moon className="h-4 w-4 text-slate-500" />
                  )}
                  <span>
                    {theme === 'dark'
                      ? t('commandPalette.switchToLight')
                      : t('commandPalette.switchToDark')}
                  </span>
                </Command.Item>

                <Command.Item
                  value="toggle language locale indonesia english bahasa"
                  onSelect={() =>
                    runAndClose(() => setLocale(locale === 'id' ? 'en' : 'id'))
                  }
                  className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-slate-700 dark:text-slate-200 cursor-pointer aria-selected:bg-brand-50 dark:aria-selected:bg-slate-800"
                >
                  <Languages className="h-4 w-4 text-brand-600" />
                  <span>
                    {locale === 'id'
                      ? t('commandPalette.switchToEn')
                      : t('commandPalette.switchToId')}
                  </span>
                </Command.Item>

                {(['backoffice', 'admin', 'reporting'] as PortalMode[]).map((p) => (
                  <Command.Item
                    key={p}
                    value={`portal ${p} ${t(`portals.${p}`)}`}
                    onSelect={() =>
                      runAndClose(() => {
                        setPortal(p);
                        const permitted = filterNavByPortalAndPermission(p, user);
                        const fallbackRoute = permitted[0]?.to;
                        if (fallbackRoute) {
                          void navigate({ to: fallbackRoute as '/' });
                        }
                      })
                    }
                    className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-slate-700 dark:text-slate-200 cursor-pointer aria-selected:bg-brand-50 dark:aria-selected:bg-slate-800"
                  >
                    <Layers className="h-4 w-4 text-slate-400" />
                    <span>
                      {t('commandPalette.switchPortalPrefix')}: {t(`portals.${p}`)}
                    </span>
                  </Command.Item>
                ))}

                {organizations.map((org) => (
                  <Command.Item
                    key={org.id}
                    value={`org entity branch holding ${org.code} ${org.name}`}
                    onSelect={() => runAndClose(() => setActiveOrg(org))}
                    className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-slate-700 dark:text-slate-200 cursor-pointer aria-selected:bg-brand-50 dark:aria-selected:bg-slate-800"
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <Building2 className="h-4 w-4 text-slate-400 shrink-0" />
                      <span className="truncate">
                        {org.code} — {org.name}
                      </span>
                    </div>
                    {activeOrg?.id === org.id ? (
                      <span className="text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                        ACTIVE
                      </span>
                    ) : null}
                  </Command.Item>
                ))}
              </Command.Group>
            </Command.List>

            <div className="flex items-center justify-between gap-2 border-t border-border-subtle dark:border-slate-800 bg-surface-alt px-4 py-2 text-[11px] text-slate-500 dark:text-slate-400">
              <span>{t('commandPalette.footerNavHint')}</span>
              <span className="font-semibold text-brand-700 dark:text-brand-400">
                Portal: {t(`portals.${portal}Short`)}
              </span>
            </div>
          </Command>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
