import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowDownLeft,
  ArrowUpRight,
  Building2,
  CheckCircle2,
  CreditCard,
  GitCompareArrows,
  Pencil,
  Plus,
  ReceiptText,
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
import { Decimal } from '../../lib/decimal';
import { exportToCsv } from '../../lib/export';
import { useTranslation } from 'react-i18next';
import { formatCurrency, formatDate } from '../../lib/utils';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  deletePayment,
  fetchBankAccounts,
  fetchBankStatementLines,
  fetchPaymentDetail,
  fetchPayments,
  fetchTreasuryAccountsLookup,
  fetchTreasuryCurrenciesLookup,
  fetchTreasuryPartiesLookup,
  reconcileBankAccount,
  settlePayment,
  treasuryQueryKeys,
} from './api';
import {
  CreateBankAccountDialog,
  PaymentFormDialog,
  StatementLinesDialog,
} from './TreasuryDialogs';
import type {
  BankAccountDto,
  BankStatementLineDto,
  PaymentDto,
  ReconcileResultDto,
} from './types';

function paymentStatusTone(
  status?: string,
): 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand' {
  switch ((status ?? '').toUpperCase()) {
    case 'SETTLED':
      return 'success';
    case 'RECONCILED':
      return 'brand';
    case 'FAILED':
    case 'CANCELLED':
      return 'danger';
    default:
      return 'neutral';
  }
}

