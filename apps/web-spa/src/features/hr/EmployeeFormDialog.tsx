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
  createEmployeeSchema,
  updateEmployeeSchema,
  useCreateEmployeeMutation,
  useUpdateEmployeeMutation,
  type CreateEmployeeValues,
  type HrEmployee,
} from './hr-api';
import { useHrText } from './hr-i18n';

interface EmployeeFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  employee?: HrEmployee | null;
}

export function EmployeeFormDialog({
  open,
  onOpenChange,
  employee,
}: EmployeeFormDialogProps) {
  const txt = useHrText();
  const isEdit = Boolean(employee);
  const createMutation = useCreateEmployeeMutation();
  const updateMutation = useUpdateEmployeeMutation();

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = useForm<CreateEmployeeValues>({
    resolver: zodResolver(isEdit ? updateEmployeeSchema.extend({
      employeeNo: createEmployeeSchema.shape.employeeNo,
      hireDate: createEmployeeSchema.shape.hireDate,
    }) : createEmployeeSchema),
    defaultValues: {
      employeeNo: '',
      fullName: '',
      nationalId: '',
      dateOfBirth: '',
      email: '',
      hireDate: new Date().toISOString().slice(0, 10),
      userRefId: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    if (employee) {
      reset({
        employeeNo: employee.employeeNo,
        fullName: employee.fullName,
        nationalId: '',
        dateOfBirth: employee.dateOfBirth ?? '',
        email: employee.email ?? '',
        hireDate: employee.hireDate,
        userRefId: employee.userRefId ?? '',
      });
    } else {
      reset({
        employeeNo: '',
        fullName: '',
        nationalId: '',
        dateOfBirth: '',
        email: '',
        hireDate: new Date().toISOString().slice(0, 10),
        userRefId: '',
      });
    }
  }, [employee, open, reset]);

  const isPending = createMutation.isPending || updateMutation.isPending;

  const onSubmit = async (values: CreateEmployeeValues) => {
    try {
      if (employee) {
        await updateMutation.mutateAsync({
          id: employee.id,
          rowVersion: employee.rowVersion,
          values: {
            fullName: values.fullName,
            nationalId: values.nationalId,
            dateOfBirth: values.dateOfBirth,
            email: values.email,
            userRefId: values.userRefId,
          },
        });
        toast.success(txt.employeeUpdated);
      } else {
        await createMutation.mutateAsync(values);
        toast.success(txt.employeeCreated);
      }
      onOpenChange(false);
    } catch (err) {
      const msg = applyApiErrorsToForm(err, setError);
      toast.error(msg);
    }
  };

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? txt.editEmployee : txt.addEmployee}
      size="lg"
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={isPending}>
            {txt.cancel}
          </Button>
          <Button type="submit" form="hr-employee-form" disabled={isPending}>
            {isEdit ? txt.save : txt.create}
          </Button>
        </div>
      }
    >
      <form
        id="hr-employee-form"
        onSubmit={handleSubmit(onSubmit)}
        className="grid grid-cols-1 gap-4 sm:grid-cols-2"
      >
        <FormField
          label={txt.colEmployeeNo}
          htmlFor="emp-no"
          error={errors.employeeNo?.message}
          required
        >
          <Input
            id="emp-no"
            placeholder="EMP-2026-001"
            disabled={isEdit || isPending}
            {...register('employeeNo')}
          />
        </FormField>

        <FormField
          label={txt.colFullName}
          htmlFor="emp-name"
          error={errors.fullName?.message}
          required
        >
          <Input
            id="emp-name"
            placeholder="Budi Santoso"
            disabled={isPending}
            {...register('fullName')}
          />
        </FormField>

        <FormField
          label={txt.colHireDate}
          htmlFor="emp-hire-date"
          error={errors.hireDate?.message}
          required
        >
          <Input
            id="emp-hire-date"
            type="date"
            disabled={isEdit || isPending}
            {...register('hireDate')}
          />
        </FormField>

        <FormField
          label={txt.colEmail}
          htmlFor="emp-email"
          error={errors.email?.message}
        >
          <Input
            id="emp-email"
            type="email"
            placeholder="budi.santoso@company.co.id"
            disabled={isPending}
            {...register('email')}
          />
        </FormField>

        <FormField
          label={isEdit ? `${txt.colNationalId} (optional replace)` : 'National ID (NIK)'}
          htmlFor="emp-nik"
          error={errors.nationalId?.message}
        >
          <Input
            id="emp-nik"
            placeholder={employee?.nationalIdMasked ?? '3174011990010001'}
            disabled={isPending}
            {...register('nationalId')}
          />
        </FormField>

        <FormField
          label="Date of Birth"
          htmlFor="emp-dob"
          error={errors.dateOfBirth?.message}
        >
          <Input
            id="emp-dob"
            type="date"
            disabled={isPending}
            {...register('dateOfBirth')}
          />
        </FormField>

        <FormField
          label="Linked User Ref ID (Optional)"
          htmlFor="emp-user-ref"
          error={errors.userRefId?.message}
          className="sm:col-span-2"
        >
          <Input
            id="emp-user-ref"
            placeholder="UUID from user-service (optional)"
            disabled={isPending}
            {...register('userRefId')}
          />
        </FormField>
      </form>
    </AppDialog>
  );
}
