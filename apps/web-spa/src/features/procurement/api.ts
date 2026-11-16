import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  CreatePurchaseRequisitionPayload,
  GoodsReceiptDetailRecord,
  GoodsReceiptRecord,
  ProcurementItemLookupRecord,
  PurchaseOrderDetailRecord,
  PurchaseOrderRecord,
  PurchaseRequisitionDetailRecord,
  PurchaseRequisitionRecord,
  ReceivePurchaseOrderPayload,
  UpsertPurchaseOrderPayload,
  VendorLookupRecord,
  WarehouseLookupRecord,
} from './types';

export interface ListPurchaseOrdersParams {
  page?: number;
  size?: number;
  status?: string;
}

export async function listPurchaseOrders(params: ListPurchaseOrdersParams = {}): Promise<{
  items: PurchaseOrderRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<ListEnvelope<PurchaseOrderRecord> | PurchaseOrderRecord[]>(
    '/purchase-orders',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function getPurchaseOrder(id: string): Promise<PurchaseOrderDetailRecord> {
  return apiClient<PurchaseOrderDetailRecord>(`/purchase-orders/${encodeURIComponent(id)}`);
}

export async function createPurchaseOrder(
  payload: UpsertPurchaseOrderPayload
): Promise<PurchaseOrderRecord> {
  return apiClient<PurchaseOrderRecord>('/purchase-orders', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function updatePurchaseOrder(
  id: string,
  rowVersion: number,
  payload: UpsertPurchaseOrderPayload
): Promise<PurchaseOrderRecord> {
  return apiClient<PurchaseOrderRecord>(`/purchase-orders/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    ifMatch: String(rowVersion),
  });
}

export async function deletePurchaseOrder(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/purchase-orders/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export async function approvePurchaseOrder(
  id: string,
  rowVersion: number
): Promise<PurchaseOrderRecord> {
  return apiClient<PurchaseOrderRecord>(`/purchase-orders/${encodeURIComponent(id)}/approve`, {
    method: 'POST',
    ifMatch: String(rowVersion),
  });
}

export async function receivePurchaseOrder(
  id: string,
  rowVersion: number,
  payload: ReceivePurchaseOrderPayload
): Promise<GoodsReceiptRecord> {
  return apiClient<GoodsReceiptRecord>(`/purchase-orders/${encodeURIComponent(id)}/receive`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    body: JSON.stringify(payload),
  });
}

export async function listPurchaseRequisitions(params: ListPurchaseOrdersParams = {}): Promise<{
  items: PurchaseRequisitionRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.status && params.status !== 'ALL') {
    query.set('status', params.status);
  }
  const raw = await apiClient<
    ListEnvelope<PurchaseRequisitionRecord> | PurchaseRequisitionRecord[]
  >('/purchase-requisitions', { query });
  return unwrapListResponse(raw);
}

export async function getPurchaseRequisition(
  id: string
): Promise<PurchaseRequisitionDetailRecord> {
  return apiClient<PurchaseRequisitionDetailRecord>(
    `/purchase-requisitions/${encodeURIComponent(id)}`
  );
}

export async function createPurchaseRequisition(
  payload: CreatePurchaseRequisitionPayload
): Promise<PurchaseRequisitionRecord> {
  return apiClient<PurchaseRequisitionRecord>('/purchase-requisitions', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function submitPurchaseRequisition(
  id: string,
  rowVersion: number
): Promise<PurchaseRequisitionRecord> {
  return apiClient<PurchaseRequisitionRecord>(
    `/purchase-requisitions/${encodeURIComponent(id)}/submit`,
    {
      method: 'POST',
      ifMatch: String(rowVersion),
    }
  );
}

export async function approvePurchaseRequisition(
  id: string,
  rowVersion: number
): Promise<PurchaseRequisitionRecord> {
  return apiClient<PurchaseRequisitionRecord>(
    `/purchase-requisitions/${encodeURIComponent(id)}/approve`,
    {
      method: 'POST',
      ifMatch: String(rowVersion),
    }
  );
}

export async function listGoodsReceipts(page = 1, size = 50): Promise<{
  items: GoodsReceiptRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  const raw = await apiClient<ListEnvelope<GoodsReceiptRecord> | GoodsReceiptRecord[]>(
    '/goods-receipts',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function getGoodsReceipt(id: string): Promise<GoodsReceiptDetailRecord> {
  return apiClient<GoodsReceiptDetailRecord>(`/goods-receipts/${encodeURIComponent(id)}`);
}

// Non-blocking lookup helpers for cross-service labels & dropdowns
export async function listVendorLookups(): Promise<VendorLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<ListEnvelope<VendorLookupRecord> | VendorLookupRecord[]>(
      '/parties',
      { query }
    );
    return unwrapListResponse(raw).items.filter(
      (p) => p.partyType === 'vendor' || p.partyType === 'both'
    );
  } catch {
    return [];
  }
}

export async function listProcurementItemLookups(): Promise<ProcurementItemLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<
      ListEnvelope<ProcurementItemLookupRecord> | ProcurementItemLookupRecord[]
    >('/items', { query });
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}

export async function listWarehouseLookups(): Promise<WarehouseLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<ListEnvelope<WarehouseLookupRecord> | WarehouseLookupRecord[]>(
      '/warehouses',
      { query }
    );
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}
