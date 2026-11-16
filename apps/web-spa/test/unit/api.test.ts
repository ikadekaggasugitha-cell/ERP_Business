import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  ApiClientError,
  apiClient,
  applyApiErrorsToForm,
  toSearchParams,
  unwrapListResponse,
} from '../../src/lib/api';
import { onErpEvent } from '../../src/lib/event-bus';
import { DEFAULT_HOLDING_ORG, useAuthStore } from '../../src/store/auth-store';

describe('API Client & Envelope Plumbing', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    useAuthStore.setState({
      user: null,
      token: null,
      activeOrg: null,
      isAuthenticated: false,
    });
  });

  it('builds canonical pagination, sorting, and filter query parameters', () => {
    const sp = toSearchParams({
      page: 2,
      size: 50,
      sort: 'postingDate,desc',
      filters: { status: 'POSTED', empty: '', skipped: undefined },
    });

    expect(sp.get('page')).toBe('2');
    expect(sp.get('size')).toBe('50');
    expect(sp.get('sort')).toBe('postingDate,desc');
    expect(sp.get('filter[status]')).toBe('POSTED');
    expect(sp.has('filter[empty]')).toBe(false);
    expect(sp.has('filter[skipped]')).toBe(false);
  });

  it('unwraps both ListEnvelope and raw array payloads with 1-based page defaults', () => {
    const empty = unwrapListResponse(null);
    expect(empty.items).toEqual([]);
    expect(empty.page.number).toBe(1);

    const arr = unwrapListResponse([{ id: '1' }, { id: '2' }]);
    expect(arr.items).toHaveLength(2);
    expect(arr.page.number).toBe(1);
    expect(arr.page.totalPages).toBe(1);
  });

  it('omits JSON Content-Type for FormData and preserves correlation and If-Match headers', async () => {
    useAuthStore.setState({
      user: null,
      token: 'jwt-real-token',
      activeOrg: DEFAULT_HOLDING_ORG,
      isAuthenticated: true,
    });

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: async () => ({ ok: true }),
    });
    vi.stubGlobal('fetch', mockFetch);

    const form = new FormData();
    form.append('file', 'content');

    await apiClient('/documents/upload', {
      method: 'POST',
      body: form,
      ifMatch: '3',
      idempotencyKey: 'idem-123',
    });

    const [, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    const sentHeaders = init.headers as Headers;
    expect(sentHeaders.has('Content-Type')).toBe(false);
    expect(sentHeaders.get('Authorization')).toBe('Bearer jwt-real-token');
    expect(sentHeaders.get('X-Organization-Id')).toBe(DEFAULT_HOLDING_ORG.id);
    expect(sentHeaders.get('If-Match')).toBe('"3"');
    expect(sentHeaders.get('Idempotency-Key')).toBe('idem-123');
    expect(init.credentials).toBe('include');
  });

  it('emits erp:unauthorized on 401 when authenticated and parses field errors cleanly', async () => {
    useAuthStore.setState({
      user: null,
      token: null,
      activeOrg: DEFAULT_HOLDING_ORG,
      isAuthenticated: true,
    });

    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      statusText: 'Unauthorized',
      headers: new Headers({ 'X-Correlation-Id': 'cid-401', 'Retry-After': '15' }),
      json: async () => ({
        error: {
          code: 'UNAUTHORIZED',
          message: 'Expired session',
          details: [{ field: 'username', issue: 'Invalid session' }],
        },
      }),
    });
    vi.stubGlobal('fetch', mockFetch);

    const events: string[] = [];
    const off = onErpEvent('erp:unauthorized', (detail) => {
      events.push(detail.correlationId ?? '');
    });

    await expect(apiClient('/journals')).rejects.toBeInstanceOf(ApiClientError);
    off();

    expect(events).toEqual(['cid-401']);

    const err = new ApiClientError('VALIDATION_ERROR', 'Bad input', 422, 'cid-422', [
      { field: 'description', issue: 'Required' },
    ]);
    const setError = vi.fn();
    const applied = applyApiErrorsToForm<{ description: string }>(err, setError);
    expect(applied).toBe(1);
    expect(setError).toHaveBeenCalledWith('description', { type: 'server', message: 'Required' });
  });
});
