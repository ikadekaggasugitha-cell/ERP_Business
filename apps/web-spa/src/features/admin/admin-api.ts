import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { apiClient, resolveApiBaseUrl, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface OrgType {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  levelHint?: number | null;
  isActive: boolean;
  rowVersion: number;
}

export interface OrgRecord {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  organizationTypeId: string;
  parentId?: string | null;
  path: string;
  status: string;
  effectiveFrom: string;
  effectiveTo?: string | null;
  descendantCount?: number;
  depth?: number;
  rowVersion: number;
}

export interface OrgTreeNode {
  node: OrgRecord;
  children: OrgTreeNode[];
}

export interface AdminUser {
  id: string;
  organizationId: string;
  username: string;
  email: string;
  fullName: string;
  status: string;
  defaultOrganizationId?: string | null;
  rowVersion: number;
}

export interface UserRoleAssignment {
  id: string;
  userId: string;
  organizationRefId: string;
  roleId: string;
  rowVersion: number;
}

export interface AdminUserDetail extends AdminUser {
  roles: UserRoleAssignment[];
}

export interface SecurityRole {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  isSystem: boolean;
  rowVersion: number;
}

export interface SecurityPermission {
  id: string;
  organizationId?: string;
  code: string;
  module: string;
  action: string;
  rowVersion?: number;
}

export interface SecurityRoleDetail extends SecurityRole {
  permissions: SecurityPermission[];
}

export interface GatewayHealthStatus {
  status: 'online' | 'degraded' | 'offline';
  httpStatus?: number;
  endpoint: string;
  checkedAt: string;
  detail?: string;
}

const isoDateRegex = /^\d{4}-\d{2}-\d{2}$/;

export const createOrgTypeSchema = z.object({
  code: z.string().trim().min(1, 'Type code is required').max(64),
  name: z.string().trim().min(1, 'Type name is required').max(150),
  levelHint: z.number().int().min(0).max(99).optional(),
});
export type CreateOrgTypeValues = z.infer<typeof createOrgTypeSchema>;

export const orgFormSchema = z.object({
  code: z.string().trim().min(1, 'Code is required').max(64),
  name: z.string().trim().min(1, 'Name is required').max(200),
  organizationTypeId: z.string().trim().min(1, 'Organization type is required'),
  parentId: z.string().trim().optional().or(z.literal('')),
  effectiveFrom: z
    .string()
    .trim()
    .refine((v) => v === '' || isoDateRegex.test(v), 'Use YYYY-MM-DD')
    .optional()
    .or(z.literal('')),
  effectiveTo: z
    .string()
    .trim()
    .refine((v) => v === '' || isoDateRegex.test(v), 'Use YYYY-MM-DD')
    .optional()
    .or(z.literal('')),
});
export type OrgFormValues = z.infer<typeof orgFormSchema>;

export const userFormSchema = z.object({
  username: z.string().trim().min(2, 'Username >= 2 chars').max(64),
  email: z.string().trim().email('Valid email required'),
  fullName: z.string().trim().min(1, 'Full name required').max(200),
  defaultOrganizationId: z.string().trim().optional().or(z.literal('')),
});
export type UserFormValues = z.infer<typeof userFormSchema>;

export const roleFormSchema = z.object({
  code: z.string().trim().min(1, 'Role code required').max(64),
  name: z.string().trim().min(1, 'Role name required').max(150),
  isSystem: z.boolean(),
});
export type RoleFormValues = z.infer<typeof roleFormSchema>;

export const permissionFormSchema = z.object({
  code: z.string().trim().min(1, 'Permission code required').max(100),
  module: z.string().trim().min(1, 'Module required').max(64),
  action: z.string().trim().min(1, 'Action required').max(64),
});
export type PermissionFormValues = z.infer<typeof permissionFormSchema>;

export const adminKeys = {
  orgAll: ['admin', 'org'] as const,
  orgTypes: (orgId: string | undefined) => ['admin', 'org-types', orgId] as const,
  orgs: (orgId: string | undefined, resource: 'organizations' | 'org-units', status: string, page: number) =>
    ['admin', 'orgs', orgId, resource, status, page] as const,
  orgHierarchy: (
    orgId: string | undefined,
    resource: 'organizations' | 'org-units',
    rootId: string | null,
  ) => ['admin', 'org-hierarchy', orgId, resource, rootId] as const,
  userAll: ['admin', 'users'] as const,
  users: (orgId: string | undefined, status: string, page: number) =>
    ['admin', 'users', orgId, status, page] as const,
  userRoles: (orgId: string | undefined, userId: string | null) =>
    ['admin', 'user-roles', orgId, userId] as const,
  securityAll: ['admin', 'security'] as const,
  roles: (orgId: string | undefined, page: number) => ['admin', 'roles', orgId, page] as const,
  rolePermissions: (orgId: string | undefined, roleId: string | null) =>
    ['admin', 'role-permissions', orgId, roleId] as const,
  permissions: (orgId: string | undefined, moduleFilter: string) =>
    ['admin', 'permissions', orgId, moduleFilter] as const,
};

function buildQuery(params: Record<string, string | number | boolean | undefined>): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v));
    }
  }
  return sp;
}

