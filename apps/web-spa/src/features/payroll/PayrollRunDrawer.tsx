import { useMemo, useState } from 'react';
import { Calculator, CheckCircle2, Plus, Send, Trash2, X } from 'lucide-react';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select } from '../../components/ui/Select';
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Switch } from '../../components/ui/Switch';
import { extractApiFieldErrors } from '../../lib/api';
import { addDecimals, formatCurrency } from '../../lib/decimal';
import { useEmployeesQuery } from '../hr/hr-api';
import {
  PTKP_OPTIONS,
  useApprovePayrollMutation,
  useCalculatePayrollMutation,
  useDeletePayrollRunMutation,
  usePayrollRunDetailQuery,
  usePostPayrollMutation,
  type CalculateEmployeeInput,
} from './payroll-api';
import { formatPayrollPeriod, usePayrollText } from './payroll-i18n';

interface PayrollRunDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  runId: string | null;
  orgId: string | undefined;
  currencyCode: string;
  canWrite: boolean;
}

const LIFECYCLE_STEPS = ['DRAFT', 'CALCULATED', 'APPROVED', 'POSTED'] as const;

export function PayrollRunDrawer({
  open,
  onOpenChange,
  runId,
  orgId,
  currencyCode,
  canWrite,
}: PayrollRunDrawerProps) {
  const txt = usePayrollText();
  const detailQuery = usePayrollRunDetailQuery(orgId, open ? runId : null);
  const employeesQuery = useEmployeesQuery(orgId, { page: 1, size: 100, status: 'Active' });

  const calculateMutation = useCalculatePayrollMutation();
  const approveMutation = useApprovePayrollMutation();
  const postMutation = usePostPayrollMutation();
  const deleteMutation = useDeletePayrollRunMutation();

  const [calcDialogOpen, setCalcDialogOpen] = useState(false);
  const [totalAmountInput, setTotalAmountInput] = useState('');
  const [calcLines, setCalcLines] = useState<CalculateEmployeeInput[]>([]);

  const [postDialogOpen, setPostDialogOpen] = useState(false);
  const [glJournalRefInput, setGlJournalRefInput] = useState('');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  const run = detailQuery.data;
  const statusUpper = (run?.status ?? '').toUpperCase();
  const stepIndex = LIFECYCLE_STEPS.indexOf(
    statusUpper as (typeof LIFECYCLE_STEPS)[number],
  );

  const employees = useMemo(() => employeesQuery.data?.items ?? [], [employeesQuery.data]);
  const employeeLookup = useMemo(() => {
    const m = new Map<string, string>();
    for (const emp of employees) {
      m.set(emp.id, `${emp.employeeNo} — ${emp.fullName}`);
    }
    return m;
  }, [employees]);

  const employeeOptions = useMemo(
    () =>
      employees.map((emp) => ({
        value: emp.id,
        label: `${emp.employeeNo} — ${emp.fullName}`,
      })),
    [employees],
  );

  const results = useMemo(() => run?.results ?? [], [run]);
  const totals = useMemo(() => {
    let gross = '0';
    let deduction = '0';
    let tax = '0';
    let net = '0';
    for (const r of results) {
      gross = addDecimals(gross, r.grossAmount);
      deduction = addDecimals(deduction, r.deductionAmount);
      tax = addDecimals(tax, r.taxAmount);
      net = addDecimals(net, r.netAmount);
    }
    return { gross, deduction, tax, net };
  }, [results]);

  const anyMutating =
    calculateMutation.isPending ||
    approveMutation.isPending ||
    postMutation.isPending ||
    deleteMutation.isPending;

  const openCalculateDialog = () => {
    setTotalAmountInput('');
    setCalcLines([]);
    setCalcDialogOpen(true);
  };

  const addCalcLine = () => {
    setCalcLines((prev) => [
      ...prev,
      {
        employeeId: employees[0]?.id ?? '',
        basicSalary: '10000000.00',
        allowance: '0.00',
        bonus: '0.00',
        otherDeductions: '0.00',
        ptkp: 'TK/0',
        hasNpwp: true,
      },
    ]);
  };

  const updateCalcLine = (idx: number, patch: Partial<CalculateEmployeeInput>) => {
    setCalcLines((prev) => prev.map((line, i) => (i === idx ? { ...line, ...patch } : line)));
  };

  const removeCalcLine = (idx: number) => {
    setCalcLines((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleRunCalculate = async () => {
    if (!run) return;
    try {
      await calculateMutation.mutateAsync({
        id: run.id,
        rowVersion: run.rowVersion,
        payload: {
          totalAmount:
            calcLines.length === 0 && totalAmountInput && totalAmountInput !== '0.00'
              ? totalAmountInput
              : undefined,
          employees: calcLines.filter((l) => l.employeeId.trim() !== ''),
        },
      });
      toast.success(txt.runCalculated);
      setCalcDialogOpen(false);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleApprove = async () => {
    if (!run) return;
    try {
      await approveMutation.mutateAsync({ id: run.id, rowVersion: run.rowVersion });
      toast.success(txt.runApproved);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handlePostToGl = async () => {
    if (!run) return;
    try {
      await postMutation.mutateAsync({
        id: run.id,
        rowVersion: run.rowVersion,
        glJournalRef: glJournalRefInput,
      });
      toast.success(txt.runPosted);
      setPostDialogOpen(false);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleDeleteDraft = async () => {
    if (!run) return;
    try {
      await deleteMutation.mutateAsync({ id: run.id, rowVersion: run.rowVersion });
      toast.success(txt.runDeleted);
      setDeleteConfirmOpen(false);
      onOpenChange(false);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  return (
    <>
      <SideDrawer
        open={open}
        onOpenChange={onOpenChange}
        title={run ? `${run.runNo} (${formatPayrollPeriod(run.periodMonth, run.periodYear)})` : 'Payroll Run'}
        description={
          run
            ? `Version v${run.rowVersion}${run.glJournalRef ? ` · GL Ref: ${run.glJournalRef}` : ''}`
            : undefined
        }
        size="xl"
      >
        {detailQuery.isLoading ? (
          <div className="space-y-3 py-6">
            <div className="h-16 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
            <div className="h-40 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
          </div>
        ) : detailQuery.isError ? (
          <ErrorState error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
        ) : run ? (
          <div className="space-y-5">
            {/* Status Stepper */}
            <div className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <StatusBadge status={run.status} />
                  {run.correlationId && (
                    <span className="font-mono text-xs text-[var(--text-muted)]">
                      corr: {run.correlationId.slice(0, 8)}
                    </span>
                  )}
                </div>

                {canWrite && (
                  <div className="flex flex-wrap items-center gap-2">
                    {statusUpper === 'DRAFT' && (
                      <>
                        <Button size="sm" onClick={openCalculateDialog} disabled={anyMutating}>
                          <Calculator className="mr-1.5 h-3.5 w-3.5" />
                          {txt.calculate}
                        </Button>
                        <Button
                          size="sm"
                          variant="danger"
                          onClick={() => setDeleteConfirmOpen(true)}
                          disabled={anyMutating}
                        >
                          <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                          {txt.deleteDraft}
                        </Button>
                      </>
                    )}
                    {statusUpper === 'CALCULATED' && (
                      <Button size="sm" onClick={handleApprove} disabled={anyMutating}>
                        <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                        {txt.approve}
                      </Button>
                    )}
                    {statusUpper === 'APPROVED' && (
                      <Button
                        size="sm"
                        onClick={() => {
                          setGlJournalRefInput(
                            run.runNo.startsWith('PR-') ? `JV-${run.runNo}` : `JV-PR-${run.runNo}`,
                          );
                          setPostDialogOpen(true);
                        }}
                        disabled={anyMutating}
                      >
                        <Send className="mr-1.5 h-3.5 w-3.5" />
                        {txt.postToGl}
                      </Button>
                    )}
                  </div>
                )}
              </div>

              <div className="mt-4 grid grid-cols-4 gap-2">
                {LIFECYCLE_STEPS.map((step, idx) => {
                  const completed = stepIndex >= idx;
                  return (
                    <div
                      key={step}
                      className={`rounded-lg border px-2.5 py-1.5 text-center text-xs font-semibold ${
                        completed
                          ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
                          : 'border-[var(--border-subtle)] bg-[var(--surface-card)] text-[var(--text-muted)]'
                      }`}
                    >
                      {idx + 1}. {step}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Fixed-Point Summary Totals */}
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-3">
                <div className="text-xs text-[var(--text-muted)]">{txt.totalGross}</div>
                <div
                  className="mt-1 font-mono text-sm font-semibold text-[var(--text-primary)]"
                  data-testid="payroll-total-gross"
                >
                  {formatCurrency(totals.gross, currencyCode)}
                </div>
              </div>
              <div className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-3">
                <div className="text-xs text-[var(--text-muted)]">{txt.totalDeductions}</div>
                <div className="mt-1 font-mono text-sm font-semibold text-[var(--text-primary)]">
                  {formatCurrency(totals.deduction, currencyCode)}
                </div>
              </div>
              <div className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-3">
                <div className="text-xs text-[var(--text-muted)]">{txt.totalTax}</div>
                <div className="mt-1 font-mono text-sm font-semibold text-amber-700 dark:text-amber-300">
                  {formatCurrency(totals.tax, currencyCode)}
                </div>
              </div>
              <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-3">
                <div className="text-xs text-[var(--text-muted)]">{txt.totalNet}</div>
                <div
                  className="mt-1 font-mono text-sm font-semibold text-emerald-700 dark:text-emerald-300"
                  data-testid="payroll-total-net"
                >
                  {formatCurrency(totals.net, currencyCode)}
                </div>
              </div>
            </div>

            {/* Results & Statutory Breakdown */}
            {results.length === 0 ? (
              <EmptyState
                title="No payslip calculation results yet"
                description="Run Calculate Payroll on a Draft batch to generate employee payslips and BPJS / PPh 21 deductions."
              />
            ) : (
              <div className="space-y-3">
                <h3 className="text-sm font-semibold text-[var(--text-primary)]">
                  Employee Payslips ({results.length})
                </h3>
                {results.map((res) => (
                  <div
                    key={res.id}
                    className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 text-sm"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--border-subtle)] pb-2">
                      <span className="font-medium text-[var(--text-primary)]">
                        {employeeLookup.get(res.employeeId) ?? res.employeeId}
                      </span>
                      <span className="font-mono text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                        Net: {formatCurrency(res.netAmount, currencyCode)}
                      </span>
                    </div>
                    <div className="mt-2 grid grid-cols-3 gap-2 text-xs text-[var(--text-secondary)]">
                      <div>Gross: {formatCurrency(res.grossAmount, currencyCode)}</div>
                      <div>Deductions: {formatCurrency(res.deductionAmount, currencyCode)}</div>
                      <div>Tax (PPh 21): {formatCurrency(res.taxAmount, currencyCode)}</div>
                    </div>
                    {res.statutoryContributions && res.statutoryContributions.length > 0 && (
                      <div className="mt-3 rounded-lg bg-[var(--surface-subtle)] p-2.5 text-xs">
                        <div className="mb-1 font-medium text-[var(--text-muted)]">
                          Statutory Contributions (BPJS)
                        </div>
                        <div className="grid grid-cols-1 gap-1 sm:grid-cols-2">
                          {res.statutoryContributions.map((sc) => (
                            <div key={sc.id} className="flex justify-between font-mono">
                              <span>{sc.contributionType}</span>
                              <span>
                                EE {formatCurrency(sc.employeeAmount, currencyCode)} / ER{' '}
                                {formatCurrency(sc.employerAmount, currencyCode)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : null}
      </SideDrawer>

      {/* Calculate Payroll Dialog */}
      <AppDialog
        open={calcDialogOpen}
        onOpenChange={setCalcDialogOpen}
        title={txt.calculate}
        description="Compute Indonesia statutory payroll (PPh 21 + BPJS) per employee or enter a batch total."
        size="xl"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setCalcDialogOpen(false)}
              disabled={calculateMutation.isPending}
            >
              {txt.cancel}
            </Button>
            <Button onClick={handleRunCalculate} disabled={calculateMutation.isPending}>
              {txt.calculate}
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs text-[var(--text-secondary)]">
              Add explicit employee salary lines below, or leave lines empty and specify a batch total amount.
            </p>
            <Button size="sm" variant="secondary" onClick={addCalcLine}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Add Employee Line
            </Button>
          </div>

          {calcLines.length === 0 ? (
            <FormField
              label="Batch Total Amount (Optional Quick Mode)"
              hint="Leave 0.00 to let backend auto-resolve active employee compensation grades."
            >
              <MoneyInput
                value={totalAmountInput}
                onValueChange={setTotalAmountInput}
                currency={currencyCode}
              />
            </FormField>
          ) : (
            <div className="max-h-96 space-y-3 overflow-y-auto pr-1">
              {calcLines.map((line, idx) => (
                <div
                  key={idx}
                  className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-3"
                >
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-semibold text-[var(--text-muted)]">
                      Employee #{idx + 1}
                    </span>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => removeCalcLine(idx)}
                      aria-label="Remove employee line"
                    >
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <FormField label="Employee" className="sm:col-span-2">
                      <Combobox
                        value={line.employeeId}
                        onValueChange={(v) => updateCalcLine(idx, { employeeId: v })}
                        options={employeeOptions}
                        placeholder="Select employee..."
                      />
                    </FormField>
                    <FormField label="PTKP Tax Status">
                      <Select
                        value={line.ptkp ?? 'TK/0'}
                        onValueChange={(v) => updateCalcLine(idx, { ptkp: v })}
                        options={[...PTKP_OPTIONS]}
                      />
                    </FormField>
                    <FormField label="Basic Salary">
                      <MoneyInput
                        value={line.basicSalary ?? '0.00'}
                        onValueChange={(v) => updateCalcLine(idx, { basicSalary: v })}
                        currency={currencyCode}
                      />
                    </FormField>
                    <FormField label="Allowance">
                      <MoneyInput
                        value={line.allowance ?? '0.00'}
                        onValueChange={(v) => updateCalcLine(idx, { allowance: v })}
                        currency={currencyCode}
                      />
                    </FormField>
                    <FormField label="Other Deductions">
                      <MoneyInput
                        value={line.otherDeductions ?? '0.00'}
                        onValueChange={(v) => updateCalcLine(idx, { otherDeductions: v })}
                        currency={currencyCode}
                      />
                    </FormField>
                    <div className="flex items-center gap-2 sm:col-span-3">
                      <Switch
                        checked={line.hasNpwp ?? true}
                        onCheckedChange={(checked) => updateCalcLine(idx, { hasNpwp: checked })}
                        label="Has NPWP"
                      />
                      <span className="text-xs text-[var(--text-secondary)]">
                        Employee holds active NPWP
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </AppDialog>

      {/* Post to GL Dialog */}
      <AppDialog
        open={postDialogOpen}
        onOpenChange={setPostDialogOpen}
        title={txt.postToGl}
        description="Finalize payroll run and assign an optional GL Journal Reference."
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setPostDialogOpen(false)}
              disabled={postMutation.isPending}
            >
              {txt.cancel}
            </Button>
            <Button onClick={handlePostToGl} disabled={postMutation.isPending}>
              {txt.postToGl}
            </Button>
          </div>
        }
      >
        <FormField label="GL Journal Reference" htmlFor="pr-gl-ref">
          <Input
            id="pr-gl-ref"
            value={glJournalRefInput}
            onChange={(e) => setGlJournalRefInput(e.target.value)}
            placeholder="JV-PR-202611-01"
          />
        </FormField>
      </AppDialog>

      {/* Delete Draft Dialog */}
      <AppDialog
        open={deleteConfirmOpen}
        onOpenChange={setDeleteConfirmOpen}
        title={txt.confirmDeleteTitle}
        description={txt.confirmDeleteDesc}
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => setDeleteConfirmOpen(false)}
              disabled={deleteMutation.isPending}
            >
              {txt.cancel}
            </Button>
            <Button
              variant="danger"
              onClick={handleDeleteDraft}
              disabled={deleteMutation.isPending}
            >
              {txt.deleteDraft}
            </Button>
          </div>
        }
      >
        {run && (
          <div className="rounded-lg border border-rose-500/20 bg-rose-500/5 p-3 text-sm font-mono">
            {run.runNo} · v{run.rowVersion}
          </div>
        )}
      </AppDialog>
    </>
  );
}
