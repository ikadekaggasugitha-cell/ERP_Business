import { useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Briefcase, Coins, Eye, EyeOff, Pencil, Plus, ShieldCheck, UserX } from 'lucide-react';
import { toast } from 'sonner';
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { applyApiErrorsToForm } from '../../lib/api';
import { addDecimals, formatCurrency, formatDate } from '../../lib/decimal';
import {
  createCompensationSchema,
  createEmploymentSchema,
  useCreateCompensationMutation,
  useCreateEmploymentMutation,
  useEmployeeCompensationsQuery,
  useEmployeeDetailQuery,
  useEmployeeEmploymentsQuery,
  type CreateCompensationValues,
  type CreateEmploymentValues,
  type HrEmployee,
  type HrPosition,
} from './hr-api';
import {
  maskEmailValue,
  maskIsoDateValue,
  maskNationalIdValue,
  useHrText,
} from './hr-i18n';

interface EmployeeDetailDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  employeeId: string | null;
  orgId: string | undefined;
  currencyCode: string;
  positions: HrPosition[];
  canWrite: boolean;
  onEdit: (emp: HrEmployee) => void;
  onRequestTerminate: (emp: HrEmployee) => void;
}

const EMPLOYMENT_TYPE_OPTIONS = [
  { value: 'Permanent', label: 'Permanent (PKWTT)' },
  { value: 'Contract', label: 'Contract (PKWT)' },
  { value: 'Probation', label: 'Probation' },
  { value: 'Intern', label: 'Internship' },
];

