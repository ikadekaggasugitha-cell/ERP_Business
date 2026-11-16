import { useEffect, useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Select } from '../../components/ui/Select';
import { Switch } from '../../components/ui/Switch';
import { ApiClientError, applyApiErrorsToForm } from '../../lib/api';
import {
  createCurrency,
  createExchangeRate,
  createExchangeRateType,
  currencyQueryKeys,
  updateCurrency,
  updateExchangeRate,
} from './api';
import {
  currencyFormSchema,
  exchangeRateFormSchema,
  rateTypeFormSchema,
  type CurrencyFormValues,
  type ExchangeRateFormValues,
  type RateTypeFormValues,
} from './schemas';
import type {
  CurrencyDto,
  ExchangeRateDto,
  ExchangeRateTypeDto,
} from './types';

function todayIso(): string {
  return new Date().toISOString().slice(0, 10);
}

interface CurrencyFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingCurrency: CurrencyDto | null;
}

export function CurrencyFormDialog({
  open,
  onOpenChange,
  editingCurrency,
}: CurrencyFormDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(editingCurrency);

  const form = useForm<CurrencyFormValues>({
    resolver: zodResolver(currencyFormSchema),
    defaultValues: {
      code: '',
      numericCode: '',
      name: '',
      precision: 2,
      isActive: true,
    },
  });

  useEffect(() => {
    if (!open) return;
    if (editingCurrency) {
      form.reset({
        code: editingCurrency.code,
        numericCode: editingCurrency.numericCode ?? '',
        name: editingCurrency.name,
        precision: editingCurrency.precision,
        isActive: editingCurrency.isActive,
      });
    } else {
      form.reset({
        code: '',
        numericCode: '',
        name: '',
        precision: 2,
        isActive: true,
      });
    }
  }, [open, editingCurrency, form]);

  const mutation = useMutation({
    mutationFn: async (values: CurrencyFormValues) => {
      if (editingCurrency) {
        return updateCurrency(editingCurrency.id, editingCurrency.rowVersion, {
          code: values.code,
          numericCode: values.numericCode,
          name: values.name,
          precision: values.precision,
          isActive: values.isActive,
        });
      }
      return createCurrency(values);
    },
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: currencyQueryKeys.currenciesAll });
      toast.success(
        isEdit
          ? `Currency ${saved.code} updated`
          : `Currency ${saved.code} (${saved.name}) added`,
      );
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg =
        err instanceof ApiClientError
          ? err.message
          : 'Unable to save currency (global system currencies are read-only)';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="md"
      title={isEdit ? `Edit Currency · ${editingCurrency?.code}` : 'Add Currency'}
      description="Configure ISO-4217 currency precision and active ledger status."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="currency-form" isLoading={mutation.isPending}>
            {isEdit ? 'Save Currency' : 'Create Currency'}
          </Button>
        </>
      }
    >
      <form
        id="currency-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <FormField
            label="ISO Code"
            htmlFor="curr-code"
            required
            error={form.formState.errors.code?.message}
          >
            <Input
              id="curr-code"
              maxLength={3}
              placeholder="THB"
              className="uppercase font-mono"
              disabled={isEdit}
              error={Boolean(form.formState.errors.code)}
              {...form.register('code')}
            />
          </FormField>

          <FormField
            label="Numeric Code"
            htmlFor="curr-numeric-code"
            error={form.formState.errors.numericCode?.message}
          >
            <Input
              id="curr-numeric-code"
              maxLength={3}
              numeric
              placeholder="764"
              className="font-mono"
              error={Boolean(form.formState.errors.numericCode)}
              {...form.register('numericCode')}
            />
          </FormField>

          <FormField
            label="Decimal Precision"
            htmlFor="curr-precision"
            required
            error={form.formState.errors.precision?.message}
          >
            <Controller
              control={form.control}
              name="precision"
              render={({ field }) => (
                <Select
                  id="curr-precision"
                  value={String(field.value)}
                  onValueChange={(val) => field.onChange(Number.parseInt(val, 10))}
                  options={[
                    { value: '0', label: '0 (JPY / Zero-decimal)' },
                    { value: '2', label: '2 (Standard)' },
                    { value: '3', label: '3 (BHD / KWD)' },
                    { value: '4', label: '4 (High precision)' },
                  ]}
                  error={Boolean(form.formState.errors.precision)}
                />
              )}
            />
          </FormField>
        </div>

        <FormField
          label="Currency Name"
          htmlFor="curr-name"
          required
          error={form.formState.errors.name?.message}
        >
          <Input
            id="curr-name"
            placeholder="Thai Baht"
            error={Boolean(form.formState.errors.name)}
            {...form.register('name')}
          />
        </FormField>

        <Controller
          control={form.control}
          name="isActive"
          render={({ field }) => (
            <Switch
              checked={field.value}
              onCheckedChange={field.onChange}
              label="Active for transactions and FX triangulation"
            />
          )}
        />
      </form>
    </AppDialog>
  );
}

