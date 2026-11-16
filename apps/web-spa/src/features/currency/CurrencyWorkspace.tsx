import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeftRight,
  Coins,
  Globe2,
  Layers,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { Button } from '../../components/ui/Button';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { ApiClientError } from '../../lib/api';
import { exportToCsv } from '../../lib/export';
import { useTranslation } from 'react-i18next';
import { formatDate, formatRate } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  currencyQueryKeys,
  deleteCurrency,
  deleteExchangeRate,
  fetchCurrencies,
  fetchExchangeRates,
  fetchExchangeRateTypes,
} from './api';
import {
  CurrencyFormDialog,
  ExchangeRateFormDialog,
  RateTypeFormDialog,
} from './CurrencyDialogs';
import { FxConverterCard } from './FxConverterCard';
import type { CurrencyDto, ExchangeRateDto } from './types';

const CORE_ISO_CODES = new Set(['IDR', 'USD', 'EUR', 'SGD', 'JPY', 'GBP', 'AUD', 'CNY', 'MYR']);

export function CurrencyWorkspace() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';
  const canWrite = useAuthStore((s) => s.hasPermission)('currency:write');

  const [activeTab, setActiveTab] = useState('currencies');

  const [currPage, setCurrPage] = useState(1);
  const [currActiveFilter, setCurrActiveFilter] = useState('ALL');
  const [currSearch, setCurrSearch] = useState('');
  const [currDialogOpen, setCurrDialogOpen] = useState(false);
  const [editingCurrency, setEditingCurrency] = useState<CurrencyDto | null>(null);
  const [selectedCurrency, setSelectedCurrency] = useState<CurrencyDto | null>(null);
  const [confirmDeleteCurrency, setConfirmDeleteCurrency] = useState(false);

  const [rateTypeDialogOpen, setRateTypeDialogOpen] = useState(false);

  const [ratePage, setRatePage] = useState(1);
  const [rateTypeFilter, setRateTypeFilter] = useState('ALL');
  const [rateSearch, setRateSearch] = useState('');
  const [rateDialogOpen, setRateDialogOpen] = useState(false);
  const [editingRate, setEditingRate] = useState<ExchangeRateDto | null>(null);
  const [selectedRate, setSelectedRate] = useState<ExchangeRateDto | null>(null);
  const [confirmDeleteRate, setConfirmDeleteRate] = useState(false);

  const currenciesQuery = useQuery({
    queryKey: currencyQueryKeys.currenciesList({
      page: currPage,
      size: 50,
      isActive: currActiveFilter,
    }),
    queryFn: () =>
      fetchCurrencies({
        page: currPage,
        size: 50,
        isActive: currActiveFilter,
      }),
  });

  const rateTypesQuery = useQuery({
    queryKey: currencyQueryKeys.rateTypesList({ page: 1, size: 50 }),
    queryFn: () => fetchExchangeRateTypes({ page: 1, size: 50 }),
  });

  const ratesQuery = useQuery({
    queryKey: currencyQueryKeys.ratesList({
      page: ratePage,
      size: 25,
      type: rateTypeFilter,
    }),
    queryFn: () =>
      fetchExchangeRates({
        page: ratePage,
        size: 25,
        type: rateTypeFilter,
      }),
  });

  const currencies = useMemo(() => currenciesQuery.data?.items ?? [], [currenciesQuery.data]);
  const currPageMeta = currenciesQuery.data?.page ?? {
    number: currPage,
    size: 50,
    totalElements: currencies.length,
    totalPages: 1,
  };

  const rateTypes = useMemo(() => rateTypesQuery.data?.items ?? [], [rateTypesQuery.data]);

  const rates = useMemo(() => ratesQuery.data?.items ?? [], [ratesQuery.data]);
  const ratePageMeta = ratesQuery.data?.page ?? {
    number: ratePage,
    size: 25,
    totalElements: rates.length,
    totalPages: 1,
  };

  const filteredCurrencies = useMemo(() => {
    const q = currSearch.trim().toLowerCase();
    if (!q) return currencies;
    return currencies.filter(
      (c) =>
        c.code.toLowerCase().includes(q) ||
        c.name.toLowerCase().includes(q) ||
        (c.numericCode ?? '').toLowerCase().includes(q),
    );
  }, [currencies, currSearch]);

  const filteredRates = useMemo(() => {
    const q = rateSearch.trim().toLowerCase();
    if (!q) return rates;
    return rates.filter(
      (r) =>
        r.fromCurrency.toLowerCase().includes(q) ||
        r.toCurrency.toLowerCase().includes(q) ||
        r.rateTypeCode.toLowerCase().includes(q) ||
        r.rateTypeName.toLowerCase().includes(q),
    );
  }, [rates, rateSearch]);

  const activeCurrencyCount = useMemo(
    () => currencies.filter((c) => c.isActive).length,
    [currencies],
  );

  const deleteCurrencyMutation = useMutation({
    mutationFn: (curr: CurrencyDto) => deleteCurrency(curr.id, curr.rowVersion),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: currencyQueryKeys.currenciesAll });
      setConfirmDeleteCurrency(false);
      setSelectedCurrency(null);
      toast.success('Currency deleted');
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : 'Unable to delete currency (system ISO-4217 entries are protected)';
      toast.error(msg);
    },
  });

  const deleteRateMutation = useMutation({
    mutationFn: (rate: ExchangeRateDto) => deleteExchangeRate(rate.id, rate.rowVersion),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: currencyQueryKeys.ratesAll });
      setConfirmDeleteRate(false);
      setSelectedRate(null);
      toast.success('Exchange rate removed');
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Unable to delete exchange rate';
      toast.error(msg);
    },
  });

  const currencyColumns: ColumnDef<CurrencyDto>[] = [
    {
      header: 'Code',
      accessorKey: 'code',
      width: '140px',
      cell: (row) => (
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs font-bold text-slate-900">{row.code}</span>
          {CORE_ISO_CODES.has(row.code) && (
            <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600">
              ISO
            </span>
          )}
        </div>
      ),
    },
    {
      header: 'Numeric',
      width: '100px',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-600">{row.numericCode || '—'}</span>
      ),
    },
    {
      header: 'Currency Name',
      accessorKey: 'name',
      width: '260px',
      cell: (row) => <span className="text-sm font-medium text-slate-900">{row.name}</span>,
    },
    {
      header: 'Precision',
      align: 'right',
      width: '110px',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-700">{row.precision} decimals</span>
      ),
    },
    {
      header: 'Status',
      width: '120px',
      cell: (row) => (
        <StatusBadge
          status={row.isActive ? 'ACTIVE' : 'INACTIVE'}
          label={row.isActive ? 'Active' : 'Inactive'}
          tone={row.isActive ? 'success' : 'neutral'}
        />
      ),
    },
  ];

  const rateColumns: ColumnDef<ExchangeRateDto>[] = [
    {
      header: 'Currency Pair',
      width: '170px',
      cell: (row) => (
        <span className="font-mono text-xs font-bold text-slate-900">
          {row.fromCurrency} → {row.toCurrency}
        </span>
      ),
    },
    {
      header: 'Rate Type',
      width: '150px',
      cell: (row) => (
        <div>
          <span className="inline-flex items-center rounded bg-brand-50 px-2 py-0.5 font-mono text-xs font-semibold text-brand-700">
            {row.rateTypeCode}
          </span>
        </div>
      ),
    },
    {
      header: 'Rate',
      align: 'right',
      width: '180px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900 tabular-nums">
          {formatRate(row.rate, numberLocale)}
        </span>
      ),
    },
    {
      header: 'Valid From',
      width: '130px',
      cell: (row) => (
        <span className="text-xs text-slate-700">{formatDate(row.validFrom, numberLocale)}</span>
      ),
    },
    {
      header: 'Valid To',
      width: '130px',
      cell: (row) => (
        <span className="text-xs text-slate-500">
          {row.validTo ? formatDate(row.validTo, numberLocale) : 'Open-ended'}
        </span>
      ),
    },
    {
      header: 'Version',
      align: 'right',
      width: '90px',
      cell: (row) => <span className="font-mono text-xs text-slate-500">v{row.rowVersion}</span>,
    },
  ];

  const handleExportRates = () => {
    exportToCsv(
      filteredRates,
      [
        { header: 'From', accessor: (r) => r.fromCurrency },
        { header: 'To', accessor: (r) => r.toCurrency },
        { header: 'Rate Type', accessor: (r) => r.rateTypeCode },
        { header: 'Rate', accessor: (r) => r.rate },
        { header: 'Valid From', accessor: (r) => r.validFrom },
        { header: 'Valid To', accessor: (r) => r.validTo ?? '' },
      ],
      'exchange-rates.csv',
    );
    toast.success('Exchange rates exported to CSV');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              {t('nav.currency')} & FX Triangulation
            </h1>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  currenciesQuery.isError || ratesQuery.isError
                    ? 'bg-rose-500'
                    : currenciesQuery.isFetching || ratesQuery.isFetching
                      ? 'bg-amber-500'
                      : 'bg-emerald-500'
                }`}
              />
              SVC-007 · FX
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            ISO-4217 multi-currency registry, SPOT/CLOSING/AVERAGE exchange rates, and FX triangulation.
          </p>
        </div>

        {canWrite && (
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              leftIcon={<Layers className="h-4 w-4" />}
              onClick={() => setRateTypeDialogOpen(true)}
            >
              New Rate Type
            </Button>
            <Button
              variant="outline"
              leftIcon={<Coins className="h-4 w-4" />}
              onClick={() => {
                setEditingCurrency(null);
                setCurrDialogOpen(true);
              }}
            >
              Add Currency
            </Button>
            <Button
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingRate(null);
                setRateDialogOpen(true);
              }}
            >
              New Exchange Rate
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Currencies"
          value={currPageMeta.totalElements}
          subtitle={`${activeCurrencyCount} active (includes system ISO-4217)`}
          accent="brand"
          icon={<Globe2 className="h-4 w-4" />}
          isLoading={currenciesQuery.isLoading}
        />
        <KpiCard
          label="Exchange Rate Quotes"
          value={ratePageMeta.totalElements}
          subtitle="Direct & anchor pairs"
          accent="sky"
          icon={<ArrowLeftRight className="h-4 w-4" />}
          isLoading={ratesQuery.isLoading}
        />
        <KpiCard
          label="Rate Types"
          value={rateTypes.length}
          subtitle={
            rateTypes.length > 0
              ? rateTypes.map((rt) => rt.code).join(' · ')
              : 'SPOT · CLOSING · AVERAGE'
          }
          accent="emerald"
          icon={<Layers className="h-4 w-4" />}
          isLoading={rateTypesQuery.isLoading}
        />
        <KpiCard
          label="Triangulation Engine"
          value="Active"
          subtitle="Identity, Direct, Inverse, Anchor"
          accent="amber"
          icon={<Coins className="h-4 w-4" />}
        />
      </div>

      <FxConverterCard currencies={currencies} rateTypes={rateTypes} />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList variant="underline">
          <TabsTrigger value="currencies" variant="underline">
            Currencies ({currPageMeta.totalElements})
          </TabsTrigger>
          <TabsTrigger value="rates" variant="underline">
            Exchange Rates & Rate Types ({ratePageMeta.totalElements})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="currencies">
          <div className="mb-3 rounded-xl border border-slate-200 bg-slate-50 px-4 py-2.5 text-xs text-slate-600">
            Global ISO-4217 currencies are transparently shared alongside your organization&apos;s
            custom currencies via system-scope resolution.
          </div>

          <VirtualizedTable
            data={filteredCurrencies}
            columns={currencyColumns}
            isLoading={currenciesQuery.isLoading}
            error={currenciesQuery.error}
            onRetry={() => void currenciesQuery.refetch()}
            emptyTitle="No currencies available"
            emptyDescription="Add an ISO-4217 currency to enable multi-currency transactions and FX rates."
            onResetFilters={
              currSearch || currActiveFilter !== 'ALL'
                ? () => {
                    setCurrSearch('');
                    setCurrActiveFilter('ALL');
                    setCurrPage(1);
                  }
                : undefined
            }
            onRowClick={(row) => {
              setConfirmDeleteCurrency(false);
              setSelectedCurrency(row);
            }}
            pagination={{
              page: currPageMeta.number || currPage,
              size: currPageMeta.size || 50,
              totalElements: currPageMeta.totalElements,
              totalPages: currPageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setCurrPage,
            }}
            toolbar={{
              searchValue: currSearch,
              onSearchChange: setCurrSearch,
              searchPlaceholder: 'Search currency code, numeric code, name...',
              filters: (
                <Select
                  aria-label="Filter active currencies"
                  className="w-40"
                  value={currActiveFilter}
                  onValueChange={(value) => {
                    setCurrActiveFilter(value);
                    setCurrPage(1);
                  }}
                  options={[
                    { value: 'ALL', label: 'All Currencies' },
                    { value: 'true', label: 'Active Only' },
                    { value: 'false', label: 'Inactive Only' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="rates">
          <div className="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-slate-200 bg-white p-3">
            <span className="text-xs font-semibold text-slate-600">Rate Types:</span>
            {rateTypes.length > 0 ? (
              rateTypes.map((rt) => (
                <span
                  key={rt.id}
                  className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-800"
                >
                  <span className="font-mono font-bold">{rt.code}</span>
                  <span className="text-slate-500">{rt.name}</span>
                </span>
              ))
            ) : (
              <span className="text-xs text-slate-400">No rate types loaded</span>
            )}
          </div>

          <VirtualizedTable
            data={filteredRates}
            columns={rateColumns}
            isLoading={ratesQuery.isLoading}
            error={ratesQuery.error}
            onRetry={() => void ratesQuery.refetch()}
            emptyTitle="No exchange rates published"
            emptyDescription="Publish SPOT, CLOSING, or AVERAGE exchange rates to enable FX triangulation."
            onResetFilters={
              rateSearch || rateTypeFilter !== 'ALL'
                ? () => {
                    setRateSearch('');
                    setRateTypeFilter('ALL');
                    setRatePage(1);
                  }
                : undefined
            }
            onRowClick={(row) => {
              setConfirmDeleteRate(false);
              setSelectedRate(row);
            }}
            pagination={{
              page: ratePageMeta.number || ratePage,
              size: ratePageMeta.size || 25,
              totalElements: ratePageMeta.totalElements,
              totalPages: ratePageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setRatePage,
            }}
            toolbar={{
              searchValue: rateSearch,
              onSearchChange: setRateSearch,
              searchPlaceholder: 'Search currency pair, rate type...',
              onExportCsv: filteredRates.length > 0 ? handleExportRates : undefined,
              filters: (
                <Select
                  aria-label="Filter rate type"
                  className="w-44"
                  value={rateTypeFilter}
                  onValueChange={(value) => {
                    setRateTypeFilter(value);
                    setRatePage(1);
                  }}
                  options={[
                    { value: 'ALL', label: 'All Rate Types' },
                    ...(rateTypes.length > 0
                      ? rateTypes.map((rt) => ({ value: rt.code, label: rt.code }))
                      : [
                          { value: 'SPOT', label: 'SPOT' },
                          { value: 'CLOSING', label: 'CLOSING' },
                          { value: 'AVERAGE', label: 'AVERAGE' },
                        ]),
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedCurrency)}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmDeleteCurrency(false);
            setSelectedCurrency(null);
          }
        }}
        size="md"
        title={selectedCurrency ? `Currency · ${selectedCurrency.code}` : 'Currency'}
        description={selectedCurrency?.name}
        footer={
          selectedCurrency && canWrite ? (
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                {!confirmDeleteCurrency ? (
                  <Button
                    variant="danger"
                    size="sm"
                    leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                    onClick={() => setConfirmDeleteCurrency(true)}
                  >
                    Delete
                  </Button>
                ) : (
                  <>
                    <Button
                      variant="danger"
                      size="sm"
                      isLoading={deleteCurrencyMutation.isPending}
                      onClick={() => deleteCurrencyMutation.mutate(selectedCurrency)}
                    >
                      Confirm Delete
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => setConfirmDeleteCurrency(false)}
                    >
                      Cancel
                    </Button>
                  </>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Pencil className="h-3.5 w-3.5" />}
                onClick={() => {
                  const target = selectedCurrency;
                  setSelectedCurrency(null);
                  setEditingCurrency(target);
                  setCurrDialogOpen(true);
                }}
              >
                Edit Currency
              </Button>
            </div>
          ) : undefined
        }
      >
        {selectedCurrency && (
          <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
            <div>
              <p className="text-xs font-medium text-slate-500">ISO Code</p>
              <p className="mt-1 font-mono font-bold text-slate-900">{selectedCurrency.code}</p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Numeric Code</p>
              <p className="mt-1 font-mono text-slate-800">
                {selectedCurrency.numericCode || '—'}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Precision</p>
              <p className="mt-1 font-mono text-slate-800">
                {selectedCurrency.precision} decimals
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Status</p>
              <div className="mt-1">
                <StatusBadge
                  status={selectedCurrency.isActive ? 'ACTIVE' : 'INACTIVE'}
                  label={selectedCurrency.isActive ? 'Active' : 'Inactive'}
                  tone={selectedCurrency.isActive ? 'success' : 'neutral'}
                />
              </div>
            </div>
          </div>
        )}
      </SideDrawer>

      <SideDrawer
        open={Boolean(selectedRate)}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmDeleteRate(false);
            setSelectedRate(null);
          }
        }}
        size="md"
        title={
          selectedRate
            ? `FX Quote · ${selectedRate.fromCurrency}/${selectedRate.toCurrency}`
            : 'Exchange Rate'
        }
        description={
          selectedRate
            ? `${selectedRate.rateTypeCode} · Version v${selectedRate.rowVersion}`
            : undefined
        }
        footer={
          selectedRate && canWrite ? (
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                {!confirmDeleteRate ? (
                  <Button
                    variant="danger"
                    size="sm"
                    leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                    onClick={() => setConfirmDeleteRate(true)}
                  >
                    Delete
                  </Button>
                ) : (
                  <>
                    <Button
                      variant="danger"
                      size="sm"
                      isLoading={deleteRateMutation.isPending}
                      onClick={() => deleteRateMutation.mutate(selectedRate)}
                    >
                      Confirm Delete
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => setConfirmDeleteRate(false)}
                    >
                      Cancel
                    </Button>
                  </>
                )}
              </div>
              <Button
                variant="outline"
                size="sm"
                leftIcon={<Pencil className="h-3.5 w-3.5" />}
                onClick={() => {
                  const target = selectedRate;
                  setSelectedRate(null);
                  setEditingRate(target);
                  setRateDialogOpen(true);
                }}
              >
                Edit Quote
              </Button>
            </div>
          ) : undefined
        }
      >
        {selectedRate && (
          <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
            <div>
              <p className="text-xs font-medium text-slate-500">Pair</p>
              <p className="mt-1 font-mono font-bold text-slate-900">
                {selectedRate.fromCurrency} → {selectedRate.toCurrency}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Rate Type</p>
              <p className="mt-1 font-mono font-semibold text-brand-700">
                {selectedRate.rateTypeCode}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Spot / Applied Rate</p>
              <p className="mt-1 font-mono font-bold text-slate-900 tabular-nums">
                {formatRate(selectedRate.rate, numberLocale)}
              </p>
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Effective Window</p>
              <p className="mt-1 text-xs text-slate-800">
                {formatDate(selectedRate.validFrom, numberLocale)} →{' '}
                {selectedRate.validTo ? formatDate(selectedRate.validTo, numberLocale) : 'Open'}
              </p>
            </div>
          </div>
        )}
      </SideDrawer>

      <CurrencyFormDialog
        open={currDialogOpen}
        onOpenChange={setCurrDialogOpen}
        editingCurrency={editingCurrency}
      />

      <RateTypeFormDialog open={rateTypeDialogOpen} onOpenChange={setRateTypeDialogOpen} />

      <ExchangeRateFormDialog
        open={rateDialogOpen}
        onOpenChange={setRateDialogOpen}
        editingRate={editingRate}
        currencies={currencies}
        rateTypes={rateTypes}
      />
    </div>
  );
}
