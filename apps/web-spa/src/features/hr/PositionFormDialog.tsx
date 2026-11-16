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
  createPositionSchema,
  useCreatePositionMutation,
  type CreatePositionValues,
  type HrPosition,
} from './hr-api';
import { useHrText } from './hr-i18n';

interface PositionFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  positions: HrPosition[];
  defaultOrgId?: string;
}

export function PositionFormDialog({
  open,
  onOpenChange,
  positions,
  defaultOrgId,
}: PositionFormDialogProps) {
  const txt = useHrText();
  const createPosition = useCreatePositionMutation();

  const {
    register,
    control,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreatePositionValues>({
    resolver: zodResolver(createPositionSchema),
    defaultValues: {
      code: '',
      title: '',
      organizationRefId: defaultOrgId ?? '',
      parentPositionId: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    reset({
      code: '',
      title: '',
      organizationRefId: defaultOrgId ?? '',
      parentPositionId: '',
    });
  }, [defaultOrgId, open, reset]);

  const parentOptions = positions.map((pos) => ({
    value: pos.id,
    label: `${pos.code} — ${pos.title}`,
  }));

  const onSubmit = async (values: CreatePositionValues) => {
    try {
      await createPosition.mutateAsync(values);
      toast.success(txt.positionCreated);
      onOpenChange(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, setError));
    }
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      title={txt.addPosition}
      size="md"
      footer={
        <div className="flex justify-end gap-2">
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={createPosition.isPending}
          >
            {txt.cancel}
          </Button>
          <Button type="submit" form="hr-position-form" disabled={createPosition.isPending}>
            {txt.create}
          </Button>
        </div>
      }
    >
      <form id="hr-position-form" onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          label={txt.colPositionCode}
          htmlFor="pos-code"
          error={errors.code?.message}
          required
        >
          <Input
            id="pos-code"
            placeholder="POS-FIN-MGR"
            disabled={createPosition.isPending}
            {...register('code')}
          />
        </FormField>

        <FormField
          label={txt.colPositionTitle}
          htmlFor="pos-title"
          error={errors.title?.message}
          required
        >
          <Input
            id="pos-title"
            placeholder="Finance & Accounting Manager"
            disabled={createPosition.isPending}
            {...register('title')}
          />
        </FormField>

        <FormField
          label={txt.colParentPosition}
          htmlFor="pos-parent"
          error={errors.parentPositionId?.message}
        >
          <Controller
            control={control}
            name="parentPositionId"
            render={({ field }) => (
              <Combobox
                value={field.value ?? ''}
                onValueChange={field.onChange}
                options={parentOptions}
                allowClear
                placeholder="Select reporting position..."
              />
            )}
          />
        </FormField>

        <FormField
          label={txt.colOrgRef}
          htmlFor="pos-org-ref"
          error={errors.organizationRefId?.message}
        >
          <Input
            id="pos-org-ref"
            placeholder="Defaults to active entity UUID"
            disabled={createPosition.isPending}
            {...register('organizationRefId')}
          />
        </FormField>
      </form>
    </AppDialog>
  );
}
