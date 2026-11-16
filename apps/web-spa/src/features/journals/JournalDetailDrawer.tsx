import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CheckCircle2, Edit3, Printer, RotateCcw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '../../components/ui/Button';
import { ErrorState } from '../../components/ui/ErrorState';
import { Input } from '../../components/ui/Input';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { CardSkeleton } from '../../components/ui/Skeleton';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { ApiClientError, apiClient } from '../../lib/api';
import { addDecimals, Decimal, formatCurrency, formatDate, formatRate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  normalizeJournalStatus,
  type AccountLookupDto,
  type FiscalPeriodLookupDto,
  type JournalEntryDto,
  type LedgerLookupDto,
} from './journals-types';

export interface JournalDetailDrawerProps {
  entryId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onEdit: (entry: JournalEntryDto) => void;
  accountsById: Map<string, AccountLookupDto>;
  ledgersById: Map<string, LedgerLookupDto>;
  periodsById: Map<string, FiscalPeriodLookupDto>;
}

export function JournalDetailDrawer({
  entryId,
  open,
  onOpenChange,
  onEdit,
  accountsById,
  ledgersById,
  periodsById,
}: JournalDetailDrawerProps) {
  const locale = useUiStore((s) => s.locale);
  const isId = locale === 'id';
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();

  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [reversePromptOpen, setReversePromptOpen] = useState(false);
  const [reverseReason, setReverseReason] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);

  const canUpdate = hasUserPermission(user, ['gl:update', 'gl.update', 'gl.*']);
  const canApprove = hasUserPermission(user, [
    'gl:approve',
    'gl.approve',
    'gl:post',
    'gl.post',
    'gl.*',
  ]);

  const detailQuery = useQuery({
    queryKey: ['journals', 'detail', entryId],
    enabled: Boolean(open && entryId),
    queryFn: async () => {
      return apiClient<JournalEntryDto>(`/journal-entries/${entryId}`);
    },
  });

  const entry = detailQuery.data ?? null;
  const normalizedStatus = entry ? normalizeJournalStatus(entry.status) : '';

  const invalidateJournalCaches = () => {
    void queryClient.invalidateQueries({ queryKey: ['journals'] });
    void queryClient.invalidateQueries({ queryKey: ['balances'] });
    void queryClient.invalidateQueries({ queryKey: ['gl-balances'] });
    void queryClient.invalidateQueries({ queryKey: ['dashboard-metrics'] });
  };

  const postMutation = useMutation({
    mutationFn: async (target: JournalEntryDto) => {
      return apiClient<JournalEntryDto>(`/journal-entries/${target.id}/post`, {
        method: 'POST',
        ifMatch: String(target.rowVersion),
        idempotencyKey: crypto.randomUUID(),
      });
    },
    onSuccess: (updated) => {
      setActionError(null);
      invalidateJournalCaches();
      void detailQuery.refetch();
      toast.success(
        isId
          ? `Jurnal ${updated.journalNo} berhasil diposting ke Buku Besar.`
          : `Journal ${updated.journalNo} posted to General Ledger.`
      );
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : isId
              ? 'Gagal memposting jurnal.'
              : 'Failed to post journal.';
      setActionError(msg);
      toast.error(msg);
    },
  });

  const reverseMutation = useMutation({
    mutationFn: async ({ target, reason }: { target: JournalEntryDto; reason: string }) => {
      return apiClient<JournalEntryDto>(`/journal-entries/${target.id}/reverse`, {
        method: 'POST',
        ifMatch: String(target.rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: reason ? JSON.stringify({ reason }) : undefined,
      });
    },
    onSuccess: (reversed) => {
      setActionError(null);
      setReversePromptOpen(false);
      setReverseReason('');
      invalidateJournalCaches();
      void detailQuery.refetch();
      toast.success(
        isId
          ? `Jurnal pembalik ${reversed.journalNo} berhasil dibuat.`
          : `Reversal journal ${reversed.journalNo} created.`
      );
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : isId
              ? 'Gagal membalik jurnal.'
              : 'Failed to reverse journal.';
      setActionError(msg);
      toast.error(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (target: JournalEntryDto) => {
      await apiClient<unknown>(`/journal-entries/${target.id}`, {
        method: 'DELETE',
        ifMatch: String(target.rowVersion),
      });
    },
    onSuccess: () => {
      setActionError(null);
      setConfirmDeleteOpen(false);
      invalidateJournalCaches();
      toast.success(isId ? 'Draf jurnal dihapus.' : 'Journal draft deleted.');
      onOpenChange(false);
    },
    onError: (err) => {
      const msg =
        err instanceof ApiClientError
          ? err.message
          : err instanceof Error
            ? err.message
            : isId
              ? 'Gagal menghapus draf jurnal.'
              : 'Failed to delete draft.';
      setActionError(msg);
      toast.error(msg);
    },
  });

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setConfirmDeleteOpen(false);
      setReversePromptOpen(false);
      setReverseReason('');
      setActionError(null);
    }
    onOpenChange(nextOpen);
  };

  const ledgerInfo = entry ? ledgersById.get(entry.ledgerId) : undefined;
  const periodInfo = entry ? periodsById.get(entry.fiscalPeriodId) : undefined;

  const isAnyMutating =
    postMutation.isPending || reverseMutation.isPending || deleteMutation.isPending;

  const isBalanced = entry
    ? new Decimal(entry.totalDebit || '0', 2).equals(new Decimal(entry.totalCredit || '0', 2))
    : false;

  const functionalTotal = entry
    ? (entry.lines ?? []).reduce(
        (acc, line) => {
          const lineDebit = new Decimal(line.debitAmount || '0', 4);
          return lineDebit.gt('0') ? addDecimals(acc, line.functionalAmount || '0', 2) : acc;
        },
        '0.00'
      )
    : '0.00';

  return (
    <SideDrawer
      open={open}
      onOpenChange={handleOpenChange}
      size="xl"
      title={
        entry
          ? `${entry.journalNo}`
          : isId
            ? 'Detail Entri Jurnal'
            : 'Journal Entry Detail'
      }
      titleBadge={entry ? <StatusBadge status={normalizedStatus || entry.status} /> : undefined}
      description={
        entry
          ? `${ledgerInfo?.code ?? 'GL'} · Multi-Currency ADR-014 · ${isId ? 'Tgl Posting' : 'Posting'} ${formatDate(entry.postingDate, locale)} · v${entry.rowVersion}`
          : undefined
      }
      footer={
        entry ? (
          <div className="flex w-full flex-wrap items-center justify-between gap-2">
            <Button variant="ghost" size="sm" onClick={() => handleOpenChange(false)}>
              {isId ? 'Tutup' : 'Close'}
            </Button>

            <div className="flex flex-wrap items-center gap-2">
              {normalizedStatus === 'DRAFT' && canUpdate ? (
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={isAnyMutating}
                    leftIcon={<Edit3 className="h-3.5 w-3.5" />}
                    onClick={() => {
                      handleOpenChange(false);
                      onEdit(entry);
                    }}
                  >
                    {isId ? 'Ubah Draf' : 'Edit Draft'}
                  </Button>
                  <Button
                    variant="danger-soft"
                    size="sm"
                    disabled={isAnyMutating}
                    leftIcon={<Trash2 className="h-3.5 w-3.5" />}
                    onClick={() => setConfirmDeleteOpen((prev) => !prev)}
                  >
                    {isId ? 'Hapus Draf' : 'Delete Draft'}
                  </Button>
                </>
              ) : null}

              {normalizedStatus === 'DRAFT' && canApprove ? (
                <Button
                  variant="success"
                  size="sm"
                  isLoading={postMutation.isPending}
                  disabled={isAnyMutating}
                  leftIcon={<CheckCircle2 className="h-3.5 w-3.5" />}
                  onClick={() => postMutation.mutate(entry)}
                >
                  {isId ? 'Posting ke GL' : 'Post to GL'}
                </Button>
              ) : null}

              {normalizedStatus === 'POSTED' ? (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={isAnyMutating}
                  leftIcon={<Printer className="h-3.5 w-3.5" />}
                  onClick={() => window.print()}
                >
                  {isId ? 'Cetak Bukti' : 'Print Voucher'}
                </Button>
              ) : null}

              {normalizedStatus === 'POSTED' && canApprove ? (
                <Button
                  variant="danger-soft"
                  size="sm"
                  disabled={isAnyMutating}
                  leftIcon={<RotateCcw className="h-3.5 w-3.5" />}
                  onClick={() => setReversePromptOpen((prev) => !prev)}
                >
                  {isId ? 'Balik Jurnal' : 'Reverse Entry'}
                </Button>
              ) : null}
            </div>
          </div>
        ) : null
      }
    >
      {detailQuery.isLoading ? (
        <CardSkeleton />
      ) : detailQuery.error ? (
        <ErrorState
          error={detailQuery.error}
          title={isId ? 'Gagal Memuat Detail Jurnal' : 'Failed to Load Journal Detail'}
          onRetry={() => void detailQuery.refetch()}
        />
      ) : !entry ? null : (
        <div className="space-y-5">
          {actionError ? (
            <div
              role="alert"
              className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700 dark:border-rose-800/70 dark:bg-rose-950/50 dark:text-rose-300"
            >
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              <span>{actionError}</span>
            </div>
          ) : null}

          {confirmDeleteOpen ? (
            <div className="rounded-xl border border-rose-200 bg-rose-50/90 p-3 text-xs dark:border-rose-800/70 dark:bg-rose-950/50">
              <p className="font-semibold text-rose-800 dark:text-rose-200">
                {isId
                  ? `Hapus permanen draf jurnal ${entry.journalNo}?`
                  : `Permanently delete draft journal ${entry.journalNo}?`}
              </p>
              <div className="mt-2.5 flex items-center gap-2">
                <Button
                  variant="danger"
                  size="xs"
                  isLoading={deleteMutation.isPending}
                  onClick={() => deleteMutation.mutate(entry)}
                >
                  {isId ? 'Ya, Hapus Draf' : 'Yes, Delete Draft'}
                </Button>
                <Button
                  variant="outline"
                  size="xs"
                  onClick={() => setConfirmDeleteOpen(false)}
                >
                  {isId ? 'Batal' : 'Cancel'}
                </Button>
              </div>
            </div>
          ) : null}

          {reversePromptOpen ? (
            <div className="rounded-xl border border-amber-200 bg-amber-50/80 p-3 text-xs dark:border-amber-800/70 dark:bg-amber-950/40">
              <p className="font-semibold text-amber-900 dark:text-amber-200">
                {isId
                  ? `Konfirmasi Pembalikan Jurnal ${entry.journalNo}`
                  : `Confirm Reversal of ${entry.journalNo}`}
              </p>
              <p className="mt-0.5 text-amber-800/80 dark:text-amber-300/80">
                {isId
                  ? 'Sistem akan membuat jurnal pembalik bernilai berlawanan dengan If-Match optimistik.'
                  : 'Creates an offsetting reversal journal entry guarded by optimistic concurrency.'}
              </p>
              <div className="mt-2.5 space-y-2">
                <Input
                  value={reverseReason}
                  onChange={(e) => setReverseReason(e.target.value)}
                  placeholder={
                    isId ? 'Alasan pembalikan (opsional)...' : 'Reversal reason (optional)...'
                  }
                  aria-label={isId ? 'Alasan pembalikan jurnal' : 'Journal reversal reason'}
                />
                <div className="flex items-center gap-2">
                  <Button
                    variant="danger"
                    size="xs"
                    isLoading={reverseMutation.isPending}
                    onClick={() =>
                      reverseMutation.mutate({ target: entry, reason: reverseReason.trim() })
                    }
                  >
                    {isId ? 'Konfirmasi Balik Jurnal' : 'Confirm Reverse'}
                  </Button>
                  <Button
                    variant="outline"
                    size="xs"
                    onClick={() => setReversePromptOpen(false)}
                  >
                    {isId ? 'Batal' : 'Cancel'}
                  </Button>
                </div>
              </div>
            </div>
          ) : null}

          {/* Header Metadata Grid */}
          <dl className="grid grid-cols-2 gap-3.5 rounded-xl border border-border bg-surface-alt/80 p-4 text-xs sm:grid-cols-3 dark:bg-slate-800/40">
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Periode Fiskal' : 'Fiscal Period'}
              </dt>
              <dd className="mt-1 font-mono font-semibold text-slate-900 dark:text-slate-100">
                {periodInfo ? `#${periodInfo.periodNo} · ${periodInfo.status}` : entry.fiscalPeriodId}
              </dd>
            </div>
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Tgl Posting' : 'Posting Date'}
              </dt>
              <dd className="mt-1 font-semibold text-slate-900 dark:text-slate-100">
                {formatDate(entry.postingDate, locale)}
              </dd>
            </div>
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Sumber Jurnal' : 'Source'}
              </dt>
              <dd className="mt-1 font-semibold text-slate-900 dark:text-slate-100">
                {entry.journalSource}
              </dd>
            </div>
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Buku Besar' : 'Ledger'}
              </dt>
              <dd className="mt-1 font-mono font-semibold text-slate-900 dark:text-slate-100">
                {ledgerInfo ? `${ledgerInfo.code} (${ledgerInfo.ledgerType})` : entry.ledgerId}
              </dd>
            </div>
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Mata Uang & Kurs' : 'Currency & Rate'}
              </dt>
              <dd className="mt-1 font-money font-semibold text-slate-900 dark:text-slate-100">
                {entry.currencyCode} @ {formatRate(entry.exchangeRate, locale)}
              </dd>
            </div>
            <div>
              <dt className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                {isId ? 'Versi & Status' : 'Version & Status'}
              </dt>
              <dd className="mt-1 flex items-center gap-2 font-mono font-semibold text-slate-800 dark:text-slate-200">
                <span>v{entry.rowVersion}</span>
                <span>·</span>
                <span>{normalizedStatus || entry.status}</span>
              </dd>
            </div>
          </dl>

          {/* Memo & Source Reference Box */}
          <div className="rounded-xl border border-border bg-surface p-3.5 text-xs">
            <p className="text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              {isId ? 'Memo & Referensi Dokumen Sumber' : 'Memo & Source Document Ref'}
            </p>
            <p className="mt-1 text-slate-800 dark:text-slate-200 font-medium">
              {entry.sourceDocumentRef || (isId ? 'Tidak ada memo dokumen sumber.' : 'No source document memo.')}
            </p>
          </div>

          {/* Journal Lines Table */}
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <h3 className="text-xs font-bold text-slate-900 dark:text-slate-100">
                {isId ? 'Baris Akun & Nilai Fungsional' : 'Account Lines & Functional Values'}
              </h3>
              <span className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
                {(entry.lines ?? []).length} {isId ? 'baris' : 'lines'} ·{' '}
                {normalizedStatus === 'POSTED'
                  ? isId
                    ? 'Terposting ke GL'
                    : 'Posted to GL'
                  : normalizedStatus || entry.status}
              </span>
            </div>

            <div className="overflow-x-auto rounded-xl border border-border">
              <table className="w-full border-collapse text-left text-xs">
                <thead>
                  <tr className="border-b border-border bg-surface-muted/70 text-[10.5px] font-bold uppercase tracking-wider text-slate-500 dark:bg-slate-800/70 dark:text-slate-400">
                    <th scope="col" className="px-3 py-2.5">
                      #
                    </th>
                    <th scope="col" className="px-3 py-2.5">
                      {isId ? 'Akun GL' : 'GL Account'}
                    </th>
                    <th scope="col" className="px-3 py-2.5">
                      {isId ? 'Keterangan' : 'Description'}
                    </th>
                    <th scope="col" className="px-3 py-2.5 text-right">
                      Debit
                    </th>
                    <th scope="col" className="px-3 py-2.5 text-right">
                      {isId ? 'Kredit' : 'Credit'}
                    </th>
                    <th scope="col" className="px-3 py-2.5 text-right">
                      {isId ? 'Fungsional' : 'Functional'}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {(entry.lines ?? []).map((line) => {
                    const account = accountsById.get(line.accountId);
                    return (
                      <tr
                        key={line.id ?? `${line.lineNo}-${line.accountId}`}
                        className="border-b border-border-subtle last:border-b-0 dark:border-slate-800/70"
                      >
                        <td className="px-3 py-2.5 font-mono text-slate-500">{line.lineNo}</td>
                        <td className="px-3 py-2.5">
                          <div className="font-mono font-bold text-brand-700 dark:text-brand-400">
                            {account ? account.code : line.accountId.slice(0, 8)}
                          </div>
                          {account ? (
                            <div className="text-[11px] text-slate-600 dark:text-slate-300">
                              {account.name}
                            </div>
                          ) : null}
                        </td>
                        <td className="px-3 py-2.5 text-slate-600 dark:text-slate-300">
                          {line.description || '—'}
                        </td>
                        <td className="px-3 py-2.5 text-right font-money text-slate-800 dark:text-slate-200">
                          {formatCurrency(line.debitAmount, entry.currencyCode, locale)}
                        </td>
                        <td className="px-3 py-2.5 text-right font-money text-slate-800 dark:text-slate-200">
                          {formatCurrency(line.creditAmount, entry.currencyCode, locale)}
                        </td>
                        <td className="px-3 py-2.5 text-right font-money text-slate-600 dark:text-slate-400">
                          {formatCurrency(line.functionalAmount, 'IDR', locale)}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
                <tfoot>
                  <tr className="border-t border-border bg-brand-50/50 font-semibold dark:bg-brand-950/30">
                    <td colSpan={3} className="px-3 py-2.5 text-xs font-bold text-slate-700 dark:text-slate-200">
                      {isId ? 'TOTAL TRANSAKSI & REKONSILIASI' : 'TOTAL & RECONCILIATION'}
                    </td>
                    <td className="px-3 py-2.5 text-right font-money text-slate-900 dark:text-slate-100">
                      {formatCurrency(entry.totalDebit, entry.currencyCode, locale)}
                    </td>
                    <td className="px-3 py-2.5 text-right font-money text-slate-900 dark:text-slate-100">
                      {formatCurrency(entry.totalCredit, entry.currencyCode, locale)}
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <StatusBadge
                        status={isBalanced ? 'POSTED' : 'REJECTED'}
                        label={
                          isBalanced
                            ? isId
                              ? 'Seimbang'
                              : 'Balanced'
                            : isId
                              ? 'Selisih'
                              : 'Unbalanced'
                        }
                      />
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>

          {/* Tri-Currency Compliance Summary Card (ADR-014) */}
          <div className="rounded-xl border border-border bg-surface-alt/80 dark:bg-slate-800/40 p-4 text-xs">
            <div className="flex flex-wrap items-center justify-between gap-2 pb-2.5 border-b border-border-subtle dark:border-slate-800">
              <span className="font-bold text-slate-900 dark:text-slate-100">
                {isId ? 'Kepatuhan Tri-Currency (ADR-014)' : 'Tri-Currency Compliance (ADR-014)'}
              </span>
              <span className="font-mono text-[11px] text-slate-500 dark:text-slate-400">
                {entry.currencyCode} @ {formatRate(entry.exchangeRate, locale)}
              </span>
            </div>
            <dl className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div>
                <dt className="text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {isId ? '1. MATA UANG TRANSAKSI' : '1. TRANSACTION CURRENCY'}
                </dt>
                <dd className="mt-1 font-money font-bold text-slate-900 dark:text-slate-100">
                  {formatCurrency(entry.totalDebit, entry.currencyCode, locale)}
                </dd>
              </div>
              <div>
                <dt className="text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {isId ? '2. FUNGSIONAL ENTITAS' : '2. FUNCTIONAL CURRENCY'}
                </dt>
                <dd className="mt-1 font-money font-bold text-slate-900 dark:text-slate-100">
                  {formatCurrency(functionalTotal, ledgerInfo?.currencyCode || 'IDR', locale)}
                </dd>
              </div>
              <div>
                <dt className="text-[10px] font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {isId ? '3. AUDIT KONKURENSI' : '3. CONCURRENCY & AUDIT'}
                </dt>
                <dd className="mt-1 font-mono font-bold text-slate-900 dark:text-slate-100">
                  If-Match v{entry.rowVersion}
                </dd>
              </div>
            </dl>
          </div>
        </div>
      )}
    </SideDrawer>
  );
}
