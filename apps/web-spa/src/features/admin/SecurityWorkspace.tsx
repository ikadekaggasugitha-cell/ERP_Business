import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { KeyRound, Lock, Plus, ShieldCheck, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { VirtualizedTable, type ColumnDef } from '../../components/data-table/VirtualizedTable';
import { AppDialog } from '../../components/ui/AppDialog';
import { Button } from '../../components/ui/Button';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { EmptyState } from '../../components/ui/EmptyState';
import { ErrorState } from '../../components/ui/ErrorState';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { KpiCard } from '../../components/ui/KpiCard';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Switch } from '../../components/ui/Switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../components/ui/Tabs';
import { applyApiErrorsToForm, extractApiFieldErrors } from '../../lib/api';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  permissionFormSchema,
  roleFormSchema,
  useCreatePermissionMutation,
  useCreateRoleMutation,
  useDeleteRoleMutation,
  useGrantRolePermissionMutation,
  usePermissionsQuery,
  useRevokeRolePermissionMutation,
  useRolePermissionsQuery,
  useRolesQuery,
  useUpdateRoleMutation,
  type PermissionFormValues,
  type RoleFormValues,
  type SecurityPermission,
  type SecurityRole,
} from './admin-api';

export function SecurityWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, 'security:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <SecurityWorkspaceInner orgId={orgId} canWrite={canWrite} />
    </DomainErrorBoundary>
  );
}

interface SecurityWorkspaceInnerProps {
  orgId: string | undefined;
  canWrite: boolean;
}

