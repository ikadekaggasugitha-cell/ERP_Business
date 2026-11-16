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
  createDefinitionSchema,
  useCreateWorkflowDefinitionMutation,
  type CreateDefinitionValues,
} from './workflow-api';
import { useWorkflowText } from './workflow-i18n';

interface CreateWorkflowDefinitionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateWorkflowDefinitionDialog({
  open,
  onOpenChange,
}: CreateWorkflowDefinitionDialogProps) {
  const txt = useWorkflowText();
  const createDefinition = useCreateWorkflowDefinitionMutation();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreateDefinitionValues>({
    resolver: zodResolver(createDefinitionSchema),
    defaultValues: {
      code: '',
      documentType: 'JOURNAL_ENTRY',
      approverRole: 'FINANCE_APPROVER',
      slaHours: 24,
    },
  });

  useEffect(() => {
    if (!open) return;
    reset({
      code: '',
      documentType: 'JOURNAL_ENTRY',
      approverRole: 'FINANCE_APPROVER',
      slaHours: 24,
    });
  }, [open, reset]);

  const onSubmit = async (values: CreateDefinitionValues) => {
    try {
      await createDefinition.mutateAsync(values);
      toast.success(txt.definitionCreatedToast);
      onOpenChange(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, setError));
    }
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      title={txt.createDefinition}
      size="md"
      footer={
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={createDefinition.isPending}
          >
            {txt.cancel}
          </Button>
          <Button type="submit" form="wf-create-def-form" disabled={createDefinition.isPending}>
            {txt.createDefinition}
          </Button>
        </div>
      }
    >
      <form id="wf-create-def-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          label="Definition Code"
          htmlFor="wf-def-code"
          error={errors.code?.message}
          required
        >
          <Input
            id="wf-def-code"
            placeholder="WF-JE-DUAL"
            disabled={createDefinition.isPending}
            {...register('code')}
          />
        </FormField>

        <FormField
          label="Document Type"
          htmlFor="wf-doc-type"
          error={errors.documentType?.message}
          required
        >
          <Input
            id="wf-doc-type"
            placeholder="JOURNAL_ENTRY / PAYROLL_RUN / PURCHASE_ORDER"
            disabled={createDefinition.isPending}
            {...register('documentType')}
          />
        </FormField>

        <div className="grid grid-cols-2 gap-4">
          <FormField
            label="Step 1 Approver Role"
            htmlFor="wf-approver-role"
            error={errors.approverRole?.message}
          >
            <Input
              id="wf-approver-role"
              placeholder="FINANCE_APPROVER"
              disabled={createDefinition.isPending}
              {...register('approverRole')}
            />
          </FormField>

          <FormField
            label="Escalation SLA (Hours)"
            htmlFor="wf-sla-hours"
            error={errors.slaHours?.message}
            required
          >
            <Input
              id="wf-sla-hours"
              type="number"
              min={1}
              max={720}
              disabled={createDefinition.isPending}
              {...register('slaHours', { valueAsNumber: true })}
            />
          </FormField>
        </div>
      </form>
    </AppDialog>
  );
}
