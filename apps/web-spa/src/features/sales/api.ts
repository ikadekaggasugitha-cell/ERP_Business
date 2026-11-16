import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  ConfirmSalesOrderPayload,
  CreateDeliveryPayload,
  CustomerLookupRecord,
  DeliveryDetailRecord,
  DeliveryRecord,
  SalesItemLookupRecord,
  SalesOrderDetailRecord,
  SalesOrderRecord,
  SalesWarehouseLookupRecord,
  UpsertSalesOrderPayload,
} from './types';

export interface ListSalesOrdersParams {
  page?: number;
  size?: number;
  status?: string;
}

export async function listSalesOrders(params: ListSalesOrdersParams = {}): Promise<{
  items: SalesOrderRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<ListEnvelope<SalesOrderRecord> | SalesOrderRecord[]>(
    '/sales-orders',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function getSalesOrder(id: string): Promise<SalesOrderDetailRecord> {
  return apiClient<SalesOrderDetailRecord>(`/sales-orders/${encodeURIComponent(id)}`);
}

export async function createSalesOrder(
  payload: UpsertSalesOrderPayload
): Promise<SalesOrderDetailRecord> {
  return apiClient<SalesOrderDetailRecord>('/sales-orders', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function updateSalesOrder(
  id: string,
  rowVersion: number,
  payload: UpsertSalesOrderPayload
): Promise<SalesOrderDetailRecord> {
  return apiClient<SalesOrderDetailRecord>(`/sales-orders/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    ifMatch: String(rowVersion),
  });
}

export async function deleteSalesOrder(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/sales-orders/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function confirmSalesOrder(
  id: string,
  rowVersion: number,
  payload: ConfirmSalesOrderPayload
): Promise<SalesOrderDetailRecord> {
  const body = payload.warehouseId ? JSON.stringify({ warehouseId: payload.warehouseId }) : undefined;
  return apiClient<SalesOrderDetailRecord>(`/sales-orders/${encodeURIComponent(id)}/confirm`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    body,
  });
}

export async function createDelivery(
  soId: string,
  rowVersion: number,
  payload: CreateDeliveryPayload
): Promise<DeliveryDetailRecord> {
  return apiClient<DeliveryDetailRecord>(
    `/sales-orders/${encodeURIComponent(soId)}/deliveries`,
    {
      method: 'POST',
      ifMatch: String(rowVersion),
      body: JSON.stringify(payload),
    }
  );
}

export async function listDeliveries(page = 1, size = 50): Promise<{
  items: DeliveryRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  const raw = await apiClient<ListEnvelope<DeliveryRecord> | DeliveryRecord[]>('/deliveries', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function getDelivery(id: string): Promise<DeliveryDetailRecord> {
  return apiClient<DeliveryDetailRecord>(`/deliveries/${encodeURIComponent(id)}`);
}

// Non-blocking lookup helpers for cross-service labels & dropdowns
export async function listCustomerLookups(): Promise<CustomerLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<ListEnvelope<CustomerLookupRecord> | CustomerLookupRecord[]>(
      '/parties',
      { query }
    );
    return unwrapListResponse(raw).items.filter(
      (p) => p.partyType === 'customer' || p.partyType === 'both'
    );
  } catch {
    return [];
  }
}

export async function listSalesItemLookups(): Promise<SalesItemLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<ListEnvelope<SalesItemLookupRecord> | SalesItemLookupRecord[]>(
      '/items',
      { query }
    );
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}

export async function listSalesWarehouseLookups(): Promise<SalesWarehouseLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<
      ListEnvelope<SalesWarehouseLookupRecord> | SalesWarehouseLookupRecord[]
    >('/warehouses', { query });
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}