function SecurityWorkspaceInner({ orgId, canWrite }: SecurityWorkspaceInnerProps) {
  const isId = useUiStore((s) => s.locale) === 'id';
  const [activeTab, setActiveTab] = useState('matrix');
  const [selectedRoleId, setSelectedRoleId] = useState<string | null>(null);
  const [moduleFilter, setModuleFilter] = useState('');

  const [roleDialogOpen, setRoleDialogOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<SecurityRole | null>(null);
  const [permDialogOpen, setPermDialogOpen] = useState(false);

  const rolesQuery = useRolesQuery(orgId, 1, 100);
  const permissionsQuery = usePermissionsQuery(orgId, moduleFilter);
  const roles = useMemo(() => rolesQuery.data?.items ?? [], [rolesQuery.data]);
  const permissions = useMemo(
    () => permissionsQuery.data?.items ?? [],
    [permissionsQuery.data],
  );

  const effectiveRoleId = useMemo(() => {
    if (selectedRoleId && roles.some((r) => r.id === selectedRoleId)) {
      return selectedRoleId;
    }
    return roles[0]?.id ?? null;
  }, [roles, selectedRoleId]);

  const activeRole = useMemo(
    () => roles.find((r) => r.id === effectiveRoleId) ?? null,
    [roles, effectiveRoleId],
  );

  const rolePermsQuery = useRolePermissionsQuery(orgId, effectiveRoleId);
  const grantedPerms = useMemo(() => rolePermsQuery.data ?? [], [rolePermsQuery.data]);
  const grantedSet = useMemo(() => new Set(grantedPerms.map((p) => p.id)), [grantedPerms]);

  const createRole = useCreateRoleMutation();
  const updateRole = useUpdateRoleMutation();
  const deleteRole = useDeleteRoleMutation();
  const createPerm = useCreatePermissionMutation();
  const grantPerm = useGrantRolePermissionMutation(effectiveRoleId ?? '');
  const revokePerm = useRevokeRolePermissionMutation(effectiveRoleId ?? '');

  const roleForm = useForm<RoleFormValues>({
    resolver: zodResolver(roleFormSchema),
    defaultValues: { code: '', name: '' },
  });

  const permForm = useForm<PermissionFormValues>({
    resolver: zodResolver(permissionFormSchema),
    defaultValues: { code: '', module: '', action: '' },
  });

  useEffect(() => {
    if (!roleDialogOpen) return;
    if (editingRole) {
      roleForm.reset({
        code: editingRole.code,
        name: editingRole.name,
      });
    } else {
      roleForm.reset({ code: '', name: '' });
    }
  }, [editingRole, roleDialogOpen, roleForm]);

  useEffect(() => {
    if (!permDialogOpen) return;
    permForm.reset({ code: '', module: '', action: '' });
  }, [permDialogOpen, permForm]);

  const permissionsByModule = useMemo(() => {
    const groups = new Map<string, SecurityPermission[]>();
    for (const perm of permissions) {
      const mod = perm.module || 'general';
      const arr = groups.get(mod) ?? [];
      arr.push(perm);
      groups.set(mod, arr);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [permissions]);

  const handleTogglePermission = async (perm: SecurityPermission, currentlyGranted: boolean) => {
    if (!effectiveRoleId || !canWrite) return;
    try {
      if (currentlyGranted) {
        await revokePerm.mutateAsync({ permissionId: perm.id });
        toast.success(`Revoked ${perm.code}`);
      } else {
        await grantPerm.mutateAsync({ permissionId: perm.id });
        toast.success(`Granted ${perm.code}`);
      }
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleSaveRole = async (values: RoleFormValues) => {
    try {
      if (editingRole) {
        await updateRole.mutateAsync({
          id: editingRole.id,
          rowVersion: editingRole.rowVersion,
          values,
        });
        toast.success('Role updated.');
      } else {
        const created = await createRole.mutateAsync(values);
        setSelectedRoleId(created.id);
        toast.success('Role created.');
      }
      setRoleDialogOpen(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, roleForm.setError));
    }
  };

  const handleDeleteRole = async (role: SecurityRole) => {
    try {
      await deleteRole.mutateAsync({ id: role.id, rowVersion: role.rowVersion });
      toast.success('Role deleted.');
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleSavePerm = async (values: PermissionFormValues) => {
    try {
      await createPerm.mutateAsync(values);
      toast.success('Permission created.');
      setPermDialogOpen(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, permForm.setError));
    }
  };

  const permColumns = useMemo<ColumnDef<SecurityPermission>[]>(
    () => [
      {
        accessorKey: 'code',
        header: 'Code',
        cell: (row) => <span className="font-mono text-xs font-semibold">{row.code}</span>,
      },
      {
        accessorKey: 'module',
        header: 'Module',
        cell: (row) => <span className="font-mono text-xs">{row.module}</span>,
      },
      {
        accessorKey: 'action',
        header: 'Action',
        cell: (row) => <StatusBadge status={row.action} />,
      },
    ],
    [],
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
            {isId ? 'Matriks RBAC, Peran & Hak Akses' : 'Security RBAC Matrix'}
          </h1>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            {isId
              ? 'Kelola peran keamanan, katalog izin modul, dan matriks pemberian hak akses interaktif.'
              : 'Configure roles, permission catalog, and interactive module-level RBAC permission assignments.'}
          </p>
        </div>

        {canWrite && (
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="secondary" onClick={() => setPermDialogOpen(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              New Permission
            </Button>
            <Button
              onClick={() => {
                setEditingRole(null);
                setRoleDialogOpen(true);
              }}
            >
              <Plus className="mr-1.5 h-4 w-4" />
              New Role
            </Button>
          </div>
        )}
      </div>

      {!canWrite && (
        <div
          role="status"
          className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-2.5 text-xs font-medium text-amber-800 dark:text-amber-300"
        >
          Read-only mode: security:write permission is required to modify roles or RBAC grants.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard
          label="Configured Roles"
          value={String(roles.length)}
          icon={<ShieldCheck className="h-4 w-4" />}
        />
        <KpiCard
          label="Permission Catalog"
          value={String(permissions.length)}
          accent="sky"
          icon={<KeyRound className="h-4 w-4" />}
        />
        <KpiCard
          label={activeRole ? `Granted on ${activeRole.code}` : 'Active Grants'}
          value={String(grantedPerms.length)}
          accent="emerald"
        />
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="matrix">RBAC Matrix</TabsTrigger>
          <TabsTrigger value="catalog">Permission Catalog</TabsTrigger>
        </TabsList>

        <TabsContent value="matrix" className="mt-4">
          {rolesQuery.isError ? (
            <ErrorState error={rolesQuery.error} onRetry={() => void rolesQuery.refetch()} />
          ) : roles.length === 0 ? (
            <EmptyState
              title="No security roles created"
              description="Create your first role to start binding permissions."
            />
          ) : (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
              {/* Role Selector Sidebar */}
              <div className="space-y-2 lg:col-span-4">
                {roles.map((role) => {
                  const selected = role.id === effectiveRoleId;
                  return (
                    <div
                      key={role.id}
                      onClick={() => setSelectedRoleId(role.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') setSelectedRoleId(role.id);
                      }}
                      className={`flex cursor-pointer items-center justify-between rounded-xl border p-3 text-sm transition ${
                        selected
                          ? 'border-brand-500 bg-brand-500/10'
                          : 'border-[var(--border-subtle)] bg-[var(--surface-card)] hover:bg-[var(--surface-subtle)]'
                      }`}
                    >
                      <div>
                        <div className="flex items-center gap-1.5 font-mono text-xs font-semibold text-[var(--text-primary)]">
                          {role.code}
                          {role.isSystem && (
                            <span
                              title="System Role Locked"
                              className="inline-flex items-center gap-0.5 rounded bg-slate-500/15 px-1.5 py-0.5 text-[10px] text-slate-600 dark:text-slate-300"
                            >
                              <Lock className="h-2.5 w-2.5" />
                              SYSTEM
                            </span>
                          )}
                        </div>
                        <div className="text-xs text-[var(--text-secondary)]">{role.name}</div>
                      </div>

                      {canWrite && (
                        <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              setEditingRole(role);
                              setRoleDialogOpen(true);
                            }}
                          >
                            Edit
                          </Button>
                          {!role.isSystem && (
                            <Button
                              size="sm"
                              variant="ghost"
                              aria-label={`Delete role ${role.code}`}
                              onClick={() => handleDeleteRole(role)}
                              disabled={deleteRole.isPending}
                            >
                              <Trash2 className="h-3.5 w-3.5 text-rose-600" />
                            </Button>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Permission Toggle Matrix */}
              <div className="space-y-4 lg:col-span-8">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h2 className="text-base font-semibold text-[var(--text-primary)]">
                      {activeRole ? `${activeRole.code} — ${activeRole.name}` : 'Select a role'}
                    </h2>
                    <p className="text-xs text-[var(--text-muted)]">
                      Toggle switches to immediately grant or revoke permissions via security-service.
                    </p>
                  </div>
                  <div className="w-52">
                    <Input
                      value={moduleFilter}
                      onChange={(e) => setModuleFilter(e.target.value)}
                      placeholder="Filter module (e.g. hr)..."
                    />
                  </div>
                </div>

                {permissionsQuery.isError || rolePermsQuery.isError ? (
                  <ErrorState
                    error={permissionsQuery.error ?? rolePermsQuery.error}
                    onRetry={() => {
                      void permissionsQuery.refetch();
                      void rolePermsQuery.refetch();
                    }}
                  />
                ) : permissionsByModule.length === 0 ? (
                  <EmptyState title="No permissions found for matrix" />
                ) : (
                  <div className="space-y-4">
                    {permissionsByModule.map(([modName, perms]) => (
                      <div
                        key={modName}
                        className="rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] p-4"
                      >
                        <div className="mb-3 flex items-center justify-between border-b border-[var(--border-subtle)] pb-2">
                          <span className="font-mono text-xs font-bold uppercase tracking-wider text-[var(--text-primary)]">
                            Module: {modName}
                          </span>
                          <span className="text-xs text-[var(--text-muted)]">
                            {perms.filter((p) => grantedSet.has(p.id)).length} / {perms.length} granted
                          </span>
                        </div>

                        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                          {perms.map((perm) => {
                            const isGranted = grantedSet.has(perm.id);
                            return (
                              <div
                                key={perm.id}
                                className="flex items-center justify-between rounded-lg border border-[var(--border-subtle)] bg-[var(--surface-subtle)] px-3 py-2 text-sm"
                              >
                                <div>
                                  <div className="font-mono text-xs font-semibold text-[var(--text-primary)]">
                                    {perm.code}
                                  </div>
                                  <div className="text-[11px] text-[var(--text-muted)]">
                                    action: {perm.action}
                                  </div>
                                </div>
                                <Switch
                                  checked={isGranted}
                                  disabled={!canWrite || grantPerm.isPending || revokePerm.isPending}
                                  label={`Toggle ${perm.code}`}
                                  onCheckedChange={() => handleTogglePermission(perm, isGranted)}
                                />
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </TabsContent>

        <TabsContent value="catalog" className="mt-4">
          <VirtualizedTable
            data={permissions}
            columns={permColumns}
            getRowId={(r) => r.id}
            isLoading={permissionsQuery.isLoading}
            error={permissionsQuery.error}
            onRetry={() => void permissionsQuery.refetch()}
            emptyTitle="No permissions defined"
            toolbar={{
              filters: (
                <div className="w-56">
                  <Input
                    value={moduleFilter}
                    onChange={(e) => setModuleFilter(e.target.value)}
                    placeholder="Filter exact module..."
                  />
                </div>
              ),
            }}
          />
        </TabsContent>
      </Tabs>

      {/* Role Create / Edit Dialog */}
      <AppDialog
        open={roleDialogOpen}
        onOpenChange={setRoleDialogOpen}
        title={editingRole ? `Edit Role: ${editingRole.code}` : 'New Security Role'}
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setRoleDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="security-role-form"
              disabled={createRole.isPending || updateRole.isPending}
            >
              {editingRole ? 'Save Role' : 'Create Role'}
            </Button>
          </div>
        }
      >
        <form id="security-role-form" onSubmit={roleForm.handleSubmit(handleSaveRole)} className="space-y-4">
          <FormField
            label="Role Code"
            htmlFor="sec-role-code"
            error={roleForm.formState.errors.code?.message}
            required
          >
            <Input
              id="sec-role-code"
              disabled={Boolean(editingRole)}
              placeholder="PAYROLL_MANAGER"
              {...roleForm.register('code')}
            />
          </FormField>
          <FormField
            label="Role Name"
            htmlFor="sec-role-name"
            error={roleForm.formState.errors.name?.message}
            required
          >
            <Input
              id="sec-role-name"
              placeholder="Payroll Manager"
              {...roleForm.register('name')}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Permission Create Dialog */}
      <AppDialog
        open={permDialogOpen}
        onOpenChange={setPermDialogOpen}
        title="New Permission Definition"
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setPermDialogOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" form="security-perm-form" disabled={createPerm.isPending}>
              Create Permission
            </Button>
          </div>
        }
      >
        <form id="security-perm-form" onSubmit={permForm.handleSubmit(handleSavePerm)} className="space-y-4">
          <FormField
            label="Permission Code"
            htmlFor="sec-perm-code"
            error={permForm.formState.errors.code?.message}
            required
          >
            <Input id="sec-perm-code" placeholder="payroll:write" {...permForm.register('code')} />
          </FormField>
          <div className="grid grid-cols-2 gap-4">
            <FormField
              label="Module"
              htmlFor="sec-perm-module"
              error={permForm.formState.errors.module?.message}
              required
            >
              <Input id="sec-perm-module" placeholder="payroll" {...permForm.register('module')} />
            </FormField>
            <FormField
              label="Action"
              htmlFor="sec-perm-action"
              error={permForm.formState.errors.action?.message}
              required
            >
              <Input id="sec-perm-action" placeholder="write" {...permForm.register('action')} />
            </FormField>
          </div>
        </form>
      </AppDialog>
    </div>
  );
}