export function TreasuryWorkspace() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';
  const canWrite = useAuthStore((s) => s.hasPermission)('treasury:write');

  const [activeTab, setActiveTab] = useState('accounts');

  const [bankPage, setBankPage] = useState(1);
  const [bankSearch, setBankSearch] = useState('');
  const [bankDialogOpen, setBankDialogOpen] = useState(false);
  const [selectedBankAccount, setSelectedBankAccount] = useState<BankAccountDto | null>(null);
  const [reconcileResult, setReconcileResult] = useState<ReconcileResultDto | null>(null);

  const [paymentPage, setPaymentPage] = useState(1);
  const [paymentStatusFilter, setPaymentStatusFilter] = useState('ALL');
  const [paymentSearch, setPaymentSearch] = useState('');
  const [paymentDialogOpen, setPaymentDialogOpen] = useState(false);
  const [editingPayment, setEditingPayment] = useState<PaymentDto | null>(null);
  const [selectedPaymentId, setSelectedPaymentId] = useState<string | null>(null);
  const [confirmDeletePayment, setConfirmDeletePayment] = useState(false);

  const [statementPage, setStatementPage] = useState(1);
  const [statementBankFilter, setStatementBankFilter] = useState('ALL');
  const [statementReconciledFilter, setStatementReconciledFilter] = useState('ALL');
  const [statementSearch, setStatementSearch] = useState('');
  const [statementDialogOpen, setStatementDialogOpen] = useState(false);

  const bankAccountsQuery = useQuery({
    queryKey: treasuryQueryKeys.bankAccountsList({ page: bankPage, size: 50 }),
    queryFn: () => fetchBankAccounts({ page: bankPage, size: 50 }),
  });

  const paymentsQuery = useQuery({
    queryKey: treasuryQueryKeys.paymentsList({
      page: paymentPage,
      size: 25,
      status: paymentStatusFilter,
    }),
    queryFn: () =>
      fetchPayments({
        page: paymentPage,
        size: 25,
        status: paymentStatusFilter,
      }),
  });

  const statementLinesQuery = useQuery({
    queryKey: treasuryQueryKeys.statementLinesList({
      page: statementPage,
      size: 25,
      bankAccountId: statementBankFilter,
      isReconciled: statementReconciledFilter,
    }),
    queryFn: () =>
      fetchBankStatementLines({
        page: statementPage,
        size: 25,
        bankAccountId: statementBankFilter,
        isReconciled: statementReconciledFilter,
      }),
  });

  const paymentDetailQuery = useQuery({
    queryKey: treasuryQueryKeys.paymentDetail(selectedPaymentId ?? ''),
    queryFn: () => fetchPaymentDetail(selectedPaymentId!),
    enabled: Boolean(selectedPaymentId),
  });

  const partiesQuery = useQuery({
    queryKey: treasuryQueryKeys.parties,
    queryFn: fetchTreasuryPartiesLookup,
    retry: false,
  });

  const accountsQuery = useQuery({
    queryKey: treasuryQueryKeys.accounts,
    queryFn: fetchTreasuryAccountsLookup,
    retry: false,
  });

  const currenciesQuery = useQuery({
    queryKey: treasuryQueryKeys.currencies,
    queryFn: fetchTreasuryCurrenciesLookup,
    retry: false,
  });

  const bankAccounts = useMemo(
    () => bankAccountsQuery.data?.items ?? [],
    [bankAccountsQuery.data],
  );
  const bankPageMeta = bankAccountsQuery.data?.page ?? {
    number: bankPage,
    size: 50,
    totalElements: bankAccounts.length,
    totalPages: 1,
  };

  const payments = useMemo(() => paymentsQuery.data?.items ?? [], [paymentsQuery.data]);
  const paymentPageMeta = paymentsQuery.data?.page ?? {
    number: paymentPage,
    size: 25,
    totalElements: payments.length,
    totalPages: 1,
  };

  const statementLines = useMemo(
    () => statementLinesQuery.data?.items ?? [],
    [statementLinesQuery.data],
  );
  const statementPageMeta = statementLinesQuery.data?.page ?? {
    number: statementPage,
    size: 25,
    totalElements: statementLines.length,
    totalPages: 1,
  };

  const parties = useMemo(() => partiesQuery.data ?? [], [partiesQuery.data]);
  const coaAccounts = useMemo(() => accountsQuery.data ?? [], [accountsQuery.data]);
  const currencies = useMemo(() => currenciesQuery.data ?? [], [currenciesQuery.data]);

  const bankById = useMemo(
    () => new Map(bankAccounts.map((b) => [b.id, b])),
    [bankAccounts],
  );
  const partyById = useMemo(
    () => new Map(parties.map((p) => [p.id, p])),
    [parties],
  );
  const coaById = useMemo(
    () => new Map(coaAccounts.map((a) => [a.id, a])),
    [coaAccounts],
  );

  const filteredBanks = useMemo(() => {
    const q = bankSearch.trim().toLowerCase();
    if (!q) return bankAccounts;
    return bankAccounts.filter(
      (b) =>
        b.code.toLowerCase().includes(q) ||
        b.bankName.toLowerCase().includes(q) ||
        b.accountNo.toLowerCase().includes(q) ||
        b.currencyCode.toLowerCase().includes(q),
    );
  }, [bankAccounts, bankSearch]);

  const filteredPayments = useMemo(() => {
    const q = paymentSearch.trim().toLowerCase();
    if (!q) return payments;
    return payments.filter((p) => {
      const bank = bankById.get(p.bankAccountId);
      const party = p.partyId ? partyById.get(p.partyId) : undefined;
      return (
        p.paymentNo.toLowerCase().includes(q) ||
        p.paymentType.toLowerCase().includes(q) ||
        (p.externalRef ?? '').toLowerCase().includes(q) ||
        (bank?.code.toLowerCase().includes(q) ?? false) ||
        (party?.name.toLowerCase().includes(q) ?? false)
      );
    });
  }, [payments, paymentSearch, bankById, partyById]);

  const filteredStatements = useMemo(() => {
    const q = statementSearch.trim().toLowerCase();
    if (!q) return statementLines;
    return statementLines.filter(
      (s) =>
        s.description.toLowerCase().includes(q) ||
        (s.externalRef ?? '').toLowerCase().includes(q) ||
        (bankById.get(s.bankAccountId)?.code.toLowerCase().includes(q) ?? false),
    );
  }, [statementLines, statementSearch, bankById]);

  const kpis = useMemo(() => {
    const primaryCurrency = bankAccounts[0]?.currencyCode || 'IDR';
    const paymentSum = Decimal.sum(
      payments.map((p) => p.amount || '0'),
      2,
    );
    const settledCount = payments.filter((p) => p.status.toUpperCase() === 'SETTLED').length;
    const unreconciledCount = statementLines.filter((l) => !l.isReconciled).length;
    return {
      primaryCurrency,
      paymentVolume: formatCurrency(paymentSum.toRawString(), primaryCurrency, numberLocale),
      settledCount,
      unreconciledCount,
    };
  }, [bankAccounts, payments, statementLines, numberLocale]);

  const reconcileMutation = useMutation({
    mutationFn: (bankId: string) => reconcileBankAccount(bankId),
    onSuccess: async (result) => {
      setReconcileResult(result);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.bankAccountsAll }),
        queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.paymentsAll }),
        queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.statementLinesAll }),
      ]);
      toast.success(
        result.matchedCount > 0
          ? `Reconciliation matched ${result.matchedCount} payment(s)`
          : 'Reconciliation completed with 0 new matches',
      );
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Bank reconciliation failed';
      toast.error(msg);
    },
  });

  const settleMutation = useMutation({
    mutationFn: (payment: PaymentDto) => settlePayment(payment.id, payment.rowVersion),
    onSuccess: async (updated) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.paymentsAll }),
        queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.paymentDetail(updated.id) }),
      ]);
      toast.success(`Payment ${updated.paymentNo} settled`);
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Failed to settle payment';
      toast.error(msg);
    },
  });

  const deletePaymentMutation = useMutation({
    mutationFn: (payment: PaymentDto) => deletePayment(payment.id, payment.rowVersion),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: treasuryQueryKeys.paymentsAll });
      setConfirmDeletePayment(false);
      setSelectedPaymentId(null);
      toast.success('Draft payment deleted');
    },
    onError: (err) => {
      const msg = err instanceof ApiClientError ? err.message : 'Failed to delete draft payment';
      toast.error(msg);
    },
  });

  const activePaymentDetail = useMemo(() => {
    const fallback = payments.find((p) => p.id === selectedPaymentId) ?? null;
    if (paymentDetailQuery.data && paymentDetailQuery.data.id) {
      return paymentDetailQuery.data;
    }
    return fallback;
  }, [paymentDetailQuery.data, payments, selectedPaymentId]);

  const bankColumns: ColumnDef<BankAccountDto>[] = [
    {
      header: 'Bank Code',
      accessorKey: 'code',
      width: '150px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">{row.code}</span>
      ),
    },
    {
      header: 'Bank Name',
      accessorKey: 'bankName',
      width: '220px',
      cell: (row) => <span className="text-sm font-medium text-slate-900">{row.bankName}</span>,
    },
    {
      header: 'Account No',
      accessorKey: 'accountNo',
      width: '175px',
      cell: (row) => <span className="font-mono text-xs text-slate-700">{row.accountNo}</span>,
    },
    {
      header: 'Currency',
      width: '100px',
      cell: (row) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 font-mono text-xs font-semibold text-slate-700">
          {row.currencyCode}
        </span>
      ),
    },
    {
      header: 'GL Control Account',
      width: '240px',
      cell: (row) => {
        const acc = coaById.get(row.glAccountId);
        return acc ? (
          <span className="text-xs text-slate-700">
            <span className="font-mono font-semibold">{acc.code}</span> {acc.name}
          </span>
        ) : (
          <span className="font-mono text-xs text-slate-500">{row.glAccountId}</span>
        );
      },
    },
    {
      header: 'Status',
      width: '115px',
      cell: (row) => (
        <StatusBadge
          status="Active"
          label={`Active · v${row.rowVersion}`}
          tone="success"
        />
      ),
    },
  ];

  const paymentColumns: ColumnDef<PaymentDto>[] = [
    {
      header: 'Payment No',
      accessorKey: 'paymentNo',
      width: '165px',
      cell: (row) => (
        <span className="font-mono text-xs font-semibold text-slate-900">{row.paymentNo}</span>
      ),
    },
    {
      header: 'Direction',
      width: '130px',
      cell: (row) => {
        const incoming = row.paymentType === 'Incoming';
        return (
          <span
            className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
              incoming ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-800'
            }`}
          >
            {incoming ? (
              <ArrowDownLeft className="h-3 w-3" />
            ) : (
              <ArrowUpRight className="h-3 w-3" />
            )}
            {row.paymentType}
          </span>
        );
      },
    },
    {
      header: 'Bank Account',
      width: '185px',
      cell: (row) => {
        const bank = bankById.get(row.bankAccountId);
        return bank ? (
          <div>
            <p className="text-xs font-semibold text-slate-900">{bank.code}</p>
            <p className="text-[11px] text-slate-500">{bank.bankName}</p>
          </div>
        ) : (
          <span className="font-mono text-xs text-slate-500">
            {row.bankAccountId.slice(0, 12)}…
          </span>
        );
      },
    },
    {
      header: 'Counterparty',
      width: '190px',
      cell: (row) => {
        if (!row.partyId) return <span className="text-xs text-slate-400">—</span>;
        const party = partyById.get(row.partyId);
        return party ? (
          <span className="text-xs font-medium text-slate-800">{party.name}</span>
        ) : (
          <span className="font-mono text-xs text-slate-500">{row.partyId.slice(0, 12)}…</span>
        );
      },
    },
    {
      header: 'Date',
      width: '120px',
      cell: (row) => (
        <span className="text-xs text-slate-600">{formatDate(row.paymentDate, numberLocale)}</span>
      ),
    },
    {
      header: 'Amount',
      align: 'right',
      width: '160px',
      cell: (row) => {
        const currency = bankById.get(row.bankAccountId)?.currencyCode || 'IDR';
        return (
          <span className="font-mono text-xs font-semibold text-slate-900 tabular-nums">
            {formatCurrency(row.amount, currency, numberLocale)}
          </span>
        );
      },
    },
    {
      header: 'Status',
      width: '125px',
      cell: (row) => (
        <StatusBadge
          status={row.status}
          label={row.status}
          tone={paymentStatusTone(row.status)}
        />
      ),
    },
  ];

  const statementColumns: ColumnDef<BankStatementLineDto>[] = [
    {
      header: 'Statement Date',
      width: '130px',
      cell: (row) => (
        <span className="text-xs text-slate-700">
          {formatDate(row.statementDate, numberLocale)}
        </span>
      ),
    },
    {
      header: 'Bank Account',
      width: '170px',
      cell: (row) => {
        const bank = bankById.get(row.bankAccountId);
        return (
          <span className="font-mono text-xs font-medium text-slate-800">
            {bank ? bank.code : `${row.bankAccountId.slice(0, 12)}…`}
          </span>
        );
      },
    },
    {
      header: 'Description',
      accessorKey: 'description',
      width: '270px',
      cell: (row) => <span className="text-xs text-slate-800">{row.description}</span>,
    },
    {
      header: 'External Ref',
      width: '150px',
      cell: (row) => (
        <span className="font-mono text-xs text-slate-500">{row.externalRef || '—'}</span>
      ),
    },
    {
      header: 'Amount (+Cr / -Dr)',
      align: 'right',
      width: '175px',
      cell: (row) => {
        const currency = bankById.get(row.bankAccountId)?.currencyCode || 'IDR';
        const isNegative = row.amount.trim().startsWith('-');
        return (
          <span
            className={`font-mono text-xs font-semibold tabular-nums ${
              isNegative ? 'text-rose-700' : 'text-emerald-700'
            }`}
          >
            {formatCurrency(row.amount, currency, numberLocale)}
          </span>
        );
      },
    },
    {
      header: 'Reconciliation',
      width: '140px',
      cell: (row) => (
        <StatusBadge
          status={row.isReconciled ? 'RECONCILED' : 'PENDING'}
          label={row.isReconciled ? 'Reconciled' : 'Unreconciled'}
          tone={row.isReconciled ? 'success' : 'warning'}
        />
      ),
    },
  ];

  const handleExportStatements = () => {
    exportToCsv(
      filteredStatements,
      [
        { header: 'Statement Date', accessor: (r) => r.statementDate },
        {
          header: 'Bank Code',
          accessor: (r) => bankById.get(r.bankAccountId)?.code ?? r.bankAccountId,
        },
        { header: 'Description', accessor: (r) => r.description },
        { header: 'External Ref', accessor: (r) => r.externalRef ?? '' },
        { header: 'Amount', accessor: (r) => r.amount },
        { header: 'Reconciled', accessor: (r) => (r.isReconciled ? 'Yes' : 'No') },
        { header: 'Matched Payment ID', accessor: (r) => r.matchedPaymentId ?? '' },
      ],
      'bank-statement-lines.csv',
    );
    toast.success('Bank statement lines exported to CSV');
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              {t('nav.treasury')} · Cash & Bank
            </h1>
            <span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  bankAccountsQuery.isError || paymentsQuery.isError || statementLinesQuery.isError
                    ? 'bg-rose-500'
                    : bankAccountsQuery.isFetching || paymentsQuery.isFetching
                      ? 'bg-amber-500'
                      : 'bg-emerald-500'
                }`}
              />
              SVC-006 · TREASURY
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            Corporate bank accounts, cash disbursements & receipts, statement import, and auto-reconciliation.
          </p>
        </div>

        {canWrite && (
          <div className="flex flex-wrap items-center gap-2">
            <Button
              variant="outline"
              leftIcon={<Building2 className="h-4 w-4" />}
              onClick={() => setBankDialogOpen(true)}
            >
              Add Bank Account
            </Button>
            <Button
              variant="outline"
              leftIcon={<ReceiptText className="h-4 w-4" />}
              onClick={() => setStatementDialogOpen(true)}
            >
              Add Statement Lines
            </Button>
            <Button
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                setEditingPayment(null);
                setPaymentDialogOpen(true);
              }}
            >
              New Payment
            </Button>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Bank Accounts"
          value={bankPageMeta.totalElements}
          subtitle="Active cash/bank ledgers"
          accent="brand"
          icon={<Building2 className="h-4 w-4" />}
          isLoading={bankAccountsQuery.isLoading}
        />
        <KpiCard
          label="Page Payment Volume"
          value={kpis.paymentVolume}
          subtitle={`${kpis.settledCount} settled on page`}
          accent="sky"
          icon={<CreditCard className="h-4 w-4" />}
          isLoading={paymentsQuery.isLoading}
        />
        <KpiCard
          label="Statement Lines"
          value={statementPageMeta.totalElements}
          subtitle="Bank feed entries"
          accent="emerald"
          icon={<CheckCircle2 className="h-4 w-4" />}
          isLoading={statementLinesQuery.isLoading}
        />
        <KpiCard
          label="Unreconciled Lines"
          value={kpis.unreconciledCount}
          subtitle="On current page"
          accent="amber"
          icon={<GitCompareArrows className="h-4 w-4" />}
          isLoading={statementLinesQuery.isLoading}
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList variant="underline">
          <TabsTrigger value="accounts" variant="underline">
            Bank Accounts ({bankPageMeta.totalElements})
          </TabsTrigger>
          <TabsTrigger value="payments" variant="underline">
            Payments ({paymentPageMeta.totalElements})
          </TabsTrigger>
          <TabsTrigger value="statements" variant="underline">
            Bank Statement Lines ({statementPageMeta.totalElements})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="accounts">
          <VirtualizedTable
            data={filteredBanks}
            columns={bankColumns}
            isLoading={bankAccountsQuery.isLoading}
            error={bankAccountsQuery.error}
            onRetry={() => void bankAccountsQuery.refetch()}
            emptyTitle="No bank accounts registered"
            emptyDescription="Register a corporate bank account linked to a Cash/Bank GL account."
            onResetFilters={bankSearch ? () => setBankSearch('') : undefined}
            onRowClick={(row) => {
              setReconcileResult(null);
              setSelectedBankAccount(row);
            }}
            pagination={{
              page: bankPageMeta.number || bankPage,
              size: bankPageMeta.size || 50,
              totalElements: bankPageMeta.totalElements,
              totalPages: bankPageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setBankPage,
            }}
            toolbar={{
              searchValue: bankSearch,
              onSearchChange: setBankSearch,
              searchPlaceholder: 'Search bank code, bank name, account no...',
            }}
          />
        </TabsContent>

        <TabsContent value="payments">
          <VirtualizedTable
            data={filteredPayments}
            columns={paymentColumns}
            isLoading={paymentsQuery.isLoading}
            error={paymentsQuery.error}
            onRetry={() => void paymentsQuery.refetch()}
            emptyTitle="No payments recorded"
            emptyDescription="Create an incoming receipt or outgoing vendor disbursement."
            onResetFilters={
              paymentSearch || paymentStatusFilter !== 'ALL'
                ? () => {
                    setPaymentSearch('');
                    setPaymentStatusFilter('ALL');
                    setPaymentPage(1);
                  }
                : undefined
            }
            onRowClick={(row) => {
              setConfirmDeletePayment(false);
              setSelectedPaymentId(row.id);
            }}
            pagination={{
              page: paymentPageMeta.number || paymentPage,
              size: paymentPageMeta.size || 25,
              totalElements: paymentPageMeta.totalElements,
              totalPages: paymentPageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setPaymentPage,
            }}
            toolbar={{
              searchValue: paymentSearch,
              onSearchChange: setPaymentSearch,
              searchPlaceholder: 'Search payment number, bank, counterparty...',
              filters: (
                <Select
                  aria-label="Filter payment status"
                  className="w-40"
                  value={paymentStatusFilter}
                  onValueChange={(value) => {
                    setPaymentStatusFilter(value);
                    setPaymentPage(1);
                  }}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Settled', label: 'Settled' },
                    { value: 'Reconciled', label: 'Reconciled' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>

        <TabsContent value="statements">
          <VirtualizedTable
            data={filteredStatements}
            columns={statementColumns}
            isLoading={statementLinesQuery.isLoading}
            error={statementLinesQuery.error}
            onRetry={() => void statementLinesQuery.refetch()}
            emptyTitle="No bank statement lines imported"
            emptyDescription="Import single or batch statement lines to run automated reconciliation."
            onResetFilters={
              statementSearch ||
              statementBankFilter !== 'ALL' ||
              statementReconciledFilter !== 'ALL'
                ? () => {
                    setStatementSearch('');
                    setStatementBankFilter('ALL');
                    setStatementReconciledFilter('ALL');
                    setStatementPage(1);
                  }
                : undefined
            }
            pagination={{
              page: statementPageMeta.number || statementPage,
              size: statementPageMeta.size || 25,
              totalElements: statementPageMeta.totalElements,
              totalPages: statementPageMeta.totalPages || 1,
              pageBase: 1,
              onPageChange: setStatementPage,
            }}
            toolbar={{
              searchValue: statementSearch,
              onSearchChange: setStatementSearch,
              searchPlaceholder: 'Search narration, external ref...',
              onExportCsv: filteredStatements.length > 0 ? handleExportStatements : undefined,
              filters: (
                <div className="flex flex-wrap items-center gap-2">
                  <Select
                    aria-label="Filter statement bank account"
                    className="w-44"
                    value={statementBankFilter}
                    onValueChange={(value) => {
                      setStatementBankFilter(value);
                      setStatementPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Bank Accounts' },
                      ...bankAccounts.map((b) => ({
                        value: b.id,
                        label: `${b.code} (${b.currencyCode})`,
                      })),
                    ]}
                  />
                  <Select
                    aria-label="Filter statement reconciled status"
                    className="w-40"
                    value={statementReconciledFilter}
                    onValueChange={(value) => {
                      setStatementReconciledFilter(value);
                      setStatementPage(1);
                    }}
                    options={[
                      { value: 'ALL', label: 'All Lines' },
                      { value: 'false', label: 'Unreconciled' },
                      { value: 'true', label: 'Reconciled' },
                    ]}
                  />
                </div>
              ),
            }}
          />
        </TabsContent>
      </Tabs>

      <SideDrawer
        open={Boolean(selectedBankAccount)}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedBankAccount(null);
            setReconcileResult(null);
          }
        }}
        size="lg"
        title={
          selectedBankAccount
            ? `Bank Account · ${selectedBankAccount.code}`
            : 'Bank Account'
        }
        description={
          selectedBankAccount
            ? `${selectedBankAccount.bankName} · ${selectedBankAccount.accountNo}`
            : undefined
        }
        footer={
          selectedBankAccount && canWrite ? (
            <div className="flex justify-end">
              <Button
                variant="primary"
                leftIcon={<GitCompareArrows className="h-4 w-4" />}
                isLoading={reconcileMutation.isPending}
                onClick={() => reconcileMutation.mutate(selectedBankAccount.id)}
              >
                Run Auto-Reconcile
              </Button>
            </div>
          ) : undefined
        }
      >
        {selectedBankAccount ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
              <div>
                <p className="text-xs font-medium text-slate-500">Account Code</p>
                <p className="mt-1 font-mono font-semibold text-slate-900">
                  {selectedBankAccount.code}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Currency</p>
                <p className="mt-1 font-mono font-semibold text-slate-900">
                  {selectedBankAccount.currencyCode}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Bank & Number</p>
                <p className="mt-1 text-slate-800">{selectedBankAccount.bankName}</p>
                <p className="font-mono text-xs text-slate-500">
                  {selectedBankAccount.accountNo}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">GL Account</p>
                <p className="mt-1 text-slate-800">
                  {coaById.get(selectedBankAccount.glAccountId)?.name ??
                    selectedBankAccount.glAccountId}
                </p>
              </div>
            </div>

            {reconcileResult && (
              <div className="rounded-xl border border-emerald-200 bg-emerald-50/70 p-4">
                <p className="text-sm font-semibold text-emerald-900">
                  Reconciliation Summary: {reconcileResult.matchedCount} match(es) found
                </p>
                {reconcileResult.matches.length > 0 ? (
                  <ul className="mt-2 space-y-1.5 text-xs text-emerald-900">
                    {reconcileResult.matches.map((match) => (
                      <li
                        key={`${match.statementLineId}-${match.paymentId}`}
                        className="flex items-center justify-between rounded-md bg-white px-3 py-1.5 ring-1 ring-emerald-200"
                      >
                        <span className="font-mono font-semibold">{match.paymentNo}</span>
                        <span className="font-mono tabular-nums">
                          {formatCurrency(
                            match.amount,
                            selectedBankAccount.currencyCode,
                            numberLocale,
                          )}
                        </span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="mt-1 text-xs text-emerald-800">
                    No matching settled payments and unreconciled statement lines found.
                  </p>
                )}
              </div>
            )}
          </div>
        ) : null}
      </SideDrawer>

      <SideDrawer
        open={Boolean(selectedPaymentId)}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmDeletePayment(false);
            setSelectedPaymentId(null);
          }
        }}
        size="lg"
        title={
          activePaymentDetail
            ? `Payment · ${activePaymentDetail.paymentNo}`
            : 'Payment Detail'
        }
        description={
          activePaymentDetail
            ? `${activePaymentDetail.paymentType} · Version v${activePaymentDetail.rowVersion}`
            : undefined
        }
        footer={
          activePaymentDetail && canWrite ? (
            <div className="flex flex-wrap items-center justify-between gap-2">
              {activePaymentDetail.status.toUpperCase() === 'DRAFT' ? (
                <>
                  <div className="flex items-center gap-2">
                    {!confirmDeletePayment ? (
                      <Button
                        variant="danger"
                        size="sm"
                        leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                        onClick={() => setConfirmDeletePayment(true)}
                      >
                        Delete
                      </Button>
                    ) : (
                      <>
                        <Button
                          variant="danger"
                          size="sm"
                          isLoading={deletePaymentMutation.isPending}
                          onClick={() => deletePaymentMutation.mutate(activePaymentDetail)}
                        >
                          Confirm Delete
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => setConfirmDeletePayment(false)}
                        >
                          Cancel
                        </Button>
                      </>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      leftIcon={<Pencil className="h-3.5 w-3.5" />}
                      onClick={() => {
                        setSelectedPaymentId(null);
                        setEditingPayment(activePaymentDetail);
                        setPaymentDialogOpen(true);
                      }}
                    >
                      Edit Draft
                    </Button>
                  </div>
                  <Button
                    variant="success"
                    size="sm"
                    leftIcon={<CheckCircle2 className="h-4 w-4" />}
                    isLoading={settleMutation.isPending}
                    onClick={() => settleMutation.mutate(activePaymentDetail)}
                  >
                    Settle Payment
                  </Button>
                </>
              ) : (
                <p className="text-xs text-slate-500">
                  Settled payments are locked for bank reconciliation integrity.
                </p>
              )}
            </div>
          ) : undefined
        }
      >
        {activePaymentDetail ? (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm">
              <div>
                <p className="text-xs font-medium text-slate-500">Status</p>
                <div className="mt-1">
                  <StatusBadge
                    status={activePaymentDetail.status}
                    label={activePaymentDetail.status}
                    tone={paymentStatusTone(activePaymentDetail.status)}
                  />
                </div>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Amount</p>
                <p className="mt-1 font-mono font-semibold text-slate-900 tabular-nums">
                  {formatCurrency(
                    activePaymentDetail.amount,
                    bankById.get(activePaymentDetail.bankAccountId)?.currencyCode || 'IDR',
                    numberLocale,
                  )}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Bank Account</p>
                <p className="mt-1 text-slate-800">
                  {bankById.get(activePaymentDetail.bankAccountId)?.code ??
                    activePaymentDetail.bankAccountId}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Payment Date</p>
                <p className="mt-1 text-slate-800">
                  {formatDate(activePaymentDetail.paymentDate, numberLocale)}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">Counterparty</p>
                <p className="mt-1 text-slate-800">
                  {activePaymentDetail.partyId
                    ? partyById.get(activePaymentDetail.partyId)?.name ??
                      activePaymentDetail.partyId
                    : '—'}
                </p>
              </div>
              <div>
                <p className="text-xs font-medium text-slate-500">External Ref</p>
                <p className="mt-1 font-mono text-xs text-slate-700">
                  {activePaymentDetail.externalRef || '—'}
                </p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-sm font-semibold text-slate-900">Document Allocations</h3>
              {paymentDetailQuery.isLoading ? (
                <div className="rounded-lg border border-slate-200 p-4 text-xs text-slate-500">
                  Loading allocations...
                </div>
              ) : activePaymentDetail.allocations &&
                activePaymentDetail.allocations.length > 0 ? (
                <div className="overflow-hidden rounded-xl border border-slate-200">
                  <table className="w-full text-left text-xs">
                    <thead className="bg-slate-50 text-slate-600">
                      <tr>
                        <th className="px-3 py-2 font-semibold">Document Reference</th>
                        <th className="px-3 py-2 text-right font-semibold">Allocated Amount</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {activePaymentDetail.allocations.map((alloc, idx) => (
                        <tr key={alloc.id ?? idx}>
                          <td className="px-3 py-2 font-mono font-semibold text-slate-800">
                            {alloc.targetDocumentRef}
                          </td>
                          <td className="px-3 py-2 text-right font-mono text-slate-900 tabular-nums">
                            {formatCurrency(
                              alloc.allocatedAmount,
                              bankById.get(activePaymentDetail.bankAccountId)?.currencyCode ||
                                'IDR',
                              numberLocale,
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="rounded-lg border border-slate-200 p-4 text-xs text-slate-500">
                  No invoice/bill allocations linked to this payment.
                </div>
              )}
            </div>
          </div>
        ) : null}
      </SideDrawer>

      <CreateBankAccountDialog
        open={bankDialogOpen}
        onOpenChange={setBankDialogOpen}
        accounts={coaAccounts}
        currencies={currencies}
      />

      <PaymentFormDialog
        open={paymentDialogOpen}
        onOpenChange={setPaymentDialogOpen}
        editingPayment={editingPayment}
        bankAccounts={bankAccounts}
        parties={parties}
      />

      <StatementLinesDialog
        open={statementDialogOpen}
        onOpenChange={setStatementDialogOpen}
        bankAccounts={bankAccounts}
      />
    </div>
  );
}
