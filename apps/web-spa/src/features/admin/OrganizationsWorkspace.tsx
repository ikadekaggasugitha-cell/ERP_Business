import { useEffect, useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  Building2,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  FolderTree,
  Layers,
  Pencil,
  Plus,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { Combobox } from '../../components/ui/Combobox';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { applyApiErrorsToForm, extractApiFieldErrors } from '../../lib/api';
import { formatDate } from '../../lib/decimal';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  buildOrgHierarchyTree,
  createOrgTypeSchema,
  orgFormSchema,
  useActivateOrgMutation,
  useCreateOrgMutation,
  useCreateOrgTypeMutation,
  useDeleteOrgMutation,
  useOrganizationTypesQuery,
  useOrganizationsQuery,
  useOrgHierarchyQuery,
  useUpdateOrgMutation,
  type CreateOrgTypeValues,
  type OrgFormValues,
  type OrgRecord,
  type OrgTreeNode,
  type OrgType,
} from './admin-api';

export function OrganizationsWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, 'org:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <OrganizationsWorkspaceInner orgId={orgId} canWrite={canWrite} />
    </DomainErrorBoundary>
  );
}

interface OrganizationsWorkspaceInnerProps {
  orgId: string | undefined;
  canWrite: boolean;
}

function OrganizationsWorkspaceInner({ orgId, canWrite }: OrganizationsWorkspaceInnerProps) {
  const isId = useUiStore((s) => s.locale) === 'id';
  const [activeTab, setActiveTab] = useState('tree');
  const [resourceMode, setResourceMode] = useState<'organizations' | 'org-units'>('organizations');
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [collapsedIds, setCollapsedIds] = useState<Record<string, boolean>>({});
  const [selectedRootId, setSelectedRootId] = useState<string | null>(null);

  const [orgDialogOpen, setOrgDialogOpen] = useState(false);
  const [editingOrg, setEditingOrg] = useState<OrgRecord | null>(null);
  const [typeDialogOpen, setTypeDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<OrgRecord | null>(null);

  const typesQuery = useOrganizationTypesQuery(orgId);
  const orgsQuery = useOrganizationsQuery(orgId, resourceMode, statusFilter, 1, 100);
  const hierarchyQuery = useOrgHierarchyQuery(orgId, resourceMode, selectedRootId);

  const createTypeMutation = useCreateOrgTypeMutation();
  const createOrgMutation = useCreateOrgMutation(resourceMode);
  const updateOrgMutation = useUpdateOrgMutation(resourceMode);
  const activateOrgMutation = useActivateOrgMutation(resourceMode);
  const deleteOrgMutation = useDeleteOrgMutation(resourceMode);

  const orgTypes = useMemo(() => typesQuery.data?.items ?? [], [typesQuery.data]);
  const orgList = useMemo(() => orgsQuery.data?.items ?? [], [orgsQuery.data]);
  const hierarchyNodes = useMemo(
    () => hierarchyQuery.data?.items ?? [],
    [hierarchyQuery.data],
  );

  const displayedNodes = useMemo(() => {
    if (selectedRootId && hierarchyNodes.length > 0) {
      return hierarchyNodes;
    }
    return orgList;
  }, [selectedRootId, hierarchyNodes, orgList]);

  const treeRoots = useMemo(() => buildOrgHierarchyTree(displayedNodes), [displayedNodes]);

  const orgTypeMap = useMemo(() => {
    const map = new Map<string, OrgType>();
    for (const t of orgTypes) map.set(t.id, t);
    return map;
  }, [orgTypes]);

  const activeCount = useMemo(
    () => orgList.filter((o) => o.status.toUpperCase() === 'ACTIVE').length,
    [orgList],
  );
  const draftCount = useMemo(
    () => orgList.filter((o) => o.status.toUpperCase() === 'DRAFT').length,
    [orgList],
  );

  const orgForm = useForm<OrgFormValues>({
    resolver: zodResolver(orgFormSchema),
    defaultValues: {
      code: '',
      name: '',
      organizationTypeId: '',
      parentId: '',
      effectiveFrom: new Date().toISOString().slice(0, 10),
      effectiveTo: '',
    },
  });

  const typeForm = useForm<CreateOrgTypeValues>({
    resolver: zodResolver(createOrgTypeSchema),
    defaultValues: {
      code: '',
      name: '',
      levelHint: 1,
    },
  });

  useEffect(() => {
    if (!orgDialogOpen) return;
    if (editingOrg) {
      orgForm.reset({
        code: editingOrg.code,
        name: editingOrg.name,
        organizationTypeId: editingOrg.organizationTypeId,
        parentId: editingOrg.parentId ?? '',
        effectiveFrom: editingOrg.effectiveFrom,
        effectiveTo: editingOrg.effectiveTo ?? '',
      });
    } else {
      orgForm.reset({
        code: '',
        name: '',
        organizationTypeId: orgTypes[0]?.id ?? '',
        parentId: '',
        effectiveFrom: new Date().toISOString().slice(0, 10),
        effectiveTo: '',
      });
    }
  }, [editingOrg, orgDialogOpen, orgForm, orgTypes]);

  useEffect(() => {
    if (!typeDialogOpen) return;
    typeForm.reset({ code: '', name: '', levelHint: 1 });
  }, [typeDialogOpen, typeForm]);

  const toggleCollapse = (id: string) => {
    setCollapsedIds((prev) => ({ ...prev, [id]: !prev[id] }));
  };

  const expandAll = () => setCollapsedIds({});

  const collapseAll = () => {
    const next: Record<string, boolean> = {};
    for (const node of displayedNodes) {
      next[node.id] = true;
    }
    setCollapsedIds(next);
  };

  const handleActivate = async (node: OrgRecord) => {
    try {
      await activateOrgMutation.mutateAsync({ id: node.id, rowVersion: node.rowVersion });
      toast.success(isId ? 'Organisasi diaktifkan.' : 'Organization activated.');
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteOrgMutation.mutateAsync({
        id: deleteTarget.id,
        rowVersion: deleteTarget.rowVersion,
      });
      toast.success(isId ? 'Organisasi dihapus.' : 'Organization deleted.');
      setDeleteTarget(null);
      if (selectedRootId === deleteTarget.id) setSelectedRootId(null);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleSaveOrg = async (values: OrgFormValues) => {
    try {
      if (editingOrg) {
        await updateOrgMutation.mutateAsync({
          id: editingOrg.id,
          rowVersion: editingOrg.rowVersion,
          values,
        });
        toast.success(isId ? 'Organisasi diperbarui.' : 'Organization updated.');
      } else {
        await createOrgMutation.mutateAsync(values);
        toast.success(isId ? 'Organisasi dibuat.' : 'Organization created.');
      }
      setOrgDialogOpen(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, orgForm.setError));
    }
  };

  const handleSaveType = async (values: CreateOrgTypeValues) => {
    try {
      await createTypeMutation.mutateAsync(values);
      toast.success(isId ? 'Tipe organisasi dibuat.' : 'Organization type created.');
      setTypeDialogOpen(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, typeForm.setError));
    }
  };

  const typeColumns = useMemo<ColumnDef<OrgType>[]>(
    () => [
      {
        accessorKey: 'code',
        header: 'Code',
        cell: (row) => <span className="font-mono text-xs font-semibold">{row.code}</span>,
      },
      {
        accessorKey: 'name',
        header: 'Type Name',
      },
      {
        accessorKey: 'levelHint',
        header: 'Level Hint',
        cell: (row) => <span className="font-mono text-xs">{row.levelHint ?? '—'}</span>,
      },
      {
        accessorKey: 'isActive',
        header: 'Status',
        cell: (row) => <StatusBadge status={row.isActive ? 'ACTIVE' : 'INACTIVE'} />,
      },
    ],
    [],
  );

  const renderTreeNode = (entry: OrgTreeNode, depth = 0) => {
    const { node, children } = entry;
    const hasChildren = children.length > 0;
    const isCollapsed = Boolean(collapsedIds[node.id]);
    const isDraft = node.status.toUpperCase() === 'DRAFT';
    const typeInfo = orgTypeMap.get(node.organizationTypeId);

    return (
      <div key={node.id} className="space-y-1.5" data-testid={`org-tree-node-${node.code}`}>
        <div
          style={{ marginLeft: `${depth * 20}px` }}
          className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] px-3 py-2.5 text-sm shadow-2xs"
        >
          <div className="flex items-center gap-2">
            {hasChildren ? (
              <button
                type="button"
                onClick={() => toggleCollapse(node.id)}
                aria-label={isCollapsed ? `Expand ${node.code}` : `Collapse ${node.code}`}
                className="inline-flex h-6 w-6 items-center justify-center rounded-md border border-[var(--border-subtle)] bg-[var(--surface-subtle)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
              >
                {isCollapsed ? (
                  <ChevronRight className="h-3.5 w-3.5" />
                ) : (
                  <ChevronDown className="h-3.5 w-3.5" />
                )}
              </button>
            ) : (
              <span className="inline-block w-6 text-center text-xs text-[var(--text-muted)]">
                •
              </span>
            )}

            <div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-mono text-xs font-semibold text-brand-700 dark:text-brand-400">
                  {node.code}
                </span>
                <span className="font-medium text-[var(--text-primary)]">{node.name}</span>
                <StatusBadge status={node.status} />
                {typeInfo && (
                  <span className="rounded bg-[var(--surface-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--text-muted)]">
                    {typeInfo.name}
                  </span>
                )}
              </div>
              <div className="mt-0.5 font-mono text-[11px] text-[var(--text-muted)]">
                path: {node.path} · effective {formatDate(node.effectiveFrom)}
              </div>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            <Button
              size="sm"
              variant={selectedRootId === node.id ? 'primary' : 'outline'}
              onClick={() => setSelectedRootId(selectedRootId === node.id ? null : node.id)}
            >
              {selectedRootId === node.id ? 'Show All' : 'Subtree'}
            </Button>

            {canWrite && (
              <>
                {isDraft && (
                  <Button
                    size="sm"
                    onClick={() => handleActivate(node)}
                    disabled={activateOrgMutation.isPending}
                  >
                    <CheckCircle2 className="mr-1 h-3.5 w-3.5" />
                    Activate
                  </Button>
                )}
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => {
                    setEditingOrg(node);
                    setOrgDialogOpen(true);
                  }}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button
                  size="sm"
                  variant="danger"
                  onClick={() => setDeleteTarget(node)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </>
            )}
          </div>
        </div>

        {hasChildren && !isCollapsed && (
          <div className="space-y-1.5">{children.map((child) => renderTreeNode(child, depth + 1))}</div>
        )}
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
            {isId ? 'Hierarki Organisasi & Unit Kerja' : 'Organization Tree & Units'}
          </h1>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            {isId
              ? 'Kelola struktur holding, entitas hukum, unit kerja, tipe organisasi, dan aktivasi lifecycle.'
              : 'Manage holding hierarchy, legal entities, org units, structural types, and Draft-to-Active activation.'}
          </p>
        </div>

        {canWrite && (
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="secondary" onClick={() => setTypeDialogOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              New Org Type
            </Button>
            <Button
              onClick={() => {
                setEditingOrg(null);
                setOrgDialogOpen(true);
              }}
            >
              <Plus className="mr-1.5 h-4 w-4" />
              New Organization
            </Button>
          </div>
        )}
      </div>

      {!canWrite && (
        <div
          role="status"
          className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-2.5 text-xs font-medium text-amber-800 dark:text-amber-300"
        >
          Read-only mode: org:write permission is required to mutate organizations or types.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label="Total Units"
          value={String(orgList.length)}
          icon={<Building2 className="h-4 w-4" />}
        />
        <KpiCard
          label="Active Units"
          value={String(activeCount)}
          accent="emerald"
          icon={<CheckCircle2 className="h-4 w-4" />}
        />
        <KpiCard label="Draft Units" value={String(draftCount)} accent="amber" />
        <KpiCard
          label="Organization Types"
          value={String(orgTypes.length)}
          accent="sky"
          icon={<Layers className="h-4 w-4" />}
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="tree">
            <FolderTree className="mr-1.5 h-4 w-4" />
            Hierarchy Tree
          </TabsTrigger>
          <TabsTrigger value="types">
            <Layers className="mr-1.5 h-4 w-4" />
            Organization Types
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tree" className="mt-4 space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-3">
            <div className="flex flex-wrap items-center gap-2">
              <div className="w-44">
                <Select
                  value={resourceMode}
                  onValueChange={(next) => {
                    setSelectedRootId(null);
                    setResourceMode(next as 'organizations' | 'org-units');
                  }}
                  options={[
                    { value: 'organizations', label: 'Endpoint: /organizations' },
                    { value: 'org-units', label: 'Endpoint: /org-units' },
                  ]}
                />
              </div>
              <div className="w-36">
                <Select
                  value={statusFilter}
                  onValueChange={setStatusFilter}
                  options={[
                    { value: 'ALL', label: 'All Statuses' },
                    { value: 'Draft', label: 'Draft' },
                    { value: 'Active', label: 'Active' },
                  ]}
                />
              </div>
              {selectedRootId && (
                <Button size="sm" variant="secondary" onClick={() => setSelectedRootId(null)}>
                  Clear Subtree Filter
                </Button>
              )}
            </div>

            <div className="flex items-center gap-2">
              <Button size="sm" variant="outline" onClick={expandAll}>
                Expand All
              </Button>
              <Button size="sm" variant="outline" onClick={collapseAll}>
                Collapse All
              </Button>
            </div>
          </div>

          {orgsQuery.isLoading || (selectedRootId && hierarchyQuery.isLoading) ? (
            <div className="space-y-2">
              <div className="h-14 animate-pulse rounded-xl bg-[var(--surface-muted)]" />
              <div className="h-14 animate-pulse rounded-xl bg-[var(--surface-muted)]" />
            </div>
          ) : orgsQuery.isError ? (
            <ErrorState error={orgsQuery.error} onRetry={() => void orgsQuery.refetch()} />
          ) : hierarchyQuery.isError ? (
            <ErrorState
              error={hierarchyQuery.error}
              onRetry={() => void hierarchyQuery.refetch()}
            />
          ) : treeRoots.length === 0 ? (
            <EmptyState
              title="No organizations or units found"
              description="Create an organization type first, then add your root company or holding node."
            />
          ) : (
            <div className="space-y-2">{treeRoots.map((root) => renderTreeNode(root, 0))}</div>
          )}
        </TabsContent>

        <TabsContent value="types" className="mt-4">
          <VirtualizedTable
            data={orgTypes}
            columns={typeColumns}
            getRowId={(r) => r.id}
            isLoading={typesQuery.isLoading}
            error={typesQuery.error}
            onRetry={() => void typesQuery.refetch()}
            emptyTitle="No organization types defined"
            emptyDescription="Define structural types such as HOLDING, COMPANY, BRANCH, or DEPARTMENT."
          />
        </TabsContent>
      </Tabs>

      {/* Organization Create / Edit Dialog */}
      <AppDialog
        open={orgDialogOpen}
        onOpenChange={setOrgDialogOpen}
        title={editingOrg ? `Edit ${editingOrg.code}` : 'New Organization / Unit'}
        size="lg"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setOrgDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="org-unit-form"
              disabled={createOrgMutation.isPending || updateOrgMutation.isPending}
            >
              {editingOrg ? 'Save Changes' : 'Create'}
            </Button>
          </div>
        }
      >
        <form id="org-unit-form" onSubmit={orgForm.handleSubmit(handleSaveOrg)} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <FormField
            label="Code"
            htmlFor="org-code"
            error={orgForm.formState.errors.code?.message}
            required
          >
            <Input id="org-code" disabled={Boolean(editingOrg)} {...orgForm.register('code')} />
          </FormField>

          <FormField
            label="Name"
            htmlFor="org-name"
            error={orgForm.formState.errors.name?.message}
            required
          >
            <Input id="org-name" {...orgForm.register('name')} />
          </FormField>

          <FormField
            label="Organization Type"
            error={orgForm.formState.errors.organizationTypeId?.message}
            required
          >
            <Controller
              control={orgForm.control}
              name="organizationTypeId"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  options={orgTypes.map((t) => ({
                    value: t.id,
                    label: `${t.code} — ${t.name}`,
                  }))}
                  placeholder="Choose type..."
                />
              )}
            />
          </FormField>

          <FormField
            label="Parent Unit (Optional)"
            error={orgForm.formState.errors.parentId?.message}
          >
            <Controller
              control={orgForm.control}
              name="parentId"
              render={({ field }) => (
                <Combobox
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  allowClear
                  disabled={Boolean(editingOrg)}
                  options={orgList
                    .filter((o) => o.id !== editingOrg?.id)
                    .map((o) => ({ value: o.id, label: `${o.code} — ${o.name}` }))}
                  placeholder="Top-level root (none)"
                />
              )}
            />
          </FormField>

          <FormField
            label="Effective From"
            htmlFor="org-eff-from"
            error={orgForm.formState.errors.effectiveFrom?.message}
          >
            <Input id="org-eff-from" type="date" {...orgForm.register('effectiveFrom')} />
          </FormField>

          <FormField
            label="Effective To (Optional)"
            htmlFor="org-eff-to"
            error={orgForm.formState.errors.effectiveTo?.message}
          >
            <Input id="org-eff-to" type="date" {...orgForm.register('effectiveTo')} />
          </FormField>
        </form>
      </AppDialog>

      {/* Organization Type Create Dialog */}
      <AppDialog
        open={typeDialogOpen}
        onOpenChange={setTypeDialogOpen}
        title="New Organization Type"
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setTypeDialogOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" form="org-type-form" disabled={createTypeMutation.isPending}>
              Create Type
            </Button>
          </div>
        }
      >
        <form id="org-type-form" onSubmit={typeForm.handleSubmit(handleSaveType)} className="space-y-4">
          <FormField label="Type Code" htmlFor="org-type-code" error={typeForm.formState.errors.code?.message} required>
            <Input id="org-type-code" placeholder="BRANCH" {...typeForm.register('code')} />
          </FormField>
          <FormField label="Type Name" htmlFor="org-type-name" error={typeForm.formState.errors.name?.message} required>
            <Input id="org-type-name" placeholder="Regional Branch" {...typeForm.register('name')} />
          </FormField>
          <FormField label="Hierarchy Level Hint" htmlFor="org-type-level">
            <Input id="org-type-level" type="number" min={0} max={99} {...typeForm.register('levelHint', { valueAsNumber: true })} />
          </FormField>
        </form>
      </AppDialog>

      {/* Delete Confirmation */}
      <AppDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDeleteTarget(null);
        }}
        title="Delete Organization Unit"
        description="Units with child descendants cannot be deleted (HTTP 409 HAS_CHILDREN)."
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmDelete}
              disabled={deleteOrgMutation.isPending}
            >
              Delete
            </Button>
          </div>
        }
      >
        {deleteTarget && (
          <div className="rounded-lg border border-rose-500/20 bg-rose-500/5 p-3 text-sm font-mono">
            {deleteTarget.code} — {deleteTarget.name}
          </div>
        )}
      </AppDialog>
    </div>
  );
}
