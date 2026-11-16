import { useMemo } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { ArrowLeftRight, Calculator, Sparkles } from 'lucide-react';
import { Button } from '../../components/ui/Button';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { MoneyInput } from '../../components/ui/MoneyInput';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { formatCurrency, formatDate, formatRate } from '../../lib/utils';
import { useUiStore } from '../../store/ui-store';
import { convertCurrency } from './api';
import { convertFxFormSchema, type ConvertFxFormValues } from './schemas';
import type { CurrencyDto, ExchangeRateTypeDto } from './types';

interface FxConverterCardProps {
  currencies: CurrencyDto[];
  rateTypes: ExchangeRateTypeDto[];
}

const DEFAULT_CURRENCY_LIST = ['USD', 'IDR', 'EUR', 'SGD', 'JPY', 'GBP'];
const DEFAULT_RATE_TYPES = ['SPOT', 'CLOSING', 'AVERAGE'];

function rateSourceTone(
  source: string,
): 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'brand' {
  switch (source.toUpperCase()) {
    case 'DIRECT':
      return 'success';
    case 'INVERSE':
      return 'info';
    case 'TRIANGULATED':
      return 'brand';
    default:
      return 'neutral';
  }
}

export function FxConverterCard({ currencies, rateTypes }: FxConverterCardProps) {
  const locale = useUiStore((s) => s.locale);
  const numberLocale = locale === 'id' ? 'id-ID' : 'en-US';

  const currencyOptions = useMemo(() => {
    const active = currencies.filter((c) => c.isActive);
    if (active.length > 0) {
      return active.map((c) => ({ value: c.code, label: `${c.code} — ${c.name}` }));
    }
    return DEFAULT_CURRENCY_LIST.map((code) => ({ value: code, label: code }));
  }, [currencies]);

  const rateTypeOptions = useMemo(() => {
    if (rateTypes.length > 0) {
      return rateTypes.map((rt) => ({ value: rt.code, label: `${rt.code} — ${rt.name}` }));
    }
    return DEFAULT_RATE_TYPES.map((code) => ({ value: code, label: code }));
  }, [rateTypes]);

  const form = useForm<ConvertFxFormValues>({
    resolver: zodResolver(convertFxFormSchema),
    defaultValues: {
      fromCurrency: 'USD',
      toCurrency: 'IDR',
      rateTypeCode: 'SPOT',
      amount: '1000',
      onDate: new Date().toISOString().slice(0, 10),
      anchor: 'AUTO',
    },
  });

  const convertMutation = useMutation({
    mutationFn: (values: ConvertFxFormValues) =>
      convertCurrency({
        fromCurrency: values.fromCurrency,
        toCurrency: values.toCurrency,
        rateTypeCode: values.rateTypeCode,
        amount: values.amount,
        onDate: values.onDate,
        anchor: values.anchor && values.anchor !== 'AUTO' ? values.anchor : undefined,
      }),
  });

  const watchedFrom = form.watch('fromCurrency');
  const watchedTo = form.watch('toCurrency');

  const handleSwap = () => {
    form.setValue('fromCurrency', watchedTo, { shouldDirty: true });
    form.setValue('toCurrency', watchedFrom, { shouldDirty: true });
  };

  return (
    <section
      aria-label="FX Triangulation Converter"
      className="rounded-2xl border border-slate-200 bg-white p-5 shadow-xs"
    >
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-brand-600" />
            <h2 className="text-base font-semibold text-slate-900">
              FX Triangulation Converter
            </h2>
          </div>
          <p className="mt-0.5 text-xs text-slate-500">
            Resolves Direct, Inverse, or Anchor-Triangulated rates cleanly via server precision math.
          </p>
        </div>
      </div>

      <form
        className="grid grid-cols-1 gap-4 lg:grid-cols-12"
        onSubmit={form.handleSubmit((values) => convertMutation.mutate(values))}
        noValidate
      >
        <div className="lg:col-span-3">
          <FormField
            label="Original Amount"
            htmlFor="fx-conv-amount"
            required
            error={form.formState.errors.amount?.message}
          >
            <Controller
              control={form.control}
              name="amount"
              render={({ field }) => (
                <MoneyInput
                  id="fx-conv-amount"
                  value={field.value}
                  onValueChange={field.onChange}
                  currency={watchedFrom}
                  error={Boolean(form.formState.errors.amount)}
                />
              )}
            />
          </FormField>
        </div>

        <div className="lg:col-span-2">
          <FormField
            label="From"
            htmlFor="fx-conv-from"
            required
            error={form.formState.errors.fromCurrency?.message}
          >
            <Controller
              control={form.control}
              name="fromCurrency"
              render={({ field }) => (
                <Select
                  id="fx-conv-from"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={currencyOptions}
                />
              )}
            />
          </FormField>
        </div>

        <div className="flex items-end justify-center pb-0.5 lg:col-span-1">
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="Swap currencies"
            onClick={handleSwap}
          >
            <ArrowLeftRight className="h-4 w-4" />
          </Button>
        </div>

        <div className="lg:col-span-2">
          <FormField
            label="To"
            htmlFor="fx-conv-to"
            required
            error={form.formState.errors.toCurrency?.message}
          >
            <Controller
              control={form.control}
              name="toCurrency"
              render={({ field }) => (
                <Select
                  id="fx-conv-to"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={currencyOptions}
                />
              )}
            />
          </FormField>
        </div>

        <div className="lg:col-span-2">
          <FormField label="Rate Type" htmlFor="fx-conv-rate-type" required>
            <Controller
              control={form.control}
              name="rateTypeCode"
              render={({ field }) => (
                <Select
                  id="fx-conv-rate-type"
                  value={field.value}
                  onValueChange={field.onChange}
                  options={rateTypeOptions}
                />
              )}
            />
          </FormField>
        </div>

        <div className="lg:col-span-2">
          <FormField label="On Date" htmlFor="fx-conv-date" required>
            <Input id="fx-conv-date" type="date" {...form.register('onDate')} />
          </FormField>
        </div>

        <div className="flex flex-wrap items-end justify-between gap-3 lg:col-span-12">
          <div className="w-56">
            <FormField label="Anchor Override (Optional)" htmlFor="fx-conv-anchor">
              <Controller
                control={form.control}
                name="anchor"
                render={({ field }) => (
                  <Select
                    id="fx-conv-anchor"
                    value={field.value || 'AUTO'}
                    onValueChange={field.onChange}
                    options={[
                      { value: 'AUTO', label: 'Auto (Default Anchor)' },
                      ...currencyOptions,
                    ]}
                  />
                )}
              />
            </FormField>
          </div>

          <Button
            type="submit"
            leftIcon={<Calculator className="h-4 w-4" />}
            isLoading={convertMutation.isPending}
          >
            Calculate Conversion
          </Button>
        </div>
      </form>

      {convertMutation.isError && (
        <div className="mt-4">
          <ErrorState
            error={convertMutation.error}
            title="FX Rate Resolution Failed"
            onRetry={() => form.handleSubmit((values) => convertMutation.mutate(values))()}
          />
        </div>
      )}

      {convertMutation.data && (
        <div className="mt-4 rounded-xl border border-brand-200 bg-brand-50/40 p-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                Converted Equivalent ({convertMutation.data.fromCurrency} →{' '}
                {convertMutation.data.toCurrency})
              </p>
              <p className="mt-1 font-mono text-2xl font-bold text-slate-900 tabular-nums">
                {formatCurrency(
                  convertMutation.data.convertedAmount,
                  convertMutation.data.toCurrency,
                  numberLocale,
                )}
              </p>
              <p className="mt-0.5 font-mono text-xs text-slate-600">
                Raw: {convertMutation.data.convertedAmount} {convertMutation.data.toCurrency}
              </p>
            </div>

            <div className="flex flex-col items-end gap-1.5 text-right">
              <StatusBadge
                status={convertMutation.data.rateSource}
                label={
                  convertMutation.data.anchorCurrency
                    ? `${convertMutation.data.rateSource} · Anchor ${convertMutation.data.anchorCurrency}`
                    : convertMutation.data.rateSource
                }
                tone={rateSourceTone(convertMutation.data.rateSource)}
              />
              <p className="font-mono text-xs font-semibold text-slate-800 tabular-nums">
                1 {convertMutation.data.fromCurrency} ={' '}
                {formatRate(convertMutation.data.rate, numberLocale)}{' '}
                {convertMutation.data.toCurrency}
              </p>
              <p className="text-[11px] text-slate-500">
                Effective {formatDate(convertMutation.data.effectiveDate, numberLocale)}
              </p>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
