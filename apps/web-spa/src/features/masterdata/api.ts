import { apiClient, unwrapListResponse } from '../../lib/api';
import type { ListEnvelope, PageMeta } from '../../types';
import type {
  CreateItemPayload,
  CreatePartyPayload,
  ItemDetailRecord,
  ItemRecord,
  PartyDetailRecord,
  PartyRecord,
  UpdateItemPayload,
  UpdatePartyPayload,
} from './types';

export interface ListPartiesParams {
  page?: number;
  size?: number;
  partyType?: string;
}

export async function listParties(params: ListPartiesParams = {}): Promise<{
  items: PartyRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.partyType && params.partyType !== 'ALL') {
    query.set('partyType', params.partyType);
  }
  const raw = await apiClient<ListEnvelope<PartyRecord> | PartyRecord[]>('/parties', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function getParty(id: string): Promise<PartyDetailRecord> {
  return apiClient<PartyDetailRecord>(`/parties/${encodeURIComponent(id)}`);
}

export async function createParty(payload: CreatePartyPayload): Promise<PartyDetailRecord> {
  return apiClient<PartyDetailRecord>('/parties', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function updateParty(
  id: string,
  rowVersion: number,
  payload: UpdatePartyPayload
): Promise<PartyRecord> {
  return apiClient<PartyRecord>(`/parties/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    ifMatch: String(rowVersion),
  });
}

export async function deleteParty(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/parties/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}

export interface ListItemsParams {
  page?: number;
  size?: number;
  itemType?: string;
}

export async function listItems(params: ListItemsParams = {}): Promise<{
  items: ItemRecord[];
  page: PageMeta;
}> {
  const query = new URLSearchParams();
  query.set('page', String(params.page ?? 1));
  query.set('size', String(params.size ?? 50));
  if (params.itemType && params.itemType !== 'ALL') {
    query.set('itemType', params.itemType);
  }
  const raw = await apiClient<ListEnvelope<ItemRecord> | ItemRecord[]>('/items', {
    query,
  });
  return unwrapListResponse(raw);
}

export async function getItem(id: string): Promise<ItemDetailRecord> {
  return apiClient<ItemDetailRecord>(`/items/${encodeURIComponent(id)}`);
}

export async function createItem(payload: CreateItemPayload): Promise<ItemDetailRecord> {
  return apiClient<ItemDetailRecord>('/items', {
    method: 'POST',
    body: JSON.stringify(payload),
    idempotencyKey: crypto.randomUUID(),
  });
}

export async function updateItem(
  id: string,
  rowVersion: number,
  payload: UpdateItemPayload
): Promise<ItemRecord> {
  return apiClient<ItemRecord>(`/items/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
    ifMatch: String(rowVersion),
  });
}

export async function deleteItem(id: string, rowVersion: number): Promise<void> {
  await apiClient<unknown>(`/items/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ifMatch: String(rowVersion),
  });
}