function cleanOptional(val: string | undefined): string | undefined {
  const t = val?.trim();
  return t ? t : undefined;
}

export function buildOrgHierarchyTree(records: OrgRecord[]): OrgTreeNode[] {
  const byId = new Map<string, OrgTreeNode>();
  for (const r of records) {
    byId.set(r.id, { node: r, children: [] });
  }
  const roots: OrgTreeNode[] = [];
  for (const r of records) {
    const entry = byId.get(r.id);
    if (!entry) continue;
    const parent = r.parentId ? byId.get(r.parentId) : undefined;
    if (parent && parent.node.id !== r.id) {
      parent.children.push(entry);
    } else {
      roots.push(entry);
    }
  }
  const sortRecursive = (nodes: OrgTreeNode[]) => {
    nodes.sort((a, b) => a.node.code.localeCompare(b.node.code));
    for (const n of nodes) sortRecursive(n.children);
  };
  sortRecursive(roots);
  return roots;
}

/* ---------------- Organization Queries & Mutations ---------------- */

export function useOrganizationTypesQuery(orgId: string | undefined) {
  return useQuery<{ items: OrgType[]; page: PageMeta }>({
    queryKey: adminKeys.orgTypes(orgId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<OrgType> | OrgType[]>(
        '/organization-types',
        { query: buildQuery({ page: 1, size: 100 }) },
      );
      return unwrapListResponse<OrgType>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useOrganizationsQuery(
  orgId: string | undefined,
  resource: 'organizations' | 'org-units',
  status: string,
  page: number,
  size = 100,
) {
  return useQuery<{ items: OrgRecord[]; page: PageMeta }>({
    queryKey: adminKeys.orgs(orgId, resource, status, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<OrgRecord> | OrgRecord[]>(`/${resource}`, {
        query: buildQuery({
          page,
          size,
          status: status && status !== 'ALL' ? status : undefined,
        }),
      });
      return unwrapListResponse<OrgRecord>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useOrgHierarchyQuery(
  orgId: string | undefined,
  resource: 'organizations' | 'org-units',
  rootId: string | null,
) {
  return useQuery<{ items: OrgRecord[]; page: PageMeta }>({
    queryKey: adminKeys.orgHierarchy(orgId, resource, rootId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<OrgRecord> | OrgRecord[]>(
        `/${resource}/${rootId}/hierarchy`,
      );
      return unwrapListResponse<OrgRecord>(payload);
    },
    enabled: Boolean(orgId && rootId),
  });
}

export function useCreateOrgTypeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateOrgTypeValues) =>
      apiClient<OrgType>('/organization-types', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          name: values.name.trim(),
          levelHint: values.levelHint,
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.orgAll });
    },
  });
}

export function useCreateOrgMutation(resource: 'organizations' | 'org-units') {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: OrgFormValues) =>
      apiClient<OrgRecord>(`/${resource}`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          name: values.name.trim(),
          organizationTypeId: values.organizationTypeId.trim(),
          parentId: cleanOptional(values.parentId),
          effectiveFrom: cleanOptional(values.effectiveFrom),
          effectiveTo: cleanOptional(values.effectiveTo),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.orgAll });
    },
  });
}

export function useUpdateOrgMutation(resource: 'organizations' | 'org-units') {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      values,
    }: {
      id: string;
      rowVersion: number;
      values: OrgFormValues;
    }) =>
      apiClient<OrgRecord>(`/${resource}/${id}`, {
        method: 'PUT',
        ifMatch: String(rowVersion),
        body: JSON.stringify({
          name: values.name.trim(),
          organizationTypeId: values.organizationTypeId.trim(),
          effectiveFrom:
            cleanOptional(values.effectiveFrom) ?? new Date().toISOString().slice(0, 10),
          effectiveTo: cleanOptional(values.effectiveTo),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.orgAll });
    },
  });
}

export function useActivateOrgMutation(resource: 'organizations' | 'org-units') {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<OrgRecord>(`/${resource}/${id}/activate`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.orgAll });
    },
  });
}

export function useDeleteOrgMutation(resource: 'organizations' | 'org-units') {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<void>(`/${resource}/${id}`, {
        method: 'DELETE',
        ifMatch: String(rowVersion),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.orgAll });
    },
  });
}

/* ---------------- User Queries & Mutations ---------------- */

export function useUsersQuery(orgId: string | undefined, status: string, page: number, size = 25) {
  return useQuery<{ items: AdminUser[]; page: PageMeta }>({
    queryKey: adminKeys.users(orgId, status, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<AdminUser> | AdminUser[]>('/users', {
        query: buildQuery({
          page,
          size,
          status: status && status !== 'ALL' ? status : undefined,
        }),
      });
      return unwrapListResponse<AdminUser>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useUserRolesQuery(orgId: string | undefined, userId: string | null) {
  return useQuery<UserRoleAssignment[]>({
    queryKey: adminKeys.userRoles(orgId, userId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<UserRoleAssignment> | UserRoleAssignment[]>(
        `/users/${userId}/roles`,
      );
      return unwrapListResponse<UserRoleAssignment>(payload).items;
    },
    enabled: Boolean(orgId && userId),
  });
}

export function useCreateUserMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: UserFormValues) =>
      apiClient<AdminUser>('/users', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          username: values.username.trim(),
          email: values.email.trim(),
          fullName: values.fullName.trim(),
          defaultOrganizationId: cleanOptional(values.defaultOrganizationId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.userAll });
    },
  });
}