interface RateTypeFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RateTypeFormDialog({ open, onOpenChange }: RateTypeFormDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<RateTypeFormValues>({
    resolver: zodResolver(rateTypeFormSchema),
    defaultValues: {
      code: '',
      name: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    form.reset({ code: '', name: '' });
  }, [open, form]);

  const mutation = useMutation({
    mutationFn: (values: RateTypeFormValues) => createExchangeRateType(values),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: currencyQueryKeys.rateTypesAll });
      toast.success(`Exchange rate type ${created.code} created`);
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to create rate type';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="sm"
      title="New Exchange Rate Type"
      description="Define a custom FX valuation tier (e.g. TAX_DJP, BUDGET, CONTRACT)."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="rate-type-form" isLoading={mutation.isPending}>
            Create Rate Type
          </Button>
        </>
      }
    >
      <form
        id="rate-type-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <FormField
          label="Code"
          htmlFor="rt-code"
          required
          error={form.formState.errors.code?.message}
        >
          <Input
            id="rt-code"
            placeholder="TAX_DJP"
            className="uppercase font-mono"
            error={Boolean(form.formState.errors.code)}
            {...form.register('code')}
          />
        </FormField>

        <FormField
          label="Display Name"
          htmlFor="rt-name"
          required
          error={form.formState.errors.name?.message}
        >
          <Input
            id="rt-name"
            placeholder="Kurs Pajak DJP Mingguan"
            error={Boolean(form.formState.errors.name)}
            {...form.register('name')}
          />
        </FormField>
      </form>
    </AppDialog>
  );
}

interface ExchangeRateFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingRate: ExchangeRateDto | null;
  currencies: CurrencyDto[];
  rateTypes: ExchangeRateTypeDto[];
}

const DEFAULT_CURRENCY_LIST = ['USD', 'IDR', 'EUR', 'SGD', 'JPY', 'GBP'];

