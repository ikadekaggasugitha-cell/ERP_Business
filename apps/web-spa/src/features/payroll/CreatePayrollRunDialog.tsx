import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { applyApiErrorsToForm } from '../../lib/api';
import {
  createPayrollRunSchema,
  useCreatePayrollRunMutation,
  type CreatePayrollRunValues,
} from './payroll-api';
import { usePayrollText } from './payroll-i18n';

interface CreatePayrollRunDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (runId: string) => void;
}

export function CreatePayrollRunDialog({
  open,
  onOpenChange,
  onCreated,
}: CreatePayrollRunDialogProps) {
  const txt = usePayrollText();
  const createRun = useCreatePayrollRunMutation();
  const now = new Date();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreatePayrollRunValues>({
    resolver: zodResolver(createPayrollRunSchema),
    defaultValues: {
      runNo: `PR-${now.getFullYear()}${String(now.getMonth() + 1).padStart(2, '0')}-01`,
      periodMonth: now.getMonth() + 1,
      periodYear: now.getFullYear(),
    },
  });

  useEffect(() => {
    if (!open) return;
    const d = new Date();
    reset({
      runNo: `PR-${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, '0')}-01`,
      periodMonth: d.getMonth() + 1,
      periodYear: d.getFullYear(),
    });
  }, [open, reset]);

  const onSubmit = async (values: CreatePayrollRunValues) => {
    try {
      const created = await createRun.mutateAsync(values);
      toast.success(txt.runCreated);
      onOpenChange(false);
      onCreated?.(created.id);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, setError));
    }
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      title={txt.newRun}
      size="md"
      footer={
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={createRun.isPending}
          >
            {txt.cancel}
          </Button>
          <Button type="submit" form="payroll-create-run-form" disabled={createRun.isPending}>
            {txt.create}
          </Button>
        </div>
      }
    >
      <form id="payroll-create-run-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          label={txt.colRunNo}
          htmlFor="pr-run-no"
          error={errors.runNo?.message}
          required
        >
          <Input
            id="pr-run-no"
            placeholder="PR-202611-01"
            disabled={createRun.isPending}
            {...register('runNo')}
          />
        </FormField>

        <div className="grid grid-cols-2 gap-4">
          <FormField
            label="Period Month (1–12)"
            htmlFor="pr-month"
            error={errors.periodMonth?.message}
            required
          >
            <Input
              id="pr-month"
              type="number"
              min={1}
              max={12}
              disabled={createRun.isPending}
              {...register('periodMonth', { valueAsNumber: true })}
            />
          </FormField>

          <FormField
            label="Period Year"
            htmlFor="pr-year"
            error={errors.periodYear?.message}
            required
          >
            <Input
              id="pr-year"
              type="number"
              min={2000}
              max={2100}
              disabled={createRun.isPending}
              {...register('periodYear', { valueAsNumber: true })}
            />
          </FormField>
        </div>
      </form>
    </AppDialog>
  );
}