export function useUpdateUserMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      values,
    }: {
      id: string;
      rowVersion: number;
      values: UserFormValues;
    }) =>
      apiClient<AdminUser>(`/users/${id}`, {
        method: 'PUT',
        ifMatch: String(rowVersion),
        body: JSON.stringify({
          email: values.email.trim(),
          fullName: values.fullName.trim(),
          defaultOrganizationId: cleanOptional(values.defaultOrganizationId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.userAll });
    },
  });
}

export function useDisableUserMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<AdminUser>(`/users/${id}/disable`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.userAll });
    },
  });
}

export function useAssignUserRoleMutation(userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId, organizationRefId }: { roleId: string; organizationRefId?: string }) =>
      apiClient<UserRoleAssignment>(`/users/${userId}/roles`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          roleId,
          organizationRefId: cleanOptional(organizationRefId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.userAll });
    },
  });
}

export function useRevokeUserRoleMutation(userId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId }: { roleId: string }) =>
      apiClient<void>(`/users/${userId}/roles/${roleId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.userAll });
    },
  });
}

/* ---------------- Security RBAC Queries & Mutations ---------------- */

export function useRolesQuery(orgId: string | undefined, page = 1, size = 100) {
  return useQuery<{ items: SecurityRole[]; page: PageMeta }>({
    queryKey: adminKeys.roles(orgId, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<SecurityRole> | SecurityRole[]>('/roles', {
        query: buildQuery({ page, size }),
      });
      return unwrapListResponse<SecurityRole>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function usePermissionsQuery(orgId: string | undefined, moduleFilter = '') {
  return useQuery<{ items: SecurityPermission[]; page: PageMeta }>({
    queryKey: adminKeys.permissions(orgId, moduleFilter),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<SecurityPermission> | SecurityPermission[]>(
        '/permissions',
        {
          query: buildQuery({
            page: 1,
            size: 200,
            module: moduleFilter.trim() || undefined,
          }),
        },
      );
      return unwrapListResponse<SecurityPermission>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useRolePermissionsQuery(orgId: string | undefined, roleId: string | null) {
  return useQuery<SecurityPermission[]>({
    queryKey: adminKeys.rolePermissions(orgId, roleId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<SecurityPermission> | SecurityPermission[]>(
        `/roles/${roleId}/permissions`,
      );
      return unwrapListResponse<SecurityPermission>(payload).items;
    },
    enabled: Boolean(orgId && roleId),
  });
}

export function useCreateRoleMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: RoleFormValues) =>
      apiClient<SecurityRole>('/roles', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          name: values.name.trim(),
          isSystem: values.isSystem,
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

export function useUpdateRoleMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      values,
    }: {
      id: string;
      rowVersion: number;
      values: RoleFormValues;
    }) =>
      apiClient<SecurityRole>(`/roles/${id}`, {
        method: 'PUT',
        ifMatch: String(rowVersion),
        body: JSON.stringify({
          name: values.name.trim(),
          isSystem: values.isSystem,
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

export function useDeleteRoleMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<void>(`/roles/${id}`, {
        method: 'DELETE',
        ifMatch: String(rowVersion),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

export function useCreatePermissionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: PermissionFormValues) =>
      apiClient<SecurityPermission>('/permissions', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          module: values.module.trim(),
          action: values.action.trim(),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

export function useGrantRolePermissionMutation(roleId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ permissionId }: { permissionId: string }) =>
      apiClient<{ roleId: string; permissionId: string }>(`/roles/${roleId}/permissions`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({ permissionId }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

export function useRevokeRolePermissionMutation(roleId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ permissionId }: { permissionId: string }) =>
      apiClient<void>(`/roles/${roleId}/permissions/${permissionId}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: adminKeys.securityAll });
    },
  });
}

/* ---------------- Gateway Health Probe ---------------- */

export async function probeGatewayHealth(): Promise<GatewayHealthStatus> {
  const baseApiUrl = resolveApiBaseUrl();
  const originRoot = baseApiUrl.replace(/\/api\/v1\/?$/, '');
  const endpoint = originRoot ? `${originRoot}/healthz` : '/healthz';
  const checkedAt = new Date().toISOString();

  try {
    const res = await fetch(endpoint, { method: 'GET', credentials: 'include' });
    if (res.ok) {
      return { status: 'online', httpStatus: res.status, endpoint, checkedAt };
    }
    return {
      status: 'degraded',
      httpStatus: res.status,
      endpoint,
      checkedAt,
      detail: `HTTP ${res.status}`,
    };
  } catch (err) {
    return {
      status: 'offline',
      endpoint,
      checkedAt,
      detail: err instanceof Error ? err.message : 'Unreachable',
    };
  }
}
