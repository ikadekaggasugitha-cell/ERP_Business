import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Building2, CalendarDays, FolderTree, Layers } from 'lucide-react';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { KpiCard } from '../../components/ui/KpiCard';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { apiClient, unwrapListResponse } from '../../lib/api';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { ListEnvelope } from '../../types';
import { AccountsTreeTab } from './AccountsTreeTab';
import {
  type AccountDto,
  type ChartOfAccountsDto,
  type CoaCurrencyLookupDto,
  type FinancialOrganizationDto,
  type FiscalCalendarDto,
  type FiscalYearDto,
  type LedgerDto,
  type OrgUnitLookupDto,
} from './coa-types';
import { FiscalPeriodsTab } from './FiscalPeriodsTab';
import { LedgersFinOrgsTab } from './LedgersFinOrgsTab';

export function CoaWorkspace() {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const user = useAuthStore((s) => s.user);
  const orgId = activeOrg?.id ?? 'default';

  const [activeTab, setActiveTab] = useState<'accounts' | 'ledgers' | 'fiscal'>('accounts');

  const canWrite = hasUserPermission(user, ['coa:write', 'coa.write', 'coa.*']);
  const canReadCurrencies = hasUserPermission(user, ['currency:read', 'currency.read', 'currency.*']);
  const canReadOrgs = hasUserPermission(user, ['org:read', 'org.read', 'org.*']);

  const chartsQuery = useQuery({
    queryKey: ['coa-accounts', 'charts', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<ChartOfAccountsDto> | ChartOfAccountsDto[]>(
        '/charts-of-accounts',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const accountsQuery = useQuery({
    queryKey: ['coa-accounts', 'list', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<AccountDto> | AccountDto[]>('/accounts', {
        query: { page: 1, size: 100 },
      });
      return unwrapListResponse(res).items;
    },
  });

  const finOrgsQuery = useQuery({
    queryKey: ['coa-accounts', 'fin-orgs', orgId],
    queryFn: async () => {
      const res = await apiClient<
        ListEnvelope<FinancialOrganizationDto> | FinancialOrganizationDto[]
      >('/financial-organizations', { query: { page: 1, size: 100 } });
      return unwrapListResponse(res).items;
    },
  });

  const ledgersQuery = useQuery({
    queryKey: ['ledgers', 'list', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<LedgerDto> | LedgerDto[]>('/ledgers', {
        query: { page: 1, size: 100 },
      });
      return unwrapListResponse(res).items;
    },
  });

  const calendarsQuery = useQuery({
    queryKey: ['fiscal-periods', 'calendars', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<FiscalCalendarDto> | FiscalCalendarDto[]>(
        '/fiscal-calendars',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const fiscalYearsQuery = useQuery({
    queryKey: ['fiscal-periods', 'years', orgId],
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<FiscalYearDto> | FiscalYearDto[]>(
        '/fiscal-years',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const currenciesQuery = useQuery({
    queryKey: ['currencies', 'coa-lookup', orgId],
    enabled: canReadCurrencies,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<CoaCurrencyLookupDto> | CoaCurrencyLookupDto[]>(
        '/currencies',
        { query: new URLSearchParams({ isActive: 'true', page: '1', size: '100' }) }
      );
      return unwrapListResponse(res).items;
    },
  });

  const orgUnitsQuery = useQuery({
    queryKey: ['organizations', 'coa-lookup', orgId],
    enabled: canReadOrgs,
    queryFn: async () => {
      const res = await apiClient<ListEnvelope<OrgUnitLookupDto> | OrgUnitLookupDto[]>(
        '/organizations',
        { query: { page: 1, size: 100 } }
      );
      return unwrapListResponse(res).items;
    },
  });

  const charts = useMemo(() => chartsQuery.data ?? [], [chartsQuery.data]);
  const accounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);
  const financialOrgs = useMemo(() => finOrgsQuery.data ?? [], [finOrgsQuery.data]);
  const ledgers = useMemo(() => ledgersQuery.data ?? [], [ledgersQuery.data]);
  const calendars = useMemo(() => calendarsQuery.data ?? [], [calendarsQuery.data]);
  const fiscalYears = useMemo(() => fiscalYearsQuery.data ?? [], [fiscalYearsQuery.data]);
  const currencies = useMemo(() => currenciesQuery.data ?? [], [currenciesQuery.data]);
  const orgUnits = useMemo(() => orgUnitsQuery.data ?? [], [orgUnitsQuery.data]);

  const postableCount = useMemo(
    () => accounts.filter((a) => a.allowPosting).length,
    [accounts]
  );

  return (
    <div className="space-y-6">
      <header>
        <Breadcrumbs
          items={[{ label: isId ? 'Bagan Akun & Periode' : 'Chart of Accounts & Periods' }]}
        />
        <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
          {isId ? 'Bagan Akun, Buku Besar & Periode Fiskal' : 'Chart of Accounts, Ledgers & Fiscal Periods'}
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {isId
            ? 'Kelola hierarki akun akuntansi, entitas keuangan, buku besar (Ledger), serta tutup/buka/kunci periode fiskal.'
            : 'Manage GL account hierarchy, financial organizations, ledgers, and fiscal period lifecycle transitions.'}
        </p>
      </header>

      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label={isId ? 'Bagan Akun (CoA)' : 'Charts of Accounts'}
          value={String(charts.length)}
          subtitle={`${accounts.length} ${isId ? 'total akun' : 'total accounts'} · ${postableCount} ${isId ? 'postable' : 'postable'}`}
          accent="brand"
          icon={<FolderTree className="h-5 w-5" />}
          isLoading={chartsQuery.isLoading || accountsQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Entitas Keuangan' : 'Financial Orgs'}
          value={String(financialOrgs.length)}
          subtitle={isId ? 'Pemetaan CoA & Kalender' : 'Mapped CoA & Calendar'}
          accent="sky"
          icon={<Building2 className="h-5 w-5" />}
          isLoading={finOrgsQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Buku Besar (Ledgers)' : 'Accounting Ledgers'}
          value={String(ledgers.length)}
          subtitle={isId ? 'Primer / Sekunder / Konsolidasi' : 'Primary / Secondary / Consolidation'}
          accent="emerald"
          icon={<Layers className="h-5 w-5" />}
          isLoading={ledgersQuery.isLoading}
        />
        <KpiCard
          label={isId ? 'Tahun & Kalender Fiskal' : 'Fiscal Years & Calendars'}
          value={String(fiscalYears.length)}
          subtitle={`${calendars.length} ${isId ? 'kalender fiskal aktif' : 'active calendars'}`}
          accent="amber"
          icon={<CalendarDays className="h-5 w-5" />}
          isLoading={calendarsQuery.isLoading || fiscalYearsQuery.isLoading}
        />
      </section>

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as 'accounts' | 'ledgers' | 'fiscal')}
      >
        <TabsList>
          <TabsTrigger value="accounts">
            {isId ? 'Akun & Hierarki Tree' : 'Accounts & Tree'}
          </TabsTrigger>
          <TabsTrigger value="ledgers">
            {isId ? 'Ledger & Entitas Keuangan' : 'Ledgers & Financial Orgs'}
          </TabsTrigger>
          <TabsTrigger value="fiscal">
            {isId ? 'Kalender, Tahun & Periode Fiskal' : 'Fiscal Calendars & Periods'}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="accounts">
          <AccountsTreeTab
            charts={charts}
            accounts={accounts}
            isLoading={chartsQuery.isLoading || accountsQuery.isLoading}
            error={chartsQuery.error ?? accountsQuery.error}
            onRetry={() => {
              void chartsQuery.refetch();
              void accountsQuery.refetch();
            }}
            canWrite={canWrite}
          />
        </TabsContent>

        <TabsContent value="ledgers">
          <LedgersFinOrgsTab
            financialOrgs={financialOrgs}
            ledgers={ledgers}
            charts={charts}
            calendars={calendars}
            currencies={currencies}
            orgUnits={orgUnits}
            isLoading={finOrgsQuery.isLoading || ledgersQuery.isLoading}
            error={finOrgsQuery.error ?? ledgersQuery.error}
            onRetry={() => {
              void finOrgsQuery.refetch();
              void ledgersQuery.refetch();
            }}
            canWrite={canWrite}
          />
        </TabsContent>

        <TabsContent value="fiscal">
          <FiscalPeriodsTab
            calendars={calendars}
            fiscalYears={fiscalYears}
            isLoading={calendarsQuery.isLoading || fiscalYearsQuery.isLoading}
            error={calendarsQuery.error ?? fiscalYearsQuery.error}
            onRetry={() => {
              void calendarsQuery.refetch();
              void fiscalYearsQuery.refetch();
            }}
            canWrite={canWrite}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
