import * as React from 'react';
import { Globe, Languages, Plus } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import {
  useCreateLanguageMutation,
  useCreateTranslationMutation,
  useLanguagesQuery,
  useLocaleFormatsQuery,
  useTranslationsQuery,
  type LanguageDto,
  type LocaleFormatDto,
  type TranslationDto,
} from './localization-api';

export function LocalizationWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, ['localization:write']);

  const [tab, setTab] = React.useState('languages');
  const [langFilter, setLangFilter] = React.useState('ALL');
  const [createLangOpen, setCreateLangOpen] = React.useState(false);
  const [createTrOpen, setCreateTrOpen] = React.useState(false);

  const [langCode, setLangCode] = React.useState('');
  const [langName, setLangName] = React.useState('');
  const [trLang, setTrLang] = React.useState('id');
  const [trKey, setTrKey] = React.useState('');
  const [trVal, setTrVal] = React.useState('');
  const [trModule, setTrModule] = React.useState('common');

  const langsQ = useLanguagesQuery(orgId);
  const trsQ = useTranslationsQuery(orgId, langFilter);
  const formatsQ = useLocaleFormatsQuery(orgId);

  const createLang = useCreateLanguageMutation();
  const createTr = useCreateTranslationMutation();

  const languages = langsQ.data?.items ?? [];
  const translations = trsQ.data?.items ?? [];
  const formats = formatsQ.data?.items ?? [];

  const langCols: ColumnDef<LanguageDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '100px', className: 'font-mono font-semibold' },
    { header: 'Language Name', accessorKey: 'name' },
    { header: 'Fallback', width: '100px', cell: (r) => (r.isFallback ? 'Default' : '—') },
    { header: 'Status', width: '110px', cell: (r) => <StatusBadge status={r.isActive ? 'ACTIVE' : 'INACTIVE'} label={r.isActive ? 'Active' : 'Inactive'} /> },
    { header: 'Scope', width: '100px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const trCols: ColumnDef<TranslationDto>[] = [
    { header: 'Lang', accessorKey: 'languageCode', width: '80px', className: 'font-mono font-semibold' },
    { header: 'Key', accessorKey: 'resourceKey', width: '240px', className: 'font-mono text-xs' },
    { header: 'Value', accessorKey: 'resourceValue' },
    { header: 'Module', width: '120px', cell: (r) => r.module ?? '—' },
  ];

  const formatCols: ColumnDef<LocaleFormatDto>[] = [
    { header: 'Locale', accessorKey: 'localeCode', width: '120px', className: 'font-mono font-semibold' },
    { header: 'Date Format', accessorKey: 'dateFormat', width: '140px', className: 'font-mono' },
    { header: 'Decimal / Thousand', width: '160px', cell: (r) => <span className="font-mono">"{r.decimalSeparator}" / "{r.thousandSeparator}"</span> },
    { header: 'Scope', width: '100px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const handleCreateLang = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createLang.mutateAsync({ code: langCode.trim().toLowerCase(), name: langName.trim(), isActive: true });
      toast.success('Language added');
      setCreateLangOpen(false);
      setLangCode('');
      setLangName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to add language');
    }
  };

  const handleCreateTr = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createTr.mutateAsync({
        languageCode: trLang,
        resourceKey: trKey.trim(),
        resourceValue: trVal.trim(),
        module: trModule.trim() || undefined,
      });
      toast.success('Translation key saved');
      setCreateTrOpen(false);
      setTrKey('');
      setTrVal('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save translation');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Platform Admin' }, { label: 'Localization' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Languages, Translations & Locale Formats
          </h1>
        </div>
        {canWrite ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateLangOpen(true)}>
              New Language
            </Button>
            <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateTrOpen(true)}>
              New Translation
            </Button>
          </div>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Languages" value={languages.length} icon={<Globe className="h-4 w-4" />} accent="brand" />
        <KpiCard label="Translation Keys" value={translations.length} icon={<Languages className="h-4 w-4" />} accent="emerald" />
        <KpiCard label="Locale Formats" value={formats.length} icon={<Globe className="h-4 w-4" />} accent="sky" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="languages">Languages ({languages.length})</TabsTrigger>
          <TabsTrigger value="translations">Translations ({translations.length})</TabsTrigger>
          <TabsTrigger value="formats">Locale Formats ({formats.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="languages" className="pt-4">
          <VirtualizedTable data={languages} columns={langCols} isLoading={langsQ.isLoading} error={langsQ.error} onRetry={() => void langsQ.refetch()} emptyTitle="No languages configured" />
        </TabsContent>
        <TabsContent value="translations" className="pt-4">
          <VirtualizedTable
            data={translations}
            columns={trCols}
            isLoading={trsQ.isLoading}
            error={trsQ.error}
            onRetry={() => void trsQ.refetch()}
            emptyTitle="No translation entries"
            toolbar={{
              filters: (
                <Select
                  value={langFilter}
                  onValueChange={setLangFilter}
                  options={[
                    { value: 'ALL', label: 'All Languages' },
                    ...languages.map((l) => ({ value: l.code, label: `${l.code} · ${l.name}` })),
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="formats" className="pt-4">
          <VirtualizedTable data={formats} columns={formatCols} isLoading={formatsQ.isLoading} error={formatsQ.error} onRetry={() => void formatsQ.refetch()} emptyTitle="No locale formats" />
        </TabsContent>
      </Tabs>

      <AppDialog open={createLangOpen} onOpenChange={setCreateLangOpen} title="Add Language" size="sm">
        <form onSubmit={handleCreateLang} className="space-y-4">
          <FormField label="ISO Code" htmlFor="locCode" required>
            <Input id="locCode" value={langCode} onChange={(e) => setLangCode(e.target.value)} placeholder="ja" required />
          </FormField>
          <FormField label="Language Name" htmlFor="locName" required>
            <Input id="locName" value={langName} onChange={(e) => setLangName(e.target.value)} placeholder="Japanese" required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateLangOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createLang.isPending}>Add</Button>
          </div>
        </form>
      </AppDialog>

      <AppDialog open={createTrOpen} onOpenChange={setCreateTrOpen} title="Add Translation Key" size="md">
        <form onSubmit={handleCreateTr} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Language Code" htmlFor="trLang" required>
              <Input id="trLang" value={trLang} onChange={(e) => setTrLang(e.target.value)} required />
            </FormField>
            <FormField label="Module" htmlFor="trMod">
              <Input id="trMod" value={trModule} onChange={(e) => setTrModule(e.target.value)} />
            </FormField>
          </div>
          <FormField label="Resource Key" htmlFor="trKey" required>
            <Input id="trKey" value={trKey} onChange={(e) => setTrKey(e.target.value)} placeholder="invoice.status.paid" required />
          </FormField>
          <FormField label="Translated Value" htmlFor="trVal" required>
            <Input id="trVal" value={trVal} onChange={(e) => setTrVal(e.target.value)} placeholder="Lunas" required />
          </FormField>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateTrOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createTr.isPending}>Save</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
