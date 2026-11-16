import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { toast } from 'sonner';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface HrListResult<T> {
  items: T[];
  page: PageMeta;
}

export interface HrEmployee {
  id: string;
  organizationId: string;
  employeeNo: string;
  fullName: string;
  nationalIdMasked?: string | null;
  dateOfBirth?: string | null;
  email?: string | null;
  hireDate: string;
  status: string;
  userRefId?: string | null;
  rowVersion: number;
}

export interface HrPosition {
  id: string;
  organizationId: string;
  code: string;
  title: string;
  organizationRefId: string;
  parentPositionId?: string | null;
  rowVersion: number;
}

export interface HrEmployment {
  id: string;
  employeeId: string;
  positionId: string;
  employmentType: string;
  effectiveFrom: string;
  effectiveTo?: string | null;
  rowVersion: number;
}

export interface HrCompensation {
  id: string;
  employeeId: string;
  componentCode: string;
  amount: string;
  effectiveFrom: string;
  rowVersion: number;
}

export interface EmployeeListParams {
  page?: number;
  size?: number;
  status?: string;
  nationalId?: string;
}

const isoDateRegex = /^\d{4}-\d{2}-\d{2}$/;
const decimalAmountRegex = /^\d+(\.\d{1,6})?$/;

export const createEmployeeSchema = z.object({
  employeeNo: z.string().trim().min(1, 'Employee number is required').max(64),
  fullName: z.string().trim().min(1, 'Full name is required').max(200),
  nationalId: z.string().trim().max(64).optional().or(z.literal('')),
  dateOfBirth: z
    .string()
    .trim()
    .refine((v) => v === '' || isoDateRegex.test(v), 'Use YYYY-MM-DD format')
    .optional()
    .or(z.literal('')),
  email: z
    .string()
    .trim()
    .refine((v) => v === '' || z.string().email().safeParse(v).success, 'Invalid email address')
    .optional()
    .or(z.literal('')),
  hireDate: z.string().trim().regex(isoDateRegex, 'Hire date must be YYYY-MM-DD'),
  userRefId: z.string().trim().max(64).optional().or(z.literal('')),
});

export type CreateEmployeeValues = z.infer<typeof createEmployeeSchema>;

export const updateEmployeeSchema = createEmployeeSchema.omit({
  employeeNo: true,
  hireDate: true,
});

export type UpdateEmployeeValues = z.infer<typeof updateEmployeeSchema>;

export const createPositionSchema = z.object({
  code: z.string().trim().min(1, 'Position code is required').max(64),
  title: z.string().trim().min(1, 'Position title is required').max(200),
  organizationRefId: z.string().trim().max(64).optional().or(z.literal('')),
  parentPositionId: z.string().trim().max(64).optional().or(z.literal('')),
});

export type CreatePositionValues = z.infer<typeof createPositionSchema>;

export const createEmploymentSchema = z.object({
  positionId: z.string().trim().min(1, 'Position is required'),
  employmentType: z.string().trim().min(1, 'Employment type is required').max(64),
  effectiveFrom: z.string().trim().regex(isoDateRegex, 'Use YYYY-MM-DD format'),
  effectiveTo: z
    .string()
    .trim()
    .refine((v) => v === '' || isoDateRegex.test(v), 'Use YYYY-MM-DD format')
    .optional()
    .or(z.literal('')),
});

export type CreateEmploymentValues = z.infer<typeof createEmploymentSchema>;

export const createCompensationSchema = z.object({
  componentCode: z.string().trim().min(1, 'Component code is required').max(64),
  amount: z
    .string()
    .trim()
    .regex(decimalAmountRegex, 'Amount must be a non-negative decimal string'),
  effectiveFrom: z.string().trim().regex(isoDateRegex, 'Use YYYY-MM-DD format'),
});

export type CreateCompensationValues = z.infer<typeof createCompensationSchema>;

export const hrKeys = {
  all: ['hr'] as const,
  employees: (orgId: string | undefined, params: EmployeeListParams) =>
    ['hr', 'employees', orgId, params] as const,
  employee: (orgId: string | undefined, id: string | null) =>
    ['hr', 'employee', orgId, id] as const,
  employments: (orgId: string | undefined, employeeId: string | null) =>
    ['hr', 'employments', orgId, employeeId] as const,
  compensations: (orgId: string | undefined, employeeId: string | null) =>
    ['hr', 'compensations', orgId, employeeId] as const,
  positions: (orgId: string | undefined, page: number, size: number) =>
    ['hr', 'positions', orgId, page, size] as const,
};

