import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  CreateReservationPayload,
  CreateStockMovementPayload,
  CreateWarehousePayload,
  IssueReservationPayload,
  ItemLookupRecord,
  ReservationRecord,
  StockBalanceRecord,
  StockMovementRecord,
  WarehouseRecord,
} from './types';

export interface ListStockParams {
  page?: number;
  size?: number;
  warehouseId?: string;
  itemId?: string;
}

export async function listStockBalances(params: ListStockParams = {}): Promise<{
  items: StockBalanceRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.warehouseId && params.warehouseId !== 'ALL') {
    query.set('warehouseId', params.warehouseId);
  }
  if (params.itemId && params.itemId !== 'ALL') {
    query.set('itemId', params.itemId);
  }
  const raw = await apiClient<ListEnvelope<StockBalanceRecord> | StockBalanceRecord[]>(
    '/stock-balances',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function listWarehouses(page = 1, size = 100): Promise<{
  items: WarehouseRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  const raw = await apiClient<ListEnvelope<WarehouseRecord> | WarehouseRecord[]>('/warehouses', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function createWarehouse(payload: CreateWarehousePayload): Promise<WarehouseRecord> {
  return apiClient<WarehouseRecord>('/warehouses', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function listStockMovements(params: ListStockParams = {}): Promise<{
  items: StockMovementRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.warehouseId && params.warehouseId !== 'ALL') {
    query.set('warehouseId', params.warehouseId);
  }
  if (params.itemId && params.itemId !== 'ALL') {
    query.set('itemId', params.itemId);
  }
  const raw = await apiClient<ListEnvelope<StockMovementRecord> | StockMovementRecord[]>(
    '/stock-movements',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function createStockMovement(
  payload: CreateStockMovementPayload
): Promise<StockMovementRecord> {
  return apiClient<StockMovementRecord>('/stock-movements', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function listReservations(page = 1, size = 50): Promise<{
  items: ReservationRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams({
    page: String(page),
    size: String(size),
  });
  const raw = await apiClient<ListEnvelope<ReservationRecord> | ReservationRecord[]>(
    '/reservations',
    { query }
  );
  return unwrapListResponse(raw);
}

export async function createReservation(
  payload: CreateReservationPayload
): Promise<ReservationRecord> {
  return apiClient<ReservationRecord>('/reservations', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function issueReservation(
  id: string,
  rowVersion: number,
  payload?: IssueReservationPayload
): Promise<ReservationRecord> {
  const hasAmount = payload?.amount && payload.amount.trim().length > 0;
  return apiClient<ReservationRecord>(`/reservations/${encodeURIComponent(id)}/issue`, {
    method: 'POST',
    ifMatch: String(rowVersion),
    body: hasAmount ? JSON.stringify({ amount: payload.amount?.trim() }) : undefined,
  });
}

export async function releaseReservation(
  id: string,
  rowVersion: number
): Promise<ReservationRecord> {
  return apiClient<ReservationRecord>(`/reservations/${encodeURIComponent(id)}/release`, {
    method: 'POST',
    ifMatch: String(rowVersion),
  });
}

/**
 * Optional item lookup to enrich SKU codes and names on inventory balances.
 * Returns [] if the caller lacks masterdata:read permission or masterdata is unreachable.
 */
export async function listInventoryItemLookups(): Promise<ItemLookupRecord[]> {
  try {
    const query = new URLSearchParams({ page: '1', size: '200' });
    const raw = await apiClient<ListEnvelope<ItemLookupRecord> | ItemLookupRecord[]>('/items', {
      query,
    });
    return unwrapListResponse(raw).items;
  } catch {
    return [];
  }
}
