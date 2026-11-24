import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface WorkflowTask {
  id: string;
  organizationId: string;
  instanceId: string;
  assigneeId: string;
  decision?: string | null;
  documentRef: string;
  instanceStatus: string;
  createdBy: string;
  rowVersion: number;
}

export interface WorkflowInstance {
  id: string;
  organizationId: string;
  definitionId: string;
  documentRef: string;
  status: string;
  currentStep?: number | null;
  rowVersion: number;
}

export interface WorkflowInstanceDetail extends WorkflowInstance {
  createdBy?: string | null;
  tasks: WorkflowTask[];
}

export interface WorkflowDefinition {
  id: string;
  organizationId: string;
  code: string;
  documentType: string;
  version: number;
  isActive: boolean;
  rowVersion: number;
}

export interface TaskDecisionResult {
  taskId: string;
  instanceId: string;
  documentRef: string;
  decision: string;
  instanceStatus: string;
  taskRowVersion: number;
  instanceRowVersion: number;
}

export interface WorkflowDefinitionStepInput {
  stepOrder: number;
  approverRule: Record<string, unknown>;
  escalationRule?: Record<string, unknown>;
}

export const createDefinitionSchema = z.object({
  code: z.string().trim().min(1, 'Definition code is required').max(64),
  documentType: z.string().trim().min(1, 'Document type is required').max(64),
  approverRole: z.string().trim().max(64).optional().or(z.literal('')),
  slaHours: z.number().int().min(1).max(720),
});

export type CreateDefinitionValues = z.infer<typeof createDefinitionSchema>;

export const createInstanceSchema = z.object({
  definitionId: z.string().trim().min(1, 'Workflow definition is required'),
  documentRef: z.string().trim().min(1, 'Document reference is required').max(120),
  assigneeId: z.string().trim().min(1, 'Assignee user ID is required (must differ from requester)').max(64),
});

export type CreateInstanceValues = z.infer<typeof createInstanceSchema>;

export const workflowKeys = {
  all: ['workflow'] as const,
  tasks: (orgId: string | undefined, status: string, page: number) =>
    ['workflow', 'tasks', orgId, status, page] as const,
  instances: (orgId: string | undefined, status: string, page: number) =>
    ['workflow', 'instances', orgId, status, page] as const,
  instanceDetail: (orgId: string | undefined, id: string | null) =>
    ['workflow', 'instance-detail', orgId, id] as const,
  definitions: (orgId: string | undefined, docType: string, page: number) =>
    ['workflow', 'definitions', orgId, docType, page] as const,
};

function buildQuery(params: Record<string, string | number | undefined>): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v));
    }
  }
  return sp;
}

export function useWorkflowTasksQuery(
  orgId: string | undefined,
  statusFilter: string,
  page: number,
  size = 25,
) {
  return useQuery<{ items: WorkflowTask[]; page: PageMeta }>({
    queryKey: workflowKeys.tasks(orgId, statusFilter, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<WorkflowTask> | WorkflowTask[]>(
        '/workflow-tasks',
        {
          query: buildQuery({
            page,
            size,
            status: statusFilter && statusFilter !== 'ALL' ? statusFilter : undefined,
          }),
        },
      );
      return unwrapListResponse<WorkflowTask>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useWorkflowInstancesQuery(
  orgId: string | undefined,
  statusFilter: string,
  page: number,
  size = 25,
) {
  return useQuery<{ items: WorkflowInstance[]; page: PageMeta }>({
    queryKey: workflowKeys.instances(orgId, statusFilter, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<WorkflowInstance> | WorkflowInstance[]>(
        '/workflow-instances',
        {
          query: buildQuery({
            page,
            size,
            status: statusFilter && statusFilter !== 'ALL' ? statusFilter : undefined,
          }),
        },
      );
      return unwrapListResponse<WorkflowInstance>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useWorkflowInstanceDetailQuery(
  orgId: string | undefined,
  instanceId: string | null,
) {
  return useQuery<WorkflowInstanceDetail>({
    queryKey: workflowKeys.instanceDetail(orgId, instanceId),
    queryFn: () => apiClient<WorkflowInstanceDetail>(`/workflow-instances/${instanceId}`),
    enabled: Boolean(orgId && instanceId),
  });
}

export function useWorkflowDefinitionsQuery(
  orgId: string | undefined,
  documentType: string,
  page: number,
  size = 50,
) {
  return useQuery<{ items: WorkflowDefinition[]; page: PageMeta }>({
    queryKey: workflowKeys.definitions(orgId, documentType, page),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<WorkflowDefinition> | WorkflowDefinition[]>(
        '/workflow-definitions',
        {
          query: buildQuery({
            page,
            size,
            documentType: documentType.trim() ? documentType.trim() : undefined,
          }),
        },
      );
      return unwrapListResponse<WorkflowDefinition>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useApproveWorkflowTaskMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      comment,
    }: {
      id: string;
      rowVersion: number;
      comment?: string;
    }) =>
      apiClient<TaskDecisionResult>(`/workflow-tasks/${id}/approve`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: comment?.trim() ? JSON.stringify({ comment: comment.trim() }) : undefined,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: workflowKeys.all });
    },
  });
}

export function useRejectWorkflowTaskMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      rowVersion,
      comment,
    }: {
      id: string;
      rowVersion: number;
      comment?: string;
    }) =>
      apiClient<TaskDecisionResult>(`/workflow-tasks/${id}/reject`, {
        method: 'POST',
        ifMatch: String(rowVersion),
        idempotencyKey: crypto.randomUUID(),
        body: comment?.trim() ? JSON.stringify({ comment: comment.trim() }) : undefined,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: workflowKeys.all });
    },
  });
}

export function useCreateWorkflowDefinitionMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateDefinitionValues) => {
      const steps: WorkflowDefinitionStepInput[] = [
        {
          stepOrder: 1,
          approverRule: {
            role: values.approverRole?.trim() || 'FINANCE_APPROVER',
            enforceSod: true,
          },
          escalationRule: {
            slaHours: values.slaHours,
          },
        },
      ];
      return apiClient<WorkflowDefinition>('/workflow-definitions', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          code: values.code.trim(),
          documentType: values.documentType.trim(),
          steps,
        }),
      });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: workflowKeys.all });
    },
  });
}

export function useCreateWorkflowInstanceMutation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (values: CreateInstanceValues) =>
      apiClient<WorkflowInstance>('/workflow-instances', {
        method: 'POST',
        idempotencyKey: crypto.randomUUID(),
        body: JSON.stringify({
          definitionId: values.definitionId.trim(),
          documentRef: values.documentRef.trim(),
          assigneeId: values.assigneeId.trim(),
        }),
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: workflowKeys.all });
    },
  });
}
