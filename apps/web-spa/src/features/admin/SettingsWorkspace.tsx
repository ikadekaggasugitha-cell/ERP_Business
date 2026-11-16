import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Activity, Building2, Globe, Laptop, Moon, RefreshCw, ShieldCheck, Sun, Wifi, WifiOff } from 'lucide-react';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { Input } from '../../components/ui/Input';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { AppLocale, AppTheme, PortalMode } from '../../types';
import { probeGatewayHealth } from './admin-api';

export function SettingsWorkspace() {
  const activeOrg = useAuthStore((s) => s.activeOrg);
  return (
    <DomainErrorBoundary resetKey={activeOrg?.id}>
      <SettingsWorkspaceInner />
    </DomainErrorBoundary>
  );
}

function SettingsWorkspaceInner() {
  const { i18n } = useTranslation();
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const organizations = useAuthStore((s) => s.organizations);
  const isDemoSession = useAuthStore((s) => s.isDemoSession);

  const locale = useUiStore((s) => s.locale);
  const setLocale = useUiStore((s) => s.setLocale);
  const theme = useUiStore((s) => s.theme);
  const setTheme = useUiStore((s) => s.setTheme);
  const portal = useUiStore((s) => s.portal);
  const setPortal = useUiStore((s) => s.setPortal);
  const isRealtimeConnected = useUiStore((s) => s.isRealtimeConnected);

  const [permSearch, setPermSearch] = useState('');

  const gatewayHealthQuery = useQuery({
    queryKey: ['settings', 'gateway-health'],
    queryFn: probeGatewayHealth,
    retry: false,
    staleTime: 30_000,
  });

  const isId = locale === 'id';

  const handleLocaleChange = (next: AppLocale) => {
    setLocale(next);
    void i18n.changeLanguage(next);
  };

  const handleThemeChange = (next: AppTheme) => {
    setTheme(next);
  };

  const handlePortalChange = (next: PortalMode) => {
    setPortal(next);
  };

  const userPermissions = useMemo(() => user?.permissions ?? [], [user]);
  const filteredPermissions = useMemo(() => {
    const q = permSearch.trim().toLowerCase();
    if (!q) return userPermissions;
    return userPermissions.filter((p) => p.toLowerCase().includes(q));
  }, [permSearch, userPermissions]);

  const userRoles = useMemo(() => user?.roles ?? [], [user]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
          {isId ? 'Konsol Preferensi & Sistem' : 'System & Preferences Console'}
        </h1>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">
          {isId
            ? 'Atur bahasa, tema antarmuka, portal default, dan periksa kesehatan gateway serta hak akses sesi aktif.'
            : 'Configure workspace language, visual theme, default portal, and inspect live session RBAC & gateway health.'}
        </p>
      </div>

      {/* Preferences Row */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Language Selector */}
        <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
          <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
            <Globe className="h-4 w-4 text-brand-600" />
            {isId ? 'Bahasa Antarmuka' : 'Interface Language'}
          </div>
          <p className="mt-1 text-xs text-[var(--text-muted)]">
            {isId ? 'Pilihan bahasa disimpan secara persisten.' : 'Language preference persists across reloads.'}
          </p>
          <div className="mt-4 flex gap-2">
            <Button
              size="sm"
              variant={locale === 'id' ? 'primary' : 'outline'}
              onClick={() => handleLocaleChange('id')}
              aria-pressed={locale === 'id'}
            >
              ID · Indonesia
            </Button>
            <Button
              size="sm"
              variant={locale === 'en' ? 'primary' : 'outline'}
              onClick={() => handleLocaleChange('en')}
              aria-pressed={locale === 'en'}
            >
              EN · English
            </Button>
          </div>
        </section>

        {/* Visual Theme Selector */}
        <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
          <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
            {theme === 'dark' ? (
              <Moon className="h-4 w-4 text-brand-600" />
            ) : (
              <Sun className="h-4 w-4 text-amber-500" />
            )}
            {isId ? 'Tema Visual' : 'Visual Theme'}
          </div>
          <p className="mt-1 text-xs text-[var(--text-muted)]">
            {isId ? 'Mode terang atau gelap berdesain kontras tinggi.' : 'High-contrast Light or Dark workspace appearance.'}
          </p>
          <div className="mt-4 flex gap-2">
            <Button
              size="sm"
              variant={theme === 'light' ? 'primary' : 'outline'}
              onClick={() => handleThemeChange('light')}
              aria-pressed={theme === 'light'}
            >
              <Sun className="mr-1.5 h-3.5 w-3.5" />
              Light
            </Button>
            <Button
              size="sm"
              variant={theme === 'dark' ? 'primary' : 'outline'}
              onClick={() => handleThemeChange('dark')}
              aria-pressed={theme === 'dark'}
            >
              <Moon className="mr-1.5 h-3.5 w-3.5" />
              Dark
            </Button>
          </div>
        </section>

        {/* Default Portal Selector */}
        <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
          <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
            <Laptop className="h-4 w-4 text-brand-600" />
            {isId ? 'Portal Default' : 'Default Portal'}
          </div>
          <p className="mt-1 text-xs text-[var(--text-muted)]">
            {isId ? 'Pilih mode navigasi utama di sidebar.' : 'Choose primary navigation workspace mode.'}
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            {(['backoffice', 'admin', 'reporting'] as const).map((mode) => (
              <Button
                key={mode}
                size="sm"
                variant={portal === mode ? 'primary' : 'outline'}
                onClick={() => handlePortalChange(mode)}
                aria-pressed={portal === mode}
              >
                {mode.charAt(0).toUpperCase() + mode.slice(1)}
              </Button>
            ))}
          </div>
        </section>
      </div>

      {/* Entity Summary & Gateway/SSE Health */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
              <Building2 className="h-4 w-4 text-brand-600" />
              {isId ? 'Ringkasan Entitas Aktif' : 'Active Entity Summary'}
            </div>
            {isDemoSession && <StatusBadge status="DEMO" />}
          </div>
          <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Entity Name</dt>
              <dd className="mt-0.5 font-medium text-[var(--text-primary)]">
                {activeOrg?.name ?? '—'}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Entity Code</dt>
              <dd className="mt-0.5 font-mono text-xs text-[var(--text-primary)]">
                {activeOrg?.code ?? '—'}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Entity Type</dt>
              <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">
                {activeOrg?.type ?? '—'}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Functional Currency</dt>
              <dd className="mt-0.5 font-mono text-xs font-semibold text-[var(--text-primary)]">
                {activeOrg?.functionalCurrency ?? '—'}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Accessible Entities</dt>
              <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">
                {organizations.length}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-[var(--text-muted)]">Authenticated User</dt>
              <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">
                {user?.username ?? 'Anonymous'} ({user?.email ?? '—'})
              </dd>
            </div>
          </dl>
        </section>

        {/* Gateway & SSE Health */}
        <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
              <Activity className="h-4 w-4 text-emerald-600" />
              {isId ? 'Kesehatan Gateway & SSE' : 'Gateway & Realtime SSE Health'}
            </div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => void gatewayHealthQuery.refetch()}
              disabled={gatewayHealthQuery.isFetching}
            >
              <RefreshCw
                className={`mr-1.5 h-3.5 w-3.5 ${gatewayHealthQuery.isFetching ? 'animate-spin' : ''}`}
              />
              Probe
            </Button>
          </div>

          <div className="mt-4 space-y-3 text-sm">
            <div className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-subtle)] px-3 py-2.5">
              <div>
                <div className="font-medium text-[var(--text-primary)]">API Gateway (`/healthz`)</div>
                <div className="font-mono text-xs text-[var(--text-muted)]">
                  {gatewayHealthQuery.data?.endpoint ?? '/healthz'}
                  {gatewayHealthQuery.data?.detail ? ` · ${gatewayHealthQuery.data.detail}` : ''}
                </div>
              </div>
              <StatusBadge
                status={
                  gatewayHealthQuery.data?.status === 'online'
                    ? 'ACTIVE'
                    : gatewayHealthQuery.data?.status === 'degraded'
                      ? 'PENDING'
                      : 'INACTIVE'
                }
              />
            </div>

            <div className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-subtle)] px-3 py-2.5">
              <div className="flex items-center gap-2">
                {isRealtimeConnected ? (
                  <Wifi className="h-4 w-4 text-emerald-600" />
                ) : (
                  <WifiOff className="h-4 w-4 text-amber-600" />
                )}
                <div>
                  <div className="font-medium text-[var(--text-primary)]">
                    SSE Stream (`/api/v1/stream/events`)
                  </div>
                  <div className="text-xs text-[var(--text-muted)]">
                    {isRealtimeConnected
                      ? 'Subscribed to domain invalidation stream'
                      : 'Disconnected / Standby'}
                  </div>
                </div>
              </div>
              <StatusBadge status={isRealtimeConnected ? 'ACTIVE' : 'PENDING'} />
            </div>
          </div>
        </section>
      </div>

      {/* Session Permissions Inspector */}
      <section className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-emerald-600" />
            <h2 className="text-sm font-semibold text-[var(--text-primary)]">
              {isId ? 'Inspektor Izin & Peran Sesi' : 'Session Permissions Inspector'}
            </h2>
            {hasUserPermission(user, '*') && (
              <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 font-mono text-[11px] font-semibold text-emerald-700 dark:text-emerald-300">
                Wildcard Super-Admin
              </span>
            )}
          </div>
          <div className="w-60">
            <Input
              value={permSearch}
              onChange={(e) => setPermSearch(e.target.value)}
              placeholder="Filter permissions..."
            />
          </div>
        </div>

        <div className="mt-3 flex flex-wrap gap-1.5">
          <span className="text-xs font-medium text-[var(--text-muted)]">Roles:</span>
          {userRoles.length === 0 ? (
            <span className="text-xs text-[var(--text-muted)]">None</span>
          ) : (
            userRoles.map((role) => (
              <span
                key={role}
                className="rounded-md border border-[var(--border-subtle)] bg-[var(--surface-subtle)] px-2 py-0.5 font-mono text-xs text-[var(--text-primary)]"
              >
                {role}
              </span>
            ))
          )}
        </div>

        <div className="mt-4 flex flex-wrap gap-1.5">
          {filteredPermissions.length === 0 ? (
            <span className="text-xs text-[var(--text-muted)]">
              No matching permission claims in active session.
            </span>
          ) : (
            filteredPermissions.map((perm) => (
              <span
                key={perm}
                className="rounded-md bg-brand-500/10 px-2 py-0.5 font-mono text-xs text-brand-700 dark:text-brand-300"
              >
                {perm}
              </span>
            ))
          )}
        </div>
      </section>
    </div>
  );
}