function cleanOptional(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

export function buildHrQuery(params: Record<string, string | number | undefined>): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v));
    }
  }
  return sp;
}

export function useEmployeesQuery(orgId: string | undefined, params: EmployeeListParams) {
  return useQuery<HrListResult<HrEmployee>>({
    queryKey: hrKeys.employees(orgId, params),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<HrEmployee> | HrEmployee[]>('/employees', {
        query: buildHrQuery({
          page: params.page ?? 1,
          size: params.size ?? 25,
          status: params.status && params.status !== 'ALL' ? params.status : undefined,
          nationalId: cleanOptional(params.nationalId),
        }),
      });
      return unwrapListResponse<HrEmployee>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useEmployeeDetailQuery(orgId: string | undefined, employeeId: string | null) {
  return useQuery<HrEmployee>({
    queryKey: hrKeys.employee(orgId, employeeId),
    queryFn: () => apiClient<HrEmployee>(`/employees/${employeeId}`),
    enabled: Boolean(orgId && employeeId),
  });
}

export function usePositionsQuery(orgId: string | undefined, page = 1, size = 100) {
  return useQuery<HrListResult<HrPosition>>({
    queryKey: hrKeys.positions(orgId, page, size),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<HrPosition> | HrPosition[]>('/positions', {
        query: buildHrQuery({ page, size }),
      });
      return unwrapListResponse<HrPosition>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useEmployeeEmploymentsQuery(orgId: string | undefined, employeeId: string | null) {
  return useQuery<HrListResult<HrEmployment>>({
    queryKey: hrKeys.employments(orgId, employeeId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<HrEmployment> | HrEmployment[]>(
        `/employees/${employeeId}/employments`,
      );
      return unwrapListResponse<HrEmployment>(payload);
    },
    enabled: Boolean(orgId && employeeId),
  });
}

export function useEmployeeCompensationsQuery(orgId: string | undefined, employeeId: string | null) {
  return useQuery<HrListResult<HrCompensation>>({
    queryKey: hrKeys.compensations(orgId, employeeId),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<HrCompensation> | HrCompensation[]>(
        `/employees/${employeeId}/compensations`,
      );
      return unwrapListResponse<HrCompensation>(payload);
    },
    enabled: Boolean(orgId && employeeId),
  });
}

export function useCreateEmployeeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateEmployeeValues) =>
      apiClient<HrEmployee>('/employees', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          employeeNo: values.employeeNo.trim(),
          fullName: values.fullName.trim(),
          nationalId: cleanOptional(values.nationalId),
          dateOfBirth: cleanOptional(values.dateOfBirth),
          email: cleanOptional(values.email),
          hireDate: values.hireDate.trim(),
          userRefId: cleanOptional(values.userRefId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}

export function useUpdateEmployeeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      values,
    }: {
      id: string;
      rowVersion: number;
      values: UpdateEmployeeValues;
    }) =>
      apiClient<HrEmployee>(`/employees/${id}`, {
        method: 'PUT',
        ifMatch: String(rowVersion),
        body: JSON.stringify({
          fullName: values.fullName.trim(),
          nationalId: cleanOptional(values.nationalId),
          dateOfBirth: cleanOptional(values.dateOfBirth),
          email: cleanOptional(values.email),
          userRefId: cleanOptional(values.userRefId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}

export function useTerminateEmployeeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<HrEmployee>(`/employees/${id}/terminate`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
      toast.success('Employee terminated');
    },
  });
}

export function useDeleteEmployeeMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, rowVersion }: { id: string; rowVersion: number }) =>
      apiClient<void>(`/employees/${id}`, {
        method: 'DELETE',
        ifMatch: String(rowVersion),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}

export function useCreatePositionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreatePositionValues) =>
      apiClient<HrPosition>('/positions', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          title: values.title.trim(),
          organizationRefId: cleanOptional(values.organizationRefId),
          parentPositionId: cleanOptional(values.parentPositionId),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}

export function useCreateEmploymentMutation(employeeId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateEmploymentValues) =>
      apiClient<HrEmployment>(`/employees/${employeeId}/employments`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          positionId: values.positionId.trim(),
          employmentType: values.employmentType.trim(),
          effectiveFrom: values.effectiveFrom.trim(),
          effectiveTo: cleanOptional(values.effectiveTo),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}

export function useCreateCompensationMutation(employeeId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateCompensationValues) =>
      apiClient<HrCompensation>(`/employees/${employeeId}/compensations`, {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          componentCode: values.componentCode.trim(),
          amount: values.amount.trim(),
          effectiveFrom: values.effectiveFrom.trim(),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: hrKeys.all });
    },
  });
}
