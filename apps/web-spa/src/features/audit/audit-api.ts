import { useQuery } from '@tanstack/react-query';
import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';

export interface AuditEventSummary {
  id: string;
  occurredAt: string;
  organizationId: string;
  actorId?: string | null;
  action: string;
  entityType: string;
  entityId?: string | null;
  correlationId?: string | null;
}

export interface AuditEventDetail extends AuditEventSummary {
  beforeValue?: unknown;
  afterValue?: unknown;
  reason?: string | null;
}

export interface AuditFilterState {
  entityType: string;
  action: string;
  actor: string;
  dateFrom: string;
  dateTo: string;
  page: number;
  size: number;
}

export type DiffStatus = 'added' | 'removed' | 'changed' | 'unchanged';

export interface ForensicDiffEntry {
  key: string;
  status: DiffStatus;
  beforeText: string;
  afterText: string;
}

export interface AuditQueryParams {
  page: number;
  size: number;
  entityType: string;
  action: string;
  from?: string;
  to?: string;
  entityId?: string;
  correlationId?: string;
}

export const auditKeys = {
  all: ['audit'] as const,
  list: (orgId: string | undefined, params: AuditQueryParams) =>
    ['audit', 'events', orgId, params] as const,
  detail: (orgId: string | undefined, id: string | null) =>
    ['audit', 'detail', orgId, id] as const,
};

function buildAuditQuery(params: Record<string, string | number | undefined>): URLSearchParams {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') {
      sp.set(k, String(v));
    }
  }
  return sp;
}

export function useAuditEventsQuery(
  orgId: string | undefined,
  params: AuditQueryParams,
) {
  return useQuery<{ items: AuditEventSummary[]; page: PageMeta }>({
    queryKey: auditKeys.list(orgId, params),
    queryFn: async () => {
      const payload = await apiClient<ListEnvelope<AuditEventSummary> | AuditEventSummary[]>(
        '/audit-events',
        {
          query: buildAuditQuery({
            page: params.page,
            size: params.size,
            entityType: params.entityType.trim() || undefined,
            action: params.action.trim() || undefined,
            from: params.from?.trim() || undefined,
            to: params.to?.trim() || undefined,
            entityId: params.entityId?.trim() || undefined,
            correlationId: params.correlationId?.trim() || undefined,
          }),
        },
      );
      return unwrapListResponse<AuditEventSummary>(payload);
    },
    enabled: Boolean(orgId),
  });
}

export function useAuditEventDetailQuery(orgId: string | undefined, eventId: string | null) {
  return useQuery<AuditEventDetail>({
    queryKey: auditKeys.detail(orgId, eventId),
    queryFn: () => apiClient<AuditEventDetail>(`/audit-events/${eventId}`),
    enabled: Boolean(orgId && eventId),
  });
}

function formatDiffCell(value: unknown): string {
  if (value === undefined) return '—';
  if (value === null) return 'null';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function asPlainObject(value: unknown): Record<string, unknown> | null {
  if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return null;
}

export function computeForensicDiff(beforeValue: unknown, afterValue: unknown): ForensicDiffEntry[] {
  const beforeObj = asPlainObject(beforeValue);
  const afterObj = asPlainObject(afterValue);

  if (!beforeObj && !afterObj) {
    if (beforeValue === undefined && afterValue === undefined) return [];
    const beforeText = formatDiffCell(beforeValue);
    const afterText = formatDiffCell(afterValue);
    return [
      {
        key: '$root',
        status: beforeText === afterText ? 'unchanged' : 'changed',
        beforeText,
        afterText,
      },
    ];
  }

  const keys = Array.from(
    new Set([...Object.keys(beforeObj ?? {}), ...Object.keys(afterObj ?? {})]),
  ).sort();

  return keys.map((key) => {
    const hasBefore = beforeObj !== null && Object.prototype.hasOwnProperty.call(beforeObj, key);
    const hasAfter = afterObj !== null && Object.prototype.hasOwnProperty.call(afterObj, key);
    const bVal = hasBefore ? beforeObj[key] : undefined;
    const aVal = hasAfter ? afterObj[key] : undefined;
    const beforeText = formatDiffCell(bVal);
    const afterText = formatDiffCell(aVal);

    let status: DiffStatus = 'unchanged';
    if (!hasBefore && hasAfter) {
      status = 'added';
    } else if (hasBefore && !hasAfter) {
      status = 'removed';
    } else if (beforeText !== afterText) {
      status = 'changed';
    }

    return {
      key,
      status,
      beforeText,
      afterText,
    };
  });
}
