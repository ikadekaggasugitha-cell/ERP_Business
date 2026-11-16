import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  BomDetailRecord,
  BomRecord,
  CompleteWorkOrderPayload,
  CreateBomPayload,
  ManufacturingItemLookupRecord,
  UpsertWorkOrderPayload,
  WorkOrderRecord,
} from './types';

export interface ListWorkOrdersParams {
  page?: number;
  size?: number;
  status?: string;
}

export async function listBoms(page = 1, size = 50): Promise<{
  items: BomRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  const raw = await apiClient<ListEnvelope<BomRecord> | BomRecord[]>('/bill-of-materials', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function getBom(id: string): Promise<BomDetailRecord> {
  return apiClient<BomDetailRecord>(`/bill-of-materials/${encodeURIComponent(id)}`);
}

export async function createBom(payload: CreateBomPayload): Promise<BomDetailRecord> {
  return apiClient<BomDetailRecord>('/bill-of-materials', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function listWorkOrders(params: ListWorkOrdersParams = {}): Promise<{
  items: WorkOrderRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<ListEnvelope<WorkOrderRecord> | WorkOrderRecord[]>('/work-orders', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function createWorkOrder(payload: UpsertWorkOrderPayload): Promise<WorkOrderRecord> {
  return apiClient<WorkOrderRecord>('/work-orders', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function updateWorkOrder(
  id: string,
  rowVersion: number,
  payload: UpsertWorkOrderPayload
): Promise<WorkOrderRecord> {
  return apiClient<WorkOrderRecord>(`/work-orders/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    ifMatch: String(rowVersion),
  });
}

export async function deleteWorkOrder(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/work-orders/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function releaseWorkOrder(id: string, rowVersion: number): Promise<WorkOrderRecord> {
  return apiClient<WorkOrderRecord>(`/work-orders/${encodeURIComponent(id)}/release`, {
    method: 'POST',
    ifMatch: String(rowVersion),
  });
}

export async function completeWorkOrder(
  id: string,
  rowVersion: number,
  payload: CompleteWorkOrderPayload
): Promise<WorkOrderRecord> {
  return apiClient<WorkOrderRecord>(`/work-orders/${encodeURIComponent(id)}/complete`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    body: JSON.stringify(payload),
  });
}

export async function listManufacturingItemLookups(): Promise<ManufacturingItemLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<
      ListEnvelope<ManufacturingItemLookupRecord> | ManufacturingItemLookupRecord[]
    >('/items', { query });
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}
