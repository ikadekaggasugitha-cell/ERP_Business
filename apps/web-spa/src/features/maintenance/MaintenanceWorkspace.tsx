import * as React from 'react';
import { Calendar, Plus, Wrench } from 'lucide-react';
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
import { formatDate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useCreateMaintenanceWorkOrderMutation,
  useMaintenanceSchedulesQuery,
  useMaintenanceWorkOrdersQuery,
  useTransitionWorkOrderMutation,
  type MaintenanceScheduleDto,
  type MaintenanceWorkOrderDto,
} from './maintenance-api';

const ZERO_UUID = '00000000-0000-0000-0000-000000000001';

export function MaintenanceWorkspace() {
  const { user, activeOrg } = useAuthStore();
  const { locale } = useUiStore();
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, ['maintenance:write', 'maintenance.create']);

  const [tab, setTab] = React.useState('work-orders');
  const [statusFilter, setStatusFilter] = React.useState('ALL');
  const [createOpen, setCreateOpen] = React.useState(false);

  const [woNo, setWoNo] = React.useState('');
  const [assetRef, setAssetRef] = React.useState(ZERO_UUID);
  const [maintenanceType, setMaintenanceType] = React.useState('Preventive');
  const [scheduledDate, setScheduledDate] = React.useState(new Date().toISOString().slice(0, 10));

  const woQ = useMaintenanceWorkOrdersQuery(orgId, statusFilter);
  const schedulesQ = useMaintenanceSchedulesQuery(orgId);
  const createWo = useCreateMaintenanceWorkOrderMutation();
  const transitionWo = useTransitionWorkOrderMutation();

  const workOrders = woQ.data?.items ?? [];
  const schedules = schedulesQ.data?.items ?? [];

  const woCols: ColumnDef<MaintenanceWorkOrderDto>[] = [
    { header: 'WO No', accessorKey: 'woNo', width: '140px', className: 'font-mono font-semibold' },
    { header: 'Type', accessorKey: 'maintenanceType', width: '130px' },
    { header: 'Scheduled Date', width: '140px', cell: (r) => (r.scheduledDate ? formatDate(r.scheduledDate, locale) : '—') },
    { header: 'Status', width: '130px', cell: (r) => <StatusBadge status={r.status.toUpperCase()} label={r.status} /> },
    {
      header: 'Workflow',
      width: '140px',
      cell: (r) => {
        if (!canWrite) return null;
        if (r.status === 'Open') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionWo
                  .mutateAsync({ id: r.id, action: 'assign', rowVersion: r.rowVersion, body: { assigneeRef: user?.id ?? ZERO_UUID } })
                  .then(() => toast.success(`WO ${r.woNo} assigned`), (err) => toast.error(err instanceof Error ? err.message : 'Failed'))
              }
            >
              Assign
            </Button>
          );
        }
        if (r.status === 'Assigned') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionWo
                  .mutateAsync({ id: r.id, action: 'start', rowVersion: r.rowVersion })
                  .then(() => toast.success(`WO ${r.woNo} started`), (err) => toast.error(err instanceof Error ? err.message : 'Failed'))
              }
            >
              Start
            </Button>
          );
        }
        if (r.status === 'InProgress') {
          return (
            <Button
              size="sm"
              variant="primary"
              onClick={() =>
                void transitionWo
                  .mutateAsync({ id: r.id, action: 'complete', rowVersion: r.rowVersion })
                  .then(() => toast.success(`WO ${r.woNo} completed`), (err) => toast.error(err instanceof Error ? err.message : 'Failed'))
              }
            >
              Complete
            </Button>
          );
        }
        if (r.status === 'Completed') {
          return (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                void transitionWo
                  .mutateAsync({ id: r.id, action: 'close', rowVersion: r.rowVersion })
                  .then(() => toast.success(`WO ${r.woNo} closed`), (err) => toast.error(err instanceof Error ? err.message : 'Failed'))
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

  const scheduleCols: ColumnDef<MaintenanceScheduleDto>[] = [
    { header: 'Type', accessorKey: 'scheduleType', width: '130px', className: 'font-semibold' },
    { header: 'Interval', width: '130px', cell: (r) => <span className="font-mono">Every {r.intervalValue} {r.intervalUnit}</span> },
    { header: 'Next Due', width: '140px', cell: (r) => (r.nextDueDate ? formatDate(r.nextDueDate, locale) : '—') },
    { header: 'Status', width: '110px', cell: (r) => <StatusBadge status={r.isActive ? 'ACTIVE' : 'INACTIVE'} label={r.isActive ? 'Active' : 'Inactive'} /> },
  ];

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createWo.mutateAsync({ woNo: woNo.trim(), assetRef, maintenanceType, scheduledDate });
      toast.success('Maintenance work order created');
      setCreateOpen(false);
      setWoNo('');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Create failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <Breadcrumbs items={[{ label: 'Operations' }, { label: 'Maintenance (CMMS)' }]} />
          <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
            Maintenance Management (CMMS)
          </h1>
        </div>
        {canWrite ? (
          <Button variant="primary" size="sm" leftIcon={<Plus className="h-4 w-4" />} onClick={() => setCreateOpen(true)}>
            New Work Order
          </Button>
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard label="Work Orders" value={workOrders.length} icon={<Wrench className="h-4 w-4" />} accent="brand" />
        <KpiCard label="In Progress" value={workOrders.filter((w) => w.status === 'InProgress').length} icon={<Wrench className="h-4 w-4" />} accent="amber" />
        <KpiCard label="PM Schedules" value={schedules.length} icon={<Calendar className="h-4 w-4" />} accent="emerald" />
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="work-orders">Work Orders ({workOrders.length})</TabsTrigger>
          <TabsTrigger value="schedules">Preventive Schedules ({schedules.length})</TabsTrigger>
        </TabsList>

        <TabsContent value="work-orders" className="pt-4">
          <VirtualizedTable
            data={workOrders}
            columns={woCols}
            isLoading={woQ.isLoading}
            error={woQ.error}
            onRetry={() => void woQ.refetch()}
            emptyTitle="No maintenance work orders"
            toolbar={{
              filters: (
                <Select
                  value={statusFilter}
                  onValueChange={setStatusFilter}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Open', label: 'Open' },
                    { value: 'Assigned', label: 'Assigned' },
                    { value: 'InProgress', label: 'In Progress' },
                    { value: 'Completed', label: 'Completed' },
                    { value: 'Closed', label: 'Closed' },
                  ]}
                />
              ),
            }}
          />
        </TabsContent>
        <TabsContent value="schedules" className="pt-4">
          <VirtualizedTable data={schedules} columns={scheduleCols} isLoading={schedulesQ.isLoading} error={schedulesQ.error} onRetry={() => void schedulesQ.refetch()} emptyTitle="No schedules" />
        </TabsContent>
      </Tabs>

      <AppDialog open={createOpen} onOpenChange={setCreateOpen} title="Create Maintenance Work Order" size="md">
        <form onSubmit={handleCreate} className="space-y-4">
          <FormField label="Work Order No" htmlFor="mntWoNo" required>
            <Input id="mntWoNo" value={woNo} onChange={(e) => setWoNo(e.target.value)} placeholder="MWO-2026-001" required />
          </FormField>
          <FormField label="Asset UUID" htmlFor="mntAsset" required>
            <Input id="mntAsset" value={assetRef} onChange={(e) => setAssetRef(e.target.value)} required />
          </FormField>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Type" htmlFor="mntType">
              <Select
                id="mntType"
                value={maintenanceType}
                onValueChange={setMaintenanceType}
                options={[
                  { value: 'Preventive', label: 'Preventive' },
                  { value: 'Corrective', label: 'Corrective' },
                ]}
              />
            </FormField>
            <FormField label="Scheduled Date" htmlFor="mntDate">
              <Input id="mntDate" type="date" value={scheduledDate} onChange={(e) => setScheduledDate(e.target.value)} />
            </FormField>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>Cancel</Button>
            <Button type="submit" variant="primary" isLoading={createWo.isPending}>Create WO</Button>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