export function ExchangeRateFormDialog({
  open,
  onOpenChange,
  editingRate,
  currencies,
  rateTypes,
}: ExchangeRateFormDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(editingRate);

  const currencyOptions = useMemo(() => {
    const active = currencies.filter((c) => c.isActive);
    if (active.length > 0) {
      return active.map((c) => ({ value: c.code, label: `${c.code} — ${c.name}` }));
    }
    return DEFAULT_CURRENCY_LIST.map((code) => ({ value: code, label: code }));
  }, [currencies]);

  const rateTypeOptions = useMemo(
    () =>
      rateTypes.length > 0
        ? rateTypes.map((rt) => ({ value: rt.id, label: `${rt.code} — ${rt.name}` }))
        : [
            { value: 'SPOT', label: 'SPOT — Spot Rate' },
            { value: 'CLOSING', label: 'CLOSING — Period Closing' },
            { value: 'AVERAGE', label: 'AVERAGE — Period Average' },
          ],
    [rateTypes],
  );

  const form = useForm<ExchangeRateFormValues>({
    resolver: zodResolver(exchangeRateFormSchema),
    defaultValues: {
      fromCurrency: 'USD',
      toCurrency: 'IDR',
      rateTypeId: '',
      rate: '15800.000000',
      validFrom: todayIso(),
      validTo: '',
    },
  });

  useEffect(() => {
    if (!open) return;
    if (editingRate) {
      form.reset({
        fromCurrency: editingRate.fromCurrency,
        toCurrency: editingRate.toCurrency,
        rateTypeId: editingRate.rateTypeId || editingRate.rateTypeCode,
        rate: editingRate.rate,
        validFrom: editingRate.validFrom.slice(0, 10),
        validTo: editingRate.validTo ? editingRate.validTo.slice(0, 10) : '',
      });
    } else {
      form.reset({
        fromCurrency: 'USD',
        toCurrency: 'IDR',
        rateTypeId: rateTypes[0]?.id ?? 'SPOT',
        rate: '15800.000000',
        validFrom: todayIso(),
        validTo: '',
      });
    }
  }, [open, editingRate, rateTypes, form]);

  const mutation = useMutation({
    mutationFn: async (values: ExchangeRateFormValues) => {
      if (editingRate) {
        return updateExchangeRate(editingRate.id, editingRate.rowVersion, {
          rate: values.rate,
          validTo: values.validTo || undefined,
        });
      }
      return createExchangeRate({
        fromCurrency: values.fromCurrency,
        toCurrency: values.toCurrency,
        rateTypeId: values.rateTypeId,
        rate: values.rate,
        validFrom: values.validFrom,
        validTo: values.validTo || undefined,
      });
    },
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: currencyQueryKeys.ratesAll });
      toast.success(
        isEdit
          ? `FX quote ${saved.fromCurrency}/${saved.toCurrency} updated`
          : `FX quote ${saved.fromCurrency}/${saved.toCurrency} published`,
      );
      onOpenChange(false);
    },
    onError: (err) => {
      applyApiErrorsToForm(err, form.setError);
      const msg = err instanceof ApiClientError ? err.message : 'Unable to save exchange rate';
      toast.error(msg);
    },
  });

  return (
    <AppDialog
      open={open}
      onOpenChange={onOpenChange}
      size="lg"
      title={
        isEdit
          ? `Edit FX Quote · ${editingRate?.fromCurrency}/${editingRate?.toCurrency}`
          : 'Publish Exchange Rate'
      }
      description="Rates support up to 6-decimal precision and effective date windows."
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" form="fx-rate-form" isLoading={mutation.isPending}>
            {isEdit ? 'Update FX Rate' : 'Publish FX Rate'}
          </Button>
        </>
      }
    >
      <form
        id="fx-rate-form"
        className="space-y-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        noValidate
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <FormField
            label="From Currency"
            htmlFor="fx-from"
            required
            error={form.formState.errors.fromCurrency?.message}
          >
            <Controller
              control={form.control}
              name="fromCurrency"
              render={({ field }) => (
                <Select
                  id="fx-from"
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={isEdit}
                  options={currencyOptions}
                  error={Boolean(form.formState.errors.fromCurrency)}
                />
              )}
            />
          </FormField>

          <FormField
            label="To Currency"
            htmlFor="fx-to"
            required
            error={form.formState.errors.toCurrency?.message}
          >
            <Controller
              control={form.control}
              name="toCurrency"
              render={({ field }) => (
                <Select
                  id="fx-to"
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={isEdit}
                  options={currencyOptions}
                  error={Boolean(form.formState.errors.toCurrency)}
                />
              )}
            />
          </FormField>

          <FormField
            label="Rate Type"
            htmlFor="fx-type"
            required
            error={form.formState.errors.rateTypeId?.message}
          >
            <Controller
              control={form.control}
              name="rateTypeId"
              render={({ field }) => (
                <Select
                  id="fx-type"
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={isEdit}
                  options={rateTypeOptions}
                  error={Boolean(form.formState.errors.rateTypeId)}
                />
              )}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <FormField
            label="Spot / Conversion Rate"
            htmlFor="fx-rate"
            required
            error={form.formState.errors.rate?.message}
          >
            <Input
              id="fx-rate"
              numeric
              placeholder="15850.000000"
              className="font-mono"
              error={Boolean(form.formState.errors.rate)}
              {...form.register('rate')}
            />
          </FormField>

          <FormField
            label="Valid From"
            htmlFor="fx-valid-from"
            required
            error={form.formState.errors.validFrom?.message}
          >
            <Input
              id="fx-valid-from"
              type="date"
              disabled={isEdit}
              error={Boolean(form.formState.errors.validFrom)}
              {...form.register('validFrom')}
            />
          </FormField>

          <FormField
            label="Valid To (Optional)"
            htmlFor="fx-valid-to"
            error={form.formState.errors.validTo?.message}
          >
            <Input
              id="fx-valid-to"
              type="date"
              error={Boolean(form.formState.errors.validTo)}
              {...form.register('validTo')}
            />
          </FormField>
        </div>
      </form>
    </AppDialog>
  );
}
