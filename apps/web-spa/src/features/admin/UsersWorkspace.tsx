import { useEffect, useMemo, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Pencil, Shield, Trash2, UserCheck, UserPlus, Users, UserX } from 'lucide-react';
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
import { SideDrawer } from '../../components/ui/SideDrawer';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { applyApiErrorsToForm, extractApiFieldErrors } from '../../lib/api';
import { hasUserPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import {
  useAssignUserRoleMutation,
  useCreateUserMutation,
  useDisableUserMutation,
  useOrganizationsQuery,
  useRevokeUserRoleMutation,
  useRolesQuery,
  useUpdateUserMutation,
  useUserRolesQuery,
  useUsersQuery,
  userFormSchema,
  type AdminUser,
  type UserFormValues,
} from './admin-api';

export function UsersWorkspace() {
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);
  const orgId = activeOrg?.id;
  const canWrite = hasUserPermission(user, 'user:write');

  return (
    <DomainErrorBoundary resetKey={orgId}>
      <UsersWorkspaceInner orgId={orgId} canWrite={canWrite} />
    </DomainErrorBoundary>
  );
}

interface UsersWorkspaceInnerProps {
  orgId: string | undefined;
  canWrite: boolean;
}

function UsersWorkspaceInner({ orgId, canWrite }: UsersWorkspaceInnerProps) {
  const isId = useUiStore((s) => s.locale) === 'id';
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [searchTerm, setSearchTerm] = useState('');

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [disableTarget, setDisableTarget] = useState<AdminUser | null>(null);
  const [rolesDrawerUser, setRolesDrawerUser] = useState<AdminUser | null>(null);

  const [assignRoleId, setAssignRoleId] = useState('');
  const [assignOrgId, setAssignOrgId] = useState('');

  const usersQuery = useUsersQuery(orgId, statusFilter, page, 25);
  const rolesQuery = useRolesQuery(orgId, 1, 100);
  const orgsQuery = useOrganizationsQuery(orgId, 'organizations', 'ALL', 1, 100);
  const userRolesQuery = useUserRolesQuery(orgId, rolesDrawerUser?.id ?? null);

  const createUser = useCreateUserMutation();
  const updateUser = useUpdateUserMutation();
  const disableUser = useDisableUserMutation();
  const assignRole = useAssignUserRoleMutation(rolesDrawerUser?.id ?? '');
  const revokeRole = useRevokeUserRoleMutation(rolesDrawerUser?.id ?? '');

  const users = useMemo(() => usersQuery.data?.items ?? [], [usersQuery.data]);
  const meta = usersQuery.data?.page;
  const roles = useMemo(() => rolesQuery.data?.items ?? [], [rolesQuery.data]);
  const organizations = useMemo(() => orgsQuery.data?.items ?? [], [orgsQuery.data]);
  const assignedRoles = useMemo(() => userRolesQuery.data ?? [], [userRolesQuery.data]);

  const roleMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const r of roles) m.set(r.id, `${r.code} — ${r.name}`);
    return m;
  }, [roles]);

  const orgMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const o of organizations) m.set(o.id, `${o.code} — ${o.name}`);
    return m;
  }, [organizations]);

  const filteredUsers = useMemo(() => {
    const q = searchTerm.trim().toLowerCase();
    if (!q) return users;
    return users.filter(
      (u) =>
        u.username.toLowerCase().includes(q) ||
        u.fullName.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q),
    );
  }, [users, searchTerm]);

  const activeCount = useMemo(
    () => users.filter((u) => u.status.toUpperCase() === 'ACTIVE').length,
    [users],
  );
  const inactiveCount = useMemo(
    () => users.filter((u) => u.status.toUpperCase() === 'INACTIVE').length,
    [users],
  );

  const userForm = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      username: '',
      email: '',
      fullName: '',
      defaultOrganizationId: orgId ?? '',
    },
  });

  useEffect(() => {
    if (!userDialogOpen) return;
    if (editingUser) {
      userForm.reset({
        username: editingUser.username,
        email: editingUser.email,
        fullName: editingUser.fullName,
        defaultOrganizationId: editingUser.defaultOrganizationId ?? '',
      });
    } else {
      userForm.reset({
        username: '',
        email: '',
        fullName: '',
        defaultOrganizationId: orgId ?? '',
      });
    }
  }, [editingUser, orgId, userDialogOpen, userForm]);

  useEffect(() => {
    if (rolesDrawerUser) {
      setAssignRoleId(roles[0]?.id ?? '');
      setAssignOrgId(rolesDrawerUser.defaultOrganizationId ?? orgId ?? '');
    }
  }, [orgId, roles, rolesDrawerUser]);

  const handleSaveUser = async (values: UserFormValues) => {
    try {
      if (editingUser) {
        await updateUser.mutateAsync({
          id: editingUser.id,
          rowVersion: editingUser.rowVersion,
          values,
        });
        toast.success(isId ? 'Pengguna diperbarui.' : 'User updated.');
      } else {
        await createUser.mutateAsync(values);
        toast.success(isId ? 'Pengguna dibuat.' : 'User created.');
      }
      setUserDialogOpen(false);
    } catch (err) {
      toast.error(applyApiErrorsToForm(err, userForm.setError));
    }
  };

  const handleConfirmDisable = async () => {
    if (!disableTarget) return;
    try {
      await disableUser.mutateAsync({
        id: disableTarget.id,
        rowVersion: disableTarget.rowVersion,
      });
      toast.success(isId ? 'Pengguna dinonaktifkan.' : 'User disabled.');
      setDisableTarget(null);
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleAssignRole = async () => {
    if (!rolesDrawerUser || !assignRoleId) return;
    try {
      await assignRole.mutateAsync({
        roleId: assignRoleId,
        organizationRefId: assignOrgId,
      });
      toast.success(isId ? 'Peran ditambahkan ke pengguna.' : 'Role assigned to user.');
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const handleRevokeRole = async (roleId: string) => {
    try {
      await revokeRole.mutateAsync({ roleId });
      toast.success(isId ? 'Peran dicabut.' : 'Role assignment revoked.');
    } catch (err) {
      toast.error(extractApiFieldErrors(err).message);
    }
  };

  const columns = useMemo<ColumnDef<AdminUser>[]>(
    () => [
      {
        accessorKey: 'username',
        header: 'Username',
        cell: (row) => (
          <span className="font-mono text-xs font-semibold text-[var(--text-primary)]">
            {row.username}
          </span>
        ),
      },
      {
        accessorKey: 'fullName',
        header: isId ? 'Nama & Email' : 'Name & Email',
        cell: (row) => (
          <div>
            <div className="font-medium text-[var(--text-primary)]">{row.fullName}</div>
            <div className="text-xs text-[var(--text-muted)]">{row.email}</div>
          </div>
        ),
      },
      {
        accessorKey: 'defaultOrganizationId',
        header: 'Default Org',
        cell: (row) => (
          <span className="text-xs text-[var(--text-secondary)]">
            {row.defaultOrganizationId
              ? orgMap.get(row.defaultOrganizationId) ?? row.defaultOrganizationId
              : '—'}
          </span>
        ),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: (row) => <StatusBadge status={row.status} />,
      },
      {
        header: 'Actions',
        cell: (u) => {
          const isInactive = u.status.toUpperCase() === 'INACTIVE';
          return (
            <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
              <Button size="sm" variant="outline" onClick={() => setRolesDrawerUser(u)}>
                <Shield className="mr-1 h-3.5 w-3.5" />
                Roles
              </Button>
              {canWrite && (
                <>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      setEditingUser(u);
                      setUserDialogOpen(true);
                    }}
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                  {!isInactive && (
                    <Button size="sm" variant="danger" onClick={() => setDisableTarget(u)}>
                      Disable
                    </Button>
                  )}
                </>
              )}
            </div>
          );
        },
      },
    ],
    [canWrite, isId, orgMap],
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-[var(--text-primary)]">
            {isId ? 'Direktori Pengguna & Penugasan Peran' : 'Users & Per-Org Role Directory'}
          </h1>
          <p className="mt-1 text-sm text-[var(--text-secondary)]">
            {isId
              ? 'Kelola akun pengguna, penonaktifan (disable), dan penugasan peran RBAC per organisasi.'
              : 'Manage user accounts, status deactivation, and scoped RBAC role assignments per organization.'}
          </p>
        </div>

        {canWrite && (
          <Button
            onClick={() => {
              setEditingUser(null);
              setUserDialogOpen(true);
            }}
          >
            <UserPlus className="mr-1.5 h-4 w-4" />
            New User
          </Button>
        )}
      </div>

      {!canWrite && (
        <div
          role="status"
          className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-2.5 text-xs font-medium text-amber-800 dark:text-amber-300"
        >
          Read-only mode: user:write permission is required to manage users or assign roles.
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <KpiCard
          label="Directory Users"
          value={String(meta?.totalElements ?? users.length)}
          icon={<Users className="h-4 w-4" />}
        />
        <KpiCard
          label="Active Accounts"
          value={String(activeCount)}
          accent="emerald"
          icon={<UserCheck className="h-4 w-4" />}
        />
        <KpiCard
          label="Inactive / Disabled"
          value={String(inactiveCount)}
          accent="rose"
          icon={<UserX className="h-4 w-4" />}
        />
      </div>

      <VirtualizedTable
        data={filteredUsers}
        columns={columns}
        getRowId={(r) => r.id}
        isLoading={usersQuery.isLoading}
        error={usersQuery.error}
        onRetry={() => void usersQuery.refetch()}
        onRowClick={(row) => setRolesDrawerUser(row)}
        emptyTitle="No users found"
        emptyDescription="Create a user account or clear status filters."
        onResetFilters={() => {
          setStatusFilter('ALL');
          setSearchTerm('');
          setPage(1);
        }}
        pagination={{
          page,
          size: 25,
          totalElements: meta?.totalElements ?? filteredUsers.length,
          totalPages: Math.max(1, meta?.totalPages ?? 1),
          pageBase: 1,
          onPageChange: setPage,
        }}
        toolbar={{
          searchValue: searchTerm,
          onSearchChange: setSearchTerm,
          searchPlaceholder: 'Search username, name, or email...',
          filters: (
            <div className="w-40">
              <Select
                value={statusFilter}
                onValueChange={(next) => {
                  setStatusFilter(next);
                  setPage(1);
                }}
                options={[
                  { value: 'ALL', label: 'All Statuses' },
                  { value: 'Active', label: 'Active' },
                  { value: 'Inactive', label: 'Inactive' },
                ]}
              />
            </div>
          ),
        }}
      />

      {/* User Create / Edit Dialog */}
      <AppDialog
        open={userDialogOpen}
        onOpenChange={setUserDialogOpen}
        title={editingUser ? `Edit ${editingUser.username}` : 'Create User'}
        size="md"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setUserDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              type="submit"
              form="admin-user-form"
              disabled={createUser.isPending || updateUser.isPending}
            >
              {editingUser ? 'Save Changes' : 'Create User'}
            </Button>
          </div>
        }
      >
        <form id="admin-user-form" onSubmit={userForm.handleSubmit(handleSaveUser)} className="space-y-4">
          <FormField
            label="Username"
            htmlFor="usr-username"
            error={userForm.formState.errors.username?.message}
            required
          >
            <Input id="usr-username" disabled={Boolean(editingUser)} {...userForm.register('username')} />
          </FormField>

          <FormField
            label="Full Name"
            htmlFor="usr-fullname"
            error={userForm.formState.errors.fullName?.message}
            required
          >
            <Input id="usr-fullname" {...userForm.register('fullName')} />
          </FormField>

          <FormField
            label="Email"
            htmlFor="usr-email"
            error={userForm.formState.errors.email?.message}
            required
          >
            <Input id="usr-email" type="email" {...userForm.register('email')} />
          </FormField>

          <FormField
            label="Default Organization"
            error={userForm.formState.errors.defaultOrganizationId?.message}
          >
            <Controller
              control={userForm.control}
              name="defaultOrganizationId"
              render={({ field }) => (
                <Combobox
                  value={field.value ?? ''}
                  onValueChange={field.onChange}
                  allowClear
                  options={organizations.map((o) => ({
                    value: o.id,
                    label: `${o.code} — ${o.name}`,
                  }))}
                  placeholder="Select default organization..."
                />
              )}
            />
          </FormField>
        </form>
      </AppDialog>

      {/* Disable Confirmation */}
      <AppDialog
        open={Boolean(disableTarget)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDisableTarget(null);
        }}
        title="Disable User Account"
        description="Disabled accounts are set to Inactive and blocked from future authentication."
        size="sm"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setDisableTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleConfirmDisable}
              disabled={disableUser.isPending}
            >
              Confirm Disable
            </Button>
          </div>
        }
      >
        {disableTarget && (
          <div className="rounded-lg border border-rose-500/20 bg-rose-500/5 p-3 text-sm font-mono">
            {disableTarget.username} ({disableTarget.email})
          </div>
        )}
      </AppDialog>

      {/* Assign Roles SideDrawer */}
      <SideDrawer
        open={Boolean(rolesDrawerUser)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setRolesDrawerUser(null);
        }}
        title={rolesDrawerUser ? `Roles: ${rolesDrawerUser.fullName}` : 'User Role Assignments'}
        description={rolesDrawerUser ? `${rolesDrawerUser.username} · ${rolesDrawerUser.email}` : undefined}
        size="lg"
      >
        {rolesDrawerUser && (
          <div className="space-y-5">
            {canWrite && (
              <div className="space-y-3 rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-subtle)] p-4">
                <div className="text-sm font-semibold text-[var(--text-primary)]">
                  Assign Role per Organization
                </div>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <FormField label="Role">
                    <Combobox
                      value={assignRoleId}
                      onValueChange={setAssignRoleId}
                      options={roles.map((r) => ({
                        value: r.id,
                        label: `${r.code} — ${r.name}`,
                      }))}
                      placeholder="Select role..."
                    />
                  </FormField>
                  <FormField label="Organization Scope">
                    <Combobox
                      value={assignOrgId}
                      onValueChange={setAssignOrgId}
                      allowClear
                      options={organizations.map((o) => ({
                        value: o.id,
                        label: `${o.code} — ${o.name}`,
                      }))}
                      placeholder="Active / Default organization..."
                    />
                  </FormField>
                </div>
                <div className="flex justify-end">
                  <Button
                    size="sm"
                    onClick={handleAssignRole}
                    disabled={!assignRoleId || assignRole.isPending}
                  >
                    Assign Role
                  </Button>
                </div>
              </div>
            )}

            {userRolesQuery.isLoading ? (
              <div className="h-24 animate-pulse rounded-xl bg-[var(--surface-muted)]" />
            ) : userRolesQuery.isError ? (
              <ErrorState
                error={userRolesQuery.error}
                onRetry={() => void userRolesQuery.refetch()}
              />
            ) : assignedRoles.length === 0 ? (
              <EmptyState title="No roles assigned to this user" />
            ) : (
              <div className="space-y-2">
                {assignedRoles.map((assignment) => (
                  <div
                    key={assignment.id}
                    className="flex items-center justify-between rounded-xl border border-[var(--border-subtle)] bg-[var(--surface-card)] px-4 py-3 text-sm"
                  >
                    <div>
                      <div className="font-medium text-[var(--text-primary)]">
                        {roleMap.get(assignment.roleId) ?? assignment.roleId}
                      </div>
                      <div className="font-mono text-xs text-[var(--text-muted)]">
                        Org: {orgMap.get(assignment.organizationRefId) ?? assignment.organizationRefId}
                      </div>
                    </div>
                    {canWrite && (
                      <Button
                        size="sm"
                        variant="danger"
                        aria-label={`Revoke role ${assignment.roleId}`}
                        onClick={() => handleRevokeRole(assignment.roleId)}
                        disabled={revokeRole.isPending}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </SideDrawer>
    </div>
  );
}