export function EmployeeDetailDrawer({
  open,
  onOpenChange,
  employeeId,
  orgId,
  currencyCode,
  positions,
  canWrite,
  onEdit,
  onRequestTerminate,
}: EmployeeDetailDrawerProps) {
  const txt = useHrText();
  const [piiMasked, setPiiMasked] = useState(true);
  const [subTab, setSubTab] = useState('employment');
  const [showEmploymentForm, setShowEmploymentForm] = useState(false);
  const [showCompensationForm, setShowCompensationForm] = useState(false);

  const detailQuery = useEmployeeDetailQuery(orgId, open ? employeeId : null);
  const employmentsQuery = useEmployeeEmploymentsQuery(orgId, open ? employeeId : null);
  const compensationsQuery = useEmployeeCompensationsQuery(orgId, open ? employeeId : null);

  const createEmployment = useCreateEmploymentMutation(employeeId ?? '');
  const createCompensation = useCreateCompensationMutation(employeeId ?? '');

  const employmentForm = useForm<CreateEmploymentValues>({
    resolver: zodResolver(createEmploymentSchema),
    defaultValues: {
      positionId: '',
      employmentType: 'Permanent',
      effectiveFrom: new Date().toISOString().slice(0, 10),
      effectiveTo: '',
    },
  });

  const compensationForm = useForm<CreateCompensationValues>({
    resolver: zodResolver(createCompensationSchema),
    defaultValues: {
      componentCode: 'BASE_SALARY',
      amount: '0.00',
      effectiveFrom: new Date().toISOString().slice(0, 10),
    },
  });

  const positionLookup = useMemo(() => {
    const map = new Map<string, HrPosition>();
    for (const pos of positions) {
      map.set(pos.id, pos);
    }
    return map;
  }, [positions]);

  const positionOptions = useMemo(
    () =>
      positions.map((pos) => ({
        value: pos.id,
        label: `${pos.code} — ${pos.title}`,
      })),
    [positions],
  );

  const compensationList = useMemo(
    () => compensationsQuery.data?.items ?? [],
    [compensationsQuery.data],
  );
  const totalCompensation = useMemo(() => {
    return compensationList.reduce((acc, item) => addDecimals(acc, item.amount), '0');
  }, [compensationList]);

  const employee = detailQuery.data;
  const isTerminated = employee?.status.toUpperCase() === 'TERMINATED';

  const handleEmploymentSubmit = async (values: CreateEmploymentValues) => {
    if (!employeeId) return;
    try {
      await createEmployment.mutateAsync(values);
      toast.success(txt.employmentCreated);
      employmentForm.reset({
        positionId: '',
        employmentType: 'Permanent',
        effectiveFrom: new Date().toISOString().slice(0, 10),
        effectiveTo: '',
      });
      setShowEmploymentForm(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, employmentForm.setError));
    }
  };

  const handleCompensationSubmit = async (values: CreateCompensationValues) => {
    if (!employeeId) return;
    try {
      await createCompensation.mutateAsync(values);
      toast.success(txt.compensationCreated);
      compensationForm.reset({
        componentCode: 'BASE_SALARY',
        amount: '0.00',
        effectiveFrom: new Date().toISOString().slice(0, 10),
      });
      setShowCompensationForm(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, compensationForm.setError));
    }
  };

  return (
    <SideDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={employee ? `${employee.fullName} (${employee.employeeNo})` : 'Employee Record'}
      description={employee ? `Hire Date: ${formatDate(employee.hireDate)} · v${employee.rowVersion}` : undefined}
      size="lg"
    >
      {detailQuery.isLoading ? (
        <div className="space-y-3 py-6">
          <div className="h-16 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
          <div className="h-32 animate-pulse rounded-lg bg-[var(--surface-muted)]" />
        </div>
      ) : detailQuery.isError ? (
        <ErrorState error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
      ) : employee ? (
        <div className="space-y-5">
          {/* Header status & actions */}
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4">
            <div className="flex flex-wrap items-center gap-2">
              <StatusBadge status={employee.status} />
              <span className="inline-flex items-center gap-1 rounded-full bg-[var(--surface-card)] px-2.5 py-0.5 text-xs font-medium text-[var(--text-secondary)]">
                <ShieldCheck className="h-3.5 w-3.5 text-emerald-600" />
                {piiMasked ? txt.piiMasked : txt.piiRevealed}
              </span>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPiiMasked((prev) => !prev)}
                aria-label={piiMasked ? txt.togglePiiShow : txt.togglePiiHide}
              >
                {piiMasked ? (
                  <>
                    <Eye className="mr-1.5 h-3.5 w-3.5" />
                    {txt.togglePiiShow}
                  </>
                ) : (
                  <>
                    <EyeOff className="mr-1.5 h-3.5 w-3.5" />
                    {txt.togglePiiHide}
                  </>
                )}
              </Button>
              {canWrite && (
                <>
                  <Button variant="secondary" size="sm" onClick={() => onEdit(employee)}>
                    <Pencil className="mr-1.5 h-3.5 w-3.5" />
                    {txt.editEmployee}
                  </Button>
                  {!isTerminated && (
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => onRequestTerminate(employee)}
                    >
                      <UserX className="mr-1.5 h-3.5 w-3.5" />
                      {txt.terminate}
                    </Button>
                  )}
                </>
              )}
            </div>
          </div>

          {/* PII Snapshot */}
          <dl className="grid grid-cols-2 gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4 text-sm">
            <div>
              <dt className="text-xs font-medium text-[var(--text-muted)]">{txt.colNationalId}</dt>
              <dd className="mt-0.5 font-mono text-[var(--text-primary)]" data-testid="drawer-pii-nik">
                {maskNationalIdValue(employee.nationalIdMasked, piiMasked)}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-[var(--text-muted)]">Date of Birth</dt>
              <dd className="mt-0.5 font-mono text-[var(--text-primary)]" data-testid="drawer-pii-dob">
                {maskIsoDateValue(employee.dateOfBirth, piiMasked)}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-[var(--text-muted)]">{txt.colEmail}</dt>
              <dd className="mt-0.5 text-[var(--text-primary)]" data-testid="drawer-pii-email">
                {maskEmailValue(employee.email, piiMasked)}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-medium text-[var(--text-muted)]">User Ref ID</dt>
              <dd className="mt-0.5 font-mono text-xs text-[var(--text-secondary)]">
                {employee.userRefId ?? '—'}
              </dd>
            </div>
          </dl>

          {/* Sub-tabs: Employment History & Compensation Grades */}
          <Tabs value={subTab} onValueChange={setSubTab}>
            <TabsList>
              <TabsTrigger value="employment">
                <Briefcase className="mr-1.5 h-3.5 w-3.5" />
                {txt.subTabEmployment}
              </TabsTrigger>
              <TabsTrigger value="compensation">
                <Coins className="mr-1.5 h-3.5 w-3.5" />
                {txt.subTabCompensation}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="employment" className="mt-4 space-y-4">
              {canWrite && (
                <div className="flex justify-end">
                  <Button
                    size="sm"
                    variant={showEmploymentForm ? 'secondary' : 'primary'}
                    onClick={() => setShowEmploymentForm((v) => !v)}
                  >
                    <Plus className="mr-1.5 h-3.5 w-3.5" />
                    {txt.addEmployment}
                  </Button>
                </div>
              )}

              {showEmploymentForm && canWrite && (
                <form
                  onSubmit={employmentForm.handleSubmit(handleEmploymentSubmit)}
                  className="grid grid-cols-1 gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4 sm:grid-cols-2"
                >
                  <FormField
                    label="Position"
                    error={employmentForm.formState.errors.positionId?.message}
                    required
                    className="sm:col-span-2"
                  >
                    <Controller
                      control={employmentForm.control}
                      name="positionId"
                      render={({ field }) => (
                        <Combobox
                          value={field.value}
                          onValueChange={field.onChange}
                          options={positionOptions}
                          placeholder="Choose position..."
                        />
                      )}
                    />
                  </FormField>

                  <FormField
                    label="Employment Type"
                    error={employmentForm.formState.errors.employmentType?.message}
                    required
                  >
                    <Controller
                      control={employmentForm.control}
                      name="employmentType"
                      render={({ field }) => (
                        <Select
                          value={field.value}
                          onValueChange={field.onChange}
                          options={EMPLOYMENT_TYPE_OPTIONS}
                        />
                      )}
                    />
                  </FormField>

                  <FormField
                    label="Effective From"
                    error={employmentForm.formState.errors.effectiveFrom?.message}
                    required
                  >
                    <Input type="date" {...employmentForm.register('effectiveFrom')} />
                  </FormField>

                  <FormField
                    label="Effective To (Optional)"
                    error={employmentForm.formState.errors.effectiveTo?.message}
                  >
                    <Input type="date" {...employmentForm.register('effectiveTo')} />
                  </FormField>

                  <div className="flex items-end justify-end gap-2">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => setShowEmploymentForm(false)}
                    >
                      {txt.cancel}
                    </Button>
                    <Button type="submit" size="sm" disabled={createEmployment.isPending}>
                      {txt.save}
                    </Button>
                  </div>
                </form>
              )}

              {employmentsQuery.isError ? (
                <ErrorState
                  error={employmentsQuery.error}
                  onRetry={() => void employmentsQuery.refetch()}
                />
              ) : (employmentsQuery.data?.items.length ?? 0) === 0 ? (
                <EmptyState title={txt.emptyEmployments} />
              ) : (
                <div className="space-y-2">
                  {employmentsQuery.data?.items.map((emp) => {
                    const pos = positionLookup.get(emp.positionId);
                    return (
                      <div
                        key={emp.id}
                        className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-card)] px-4 py-3 text-sm"
                      >
                        <div>
                          <div className="font-medium text-[var(--text-primary)]">
                            {pos ? `${pos.code} — ${pos.title}` : emp.positionId}
                          </div>
                          <div className="mt-0.5 text-xs text-[var(--text-muted)]">
                            {emp.employmentType} · {formatDate(emp.effectiveFrom)}
                            {emp.effectiveTo ? ` -> ${formatDate(emp.effectiveTo)}` : ' -> Present'}
                          </div>
                        </div>
                        <span className="font-mono text-xs text-[var(--text-muted)]">
                          v{emp.rowVersion}
                        </span>
                      </div>
                    );
                  })}
                </div>
              )}
            </TabsContent>

            <TabsContent value="compensation" className="mt-4 space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-emerald-500/20 bg-emerald-500/5 px-4 py-3">
                <div>
                  <div className="text-xs font-medium text-[var(--text-muted)]">
                    {txt.totalCompensationLabel}
                  </div>
                  <div
                    className="font-mono text-lg font-semibold text-[var(--text-primary)]"
                    data-testid="hr-total-compensation"
                  >
                    {formatCurrency(totalCompensation, currencyCode)}
                  </div>
                </div>
                {canWrite && (
                  <Button
                    size="sm"
                    variant={showCompensationForm ? 'secondary' : 'primary'}
                    onClick={() => setShowCompensationForm((v) => !v)}
                  >
                    <Plus className="mr-1.5 h-3.5 w-3.5" />
                    {txt.addCompensation}
                  </Button>
                )}
              </div>

              {showCompensationForm && canWrite && (
                <form
                  onSubmit={compensationForm.handleSubmit(handleCompensationSubmit)}
                  className="grid grid-cols-1 gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4 sm:grid-cols-2"
                >
                  <FormField
                    label="Component Code"
                    error={compensationForm.formState.errors.componentCode?.message}
                    required
                  >
                    <Input
                      placeholder="BASE_SALARY / TRANSPORT_ALLOWANCE"
                      {...compensationForm.register('componentCode')}
                    />
                  </FormField>

                  <FormField
                    label="Effective From"
                    error={compensationForm.formState.errors.effectiveFrom?.message}
                    required
                  >
                    <Input type="date" {...compensationForm.register('effectiveFrom')} />
                  </FormField>

                  <FormField
                    label="Amount"
                    error={compensationForm.formState.errors.amount?.message}
                    required
                    className="sm:col-span-2"
                  >
                    <Controller
                      control={compensationForm.control}
                      name="amount"
                      render={({ field }) => (
                        <MoneyInput
                          value={field.value}
                          onValueChange={field.onChange}
                          currency={currencyCode}
                        />
                      )}
                    />
                  </FormField>

                  <div className="flex justify-end gap-2 sm:col-span-2">
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => setShowCompensationForm(false)}
                    >
                      {txt.cancel}
                    </Button>
                    <Button type="submit" size="sm" disabled={createCompensation.isPending}>
                      {txt.save}
                    </Button>
                  </div>
                </form>
              )}

              {compensationsQuery.isError ? (
                <ErrorState
                  error={compensationsQuery.error}
                  onRetry={() => void compensationsQuery.refetch()}
                />
              ) : compensationList.length === 0 ? (
                <EmptyState title={txt.emptyCompensations} />
              ) : (
                <div className="space-y-2">
                  {compensationList.map((comp) => (
                    <div
                      key={comp.id}
                      className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-card)] px-4 py-3 text-sm"
                    >
                      <div>
                        <div className="font-mono font-medium text-[var(--text-primary)]">
                          {comp.componentCode}
                        </div>
                        <div className="text-xs text-[var(--text-muted)]">
                          Effective {formatDate(comp.effectiveFrom)}
                        </div>
                      </div>
                      <div className="font-mono font-semibold text-[var(--text-primary)]">
                        {formatCurrency(comp.amount, currencyCode)}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </TabsContent>
          </Tabs>
        </div>
      ) : null}
    </SideDrawer>
  );
}
