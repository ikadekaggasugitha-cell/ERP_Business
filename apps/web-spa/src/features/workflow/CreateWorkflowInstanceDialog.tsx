import { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { applyApiErrorsToForm } from '../../lib/api';
import {
  createInstanceSchema,
  useCreateWorkflowInstanceMutation,
  type CreateInstanceValues,
  type WorkflowDefinition,
} from './workflow-api';
import { useWorkflowText } from './workflow-i18n';

interface CreateWorkflowInstanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definitions: WorkflowDefinition[];
}

export function CreateWorkflowInstanceDialog({
  open,
  onOpenChange,
  definitions,
}: CreateWorkflowInstanceDialogProps) {
  const txt = useWorkflowText();
  const createInstance = useCreateWorkflowInstanceMutation();

  const {
    register,
    control,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreateInstanceValues>({
    resolver: zodResolver(createInstanceSchema),
    defaultValues: {
      definitionId: '',
      documentRef: '',
      assigneeId: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    reset({
      definitionId: definitions[0]?.id ?? '',
      documentRef: '',
      assigneeId: '',
    });
  }, [definitions, open, reset]);

  const definitionOptions = definitions.map((d) => ({
    value: d.id,
    label: `${d.code} — ${d.documentType} (v${d.version})`,
  }));

  const onSubmit = async (values: CreateInstanceValues) => {
    try {
      await createInstance.mutateAsync(values);
      toast.success(txt.instanceCreatedToast);
      onOpenChange(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, setError));
    }
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      title={txt.createInstance}
      size="md"
      footer={
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={createInstance.isPending}
          >
            {txt.cancel}
          </Button>
          <Button type="submit" form="wf-create-inst-form" disabled={createInstance.isPending}>
            {txt.createInstance}
          </Button>
        </div>
      }
    >
      <form id="wf-create-inst-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          label="Workflow Definition"
          error={errors.definitionId?.message}
          required
        >
          <Controller
            control={control}
            name="definitionId"
            render={({ field }) => (
              <Combobox
                value={field.value}
                onValueChange={field.onChange}
                options={definitionOptions}
                placeholder="Choose workflow definition..."
              />
            )}
          />
        </FormField>

        <FormField
          label="Document Reference"
          htmlFor="wf-doc-ref"
          error={errors.documentRef?.message}
          required
        >
          <Input
            id="wf-doc-ref"
            placeholder="PR-202611-01 or JV-2026-0042"
            disabled={createInstance.isPending}
            {...register('documentRef')}
          />
        </FormField>

        <FormField
          label="Assignee User ID (Optional)"
          htmlFor="wf-assignee-id"
          error={errors.assigneeId?.message}
        >
          <Input
            id="wf-assignee-id"
            placeholder="UUID (defaults to system/actor)"
            disabled={createInstance.isPending}
            {...register('assigneeId')}
          />
        </FormField>
      </form>
    </AppDialog>
  );
}
