import * as React from 'react';
import { Handshake, Headset, Plus, Target } from 'lucide-react';
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
import { addDecimals, formatCurrency } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useCloseOpportunityMutation,
  useCreateLeadMutation,
  useCustomerSegmentsQuery,
  useLeadsQuery,
  useOpportunitiesQuery,
  useSupportCasesQuery,
  useTransitionLeadMutation,
  useTransitionSupportCaseMutation,
  type CustomerSegmentDto,
  type LeadDto,
  type OpportunityDto,
  type SupportCaseDto,
} from './crm-api';

export function CrmWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const currency = activeOrg?.functionalCurrency ?? 'IDR';
  const canWrite = hasUserPermission(user, ['crm:write', 'crm.create']);

  const [tab, setTab] = React.useState('leads');
  const [leadStatus, setLeadStatus] = React.useState('ALL');
  const [createLeadOpen, setCreateLeadOpen] = React.useState(false);

  const [code, setCode] = React.useState('');
  const [name, setName] = React.useState('');
  const [source, setSource] = React.useState('Referral');
  const [estValue, setEstValue] = React.useState('250000000');

  const leadsQ = useLeadsQuery(orgId, leadStatus);
  const oppsQ = useOpportunitiesQuery(orgId);
  const segsQ = useCustomerSegmentsQuery(orgId);
  const casesQ = useSupportCasesQuery(orgId);

  const createLead = useCreateLeadMutation();
  const transitionLead = useTransitionLeadMutation();
  const closeOpp = useCloseOpportunityMutation();
  const transitionCase = useTransitionSupportCaseMutation();

  const leads = leadsQ.data?.items ?? [];
  const opportunities = oppsQ.data?.items ?? [];
  const segments = segsQ.data?.items ?? [];
  const cases = casesQ.data?.items ?? [];

  const pipelineValue = React.useMemo(
    () =>
      opportunities
        .filter((o) => o.status === 'Open')
        .reduce((acc, o) => addDecimals(acc, o.estimatedValue ?? '0', 4), '0'),
    [opportunities]
  );

  const leadCols: ColumnDef<LeadDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '130px', className: 'font-mono font-semibold' },
    { header: 'Lead Name', accessorKey: 'name' },
    { header: 'Source', width: '120px', cell: (r) => r.source ?? '—' },
    {
      header: 'Est. Value',
      align: 'right',
      width: '150px',
      cell: (r) => (
        <span className="font-mono tabular-nums">
          {r.estimatedValue ? formatCurrency(r.estimatedValue, currency, locale) : '—'}
        </span>
      ),
    },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '140px',
      cell: (r) => {
        if (!canWrite) return null;
        if (r.status === 'New') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionLead.mutateAsync({ id: r.id, action: 'qualify', rowVersion: r.rowVersion }).then(
                  () => toast.success(`Lead ${r.code} qualified`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Qualify
            </Button>
          );
        }
        if (r.status === 'Qualified') {
          return (
            <Button
              size="sm"
              variant="primary"
              onClick={() =>
                void transitionLead.mutateAsync({ id: r.id, action: 'convert', rowVersion: r.rowVersion }).then(
                  () => toast.success(`Lead ${r.code} converted to Opportunity`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Convert
            </Button>
          );
        }
        return null;
      },
    },
  ];

  const oppCols: ColumnDef<OpportunityDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '130px', className: 'font-mono font-semibold' },
    { header: 'Stage', accessorKey: 'stage', width: '140px' },
    { header: 'Prob %', width: '90px', align: 'right', cell: (r) => <span className="font-mono tabular-nums">{r.probabilityPct ?? '0'}%</span> },
    {
      header: 'Value',
      align: 'right',
      width: '150px',
      cell: (r) => (
        <span className="font-mono tabular-nums">
          {r.estimatedValue ? formatCurrency(r.estimatedValue, currency, locale) : '—'}
        </span>
      ),
    },
    { header: 'Status', width: '110px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '140px',
      cell: (r) =>
        canWrite && r.status === 'Open' ? (
          <div className="flex gap-1">
            <Button
              size="sm"
              variant="primary"
              onClick={() =>
                void closeOpp.mutateAsync({ id: r.id, action: 'win', rowVersion: r.rowVersion, currencyCode: currency }).then(
                  () => toast.success(`Opportunity ${r.code} marked Won`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Win
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void closeOpp.mutateAsync({ id: r.id, action: 'lose', rowVersion: r.rowVersion }).then(
                  () => toast.info(`Opportunity ${r.code} marked Lost`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Lose
            </Button>
          </div>
        ) : null,
    },
  ];

  const segCols: ColumnDef<CustomerSegmentDto>[] = [
    { header: 'Code', accessorKey: 'code', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Segment Name', accessorKey: 'name' },
    { header: 'Scope', width: '110px', cell: (r) => <StatusBadge status={r.isSystem ? 'ISSUED' : 'ACTIVE'} label={r.isSystem ? 'System' : 'Tenant'} /> },
  ];

  const caseCols: ColumnDef<SupportCaseDto>[] = [
    { header: 'Case No', accessorKey: 'caseNo', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Subject', accessorKey: 'subject' },
    { header: 'Status', width: '120px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Action',
      width: '130px',
      cell: (r) => {
        if (!canWrite) return null;
        if (r.status === 'Open' || r.status === 'Escalated') {
          return (
            <Button
              size="sm"
              variant="primary"
              onClick={() =>
                void transitionCase.mutateAsync({ id: r.id, action: 'resolve', rowVersion: r.rowVersion }).then(
                  () => toast.success(`Case ${r.caseNo} resolved`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Resolve
            </Button>
          );
        }
        if (r.status === 'Resolved') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionCase.mutateAsync({ id: r.id, action: 'close', rowVersion: r.rowVersion }).then(
                  () => toast.success(`Case ${r.caseNo} closed`),
                  (err) => toast.error(err instanceof Error ? err.message : 'Failed')
                )
              }
            >
              Close
            </Button>
          );
        }
        return null;
      },
    },
  ];

  const handleCreateLead = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createLead.mutateAsync({
        code: code.trim(),
        name: name.trim(),
        source: source.trim() || undefined,
        estimatedValue: estValue.trim() || undefined,
      });
      toast.success('Lead created');
      setCreateLeadOpen(false);
      setCode('');
      setName('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to create lead');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Commercial' }, { label: 'CRM & Pipeline' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            CRM Leads, Opportunities & Support Cases
          </h1>
        </div>
        {canWrite ? (
          <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateLeadOpen(true)}>
            New Lead
          </Button>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Active Leads" value={leads.length} icon={<Target className="h-4 w-4" />} accent="brand" />
        <KpiCard
          label="Open Pipeline Value"
          value={<span className="font-mono tabular-nums">{formatCurrency(pipelineValue, currency, locale)}</span>}
          icon={<Handshake className="h-4 w-4" />}
          accent="emerald"
        />
        <KpiCard label="Open Support Cases" value={cases.filter((c) => c.status !== 'Closed').length} icon={<Headset className="h-4 w-4" />} accent="amber" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="leads">Leads ({leads.length})</TabsTrigger>
          <TabsTrigger value="opportunities">Opportunities ({opportunities.length})</TabsTrigger>
          <TabsTrigger value="segments">Segments ({segments.length})</TabsTrigger>
          <TabsTrigger value="cases">Support Cases ({cases.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="leads" className="pt-4">
          <VirtualizedTable
            data={leads}
            columns={leadCols}
            isLoading={leadsQ.isLoading}
            error={leadsQ.error}
            onRetry={() => void leadsQ.refetch()}
            emptyTitle="No leads yet"
            toolbar={{
              filters: (
                <Select
                  value={leadStatus}
                  onValueChange={setLeadStatus}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'New', label: 'New' },
                    { value: 'Qualified', label: 'Qualified' },
                    { value: 'Converted', label: 'Converted' },
                    { value: 'Lost', label: 'Lost' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="opportunities" className="pt-4">
          <VirtualizedTable data={opportunities} columns={oppCols} isLoading={oppsQ.isLoading} error={oppsQ.error} onRetry={() => void oppsQ.refetch()} emptyTitle="No opportunities" />
        </TabsContent>
        <TabsContent value="segments" className="pt-4">
          <VirtualizedTable data={segments} columns={segCols} isLoading={segsQ.isLoading} error={segsQ.error} onRetry={() => void segsQ.refetch()} emptyTitle="No customer segments" />
        </TabsContent>
        <TabsContent value="cases" className="pt-4">
          <VirtualizedTable data={cases} columns={caseCols} isLoading={casesQ.isLoading} error={casesQ.error} onRetry={() => void casesQ.refetch()} emptyTitle="No support cases" />
        </TabsContent>
      </Tabs>

      <AppDialog open={createLeadOpen} onOpenChange={setCreateLeadOpen} title="Create Sales Lead" size="md">
        <form onSubmit={handleCreateLead} className="space-y-4">
          <FormField label="Lead Code" htmlFor="leadCode" required>
            <Input id="leadCode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="LD-2026-001" required />
          </FormField>
          <FormField label="Lead Name" htmlFor="leadName" required>
            <Input id="leadName" value={name} onChange={(e) => setName(e.target.value)} placeholder="PT Karya Mandiri" required />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Source" htmlFor="leadSrc">
              <Input id="leadSrc" value={source} onChange={(e) => setSource(e.target.value)} />
            </FormField>
            <FormField label="Estimated Value" htmlFor="leadVal">
              <Input id="leadVal" value={estValue} onChange={(e) => setEstValue(e.target.value)} />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateLeadOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createLead.isPending}>Create Lead</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
