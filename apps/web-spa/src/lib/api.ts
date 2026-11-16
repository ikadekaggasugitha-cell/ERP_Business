import type { FieldPath, FieldValues, UseFormSetError } from 'react-hook-form';
import { useAuthStore } from '../store/auth-store';
import { emitErpEvent } from './event-bus';
import type { ApiErrorResponse, ApiFieldError, ListEnvelope, PageQueryParams } from '../types';

export function resolveApiBaseUrl(): string {
  const envUrl = import.meta.env.VITE_API_URL;
  if (!envUrl) {
    return '/api/v1';
  }
  const trimmed = envUrl.replace(/\/+$/, '');
  return trimmed.endsWith('/api/v1') ? trimmed : `${trimmed}/api/v1`;
}

const BASE_URL = resolveApiBaseUrl();

export class ApiClientError extends Error {
  code: string;
  correlationId?: string;
  details?: ApiFieldError[];
  status: number;
  retryAfterSeconds?: number;
  etag?: string;

  constructor(
    code: string,
    message: string,
    status: number,
    correlationId?: string,
    details?: ApiFieldError[],
    retryAfterSeconds?: number,
    etag?: string
  ) {
    super(message);
    this.name = 'ApiClientError';
    this.code = code;
    this.status = status;
    this.correlationId = correlationId;
    this.details = details;
    this.retryAfterSeconds = retryAfterSeconds;
    this.etag = etag;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }

  get isConflict(): boolean {
    return this.status === 409;
  }

  get isValidationError(): boolean {
    return this.status === 422 || this.status === 400;
  }

  get isRateLimited(): boolean {
    return this.status === 429;
  }
}

export interface RequestOptions extends RequestInit {
  idempotencyKey?: string;
  orgId?: string;
  ifMatch?: string;
  query?: PageQueryParams | URLSearchParams;
  skipAuthRedirect?: boolean;
}

/**
 * Converts ERP standard pagination/sort/filter parameters into URLSearchParams:
 * `?page=1&size=25&sort=postingDate,desc&filter[status]=POSTED`
 */
export function toSearchParams(params?: PageQueryParams | URLSearchParams): URLSearchParams {
  if (params instanceof URLSearchParams) {
    return params;
  }
  const sp = new URLSearchParams();
  if (!params) return sp;

  if (typeof params.page === 'number' && !Number.isNaN(params.page)) {
    sp.set('page', String(params.page));
  }
  if (typeof params.size === 'number' && !Number.isNaN(params.size)) {
    sp.set('size', String(params.size));
  }
  if (params.sort) {
    sp.set('sort', params.sort);
  }
  if (params.filters) {
    for (const [key, val] of Object.entries(params.filters)) {
      if (val !== undefined && val !== null && val !== '') {
        sp.set(`filter[${key}]`, String(val));
      }
    }
  }
  return sp;
}

export async function apiClient<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const { token, activeOrg, isAuthenticated } = useAuthStore.getState();
  const correlationId = crypto.randomUUID();

  const headers = new Headers(options.headers);
  const isMultipartOrBinary =
    (typeof FormData !== 'undefined' && options.body instanceof FormData) ||
    (typeof URLSearchParams !== 'undefined' && options.body instanceof URLSearchParams) ||
    (typeof Blob !== 'undefined' && options.body instanceof Blob);

  if (!headers.has('Content-Type') && !isMultipartOrBinary) {
    headers.set('Content-Type', 'application/json');
  }
  if (!headers.has('X-Correlation-Id')) {
    headers.set('X-Correlation-Id', correlationId);
  }

  if (token && token !== 'dev-jwt-offline-token') {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const targetOrgId = options.orgId || activeOrg?.id;
  if (targetOrgId) {
    headers.set('X-Organization-Id', targetOrgId);
  }

  if (options.idempotencyKey) {
    headers.set('Idempotency-Key', options.idempotencyKey);
  }

  if (options.ifMatch) {
    headers.set('If-Match', options.ifMatch.startsWith('"') ? options.ifMatch : `"${options.ifMatch}"`);
  }

  let finalUrl: string;
  if (endpoint.startsWith('http://') || endpoint.startsWith('https://')) {
    finalUrl = endpoint;
  } else {
    const cleanEndpoint = endpoint.startsWith('/api/v1') ? endpoint.slice('/api/v1'.length) : endpoint;
    const path = cleanEndpoint.startsWith('/') ? cleanEndpoint : `/${cleanEndpoint}`;
    finalUrl = `${BASE_URL}${path}`;
  }

  if (options.query) {
    const sp = toSearchParams(options.query);
    const qs = sp.toString();
    if (qs) {
      finalUrl += finalUrl.includes('?') ? `&${qs}` : `?${qs}`;
    }
  }

  const {
    idempotencyKey: _idemp,
    orgId: _org,
    ifMatch: _ifMatch,
    query: _query,
    skipAuthRedirect,
    ...fetchOptions
  } = options;

  const response = await fetch(finalUrl, {
    ...fetchOptions,
    credentials: 'include',
    headers,
  });

  if (!response.ok) {
    let errorData: ApiErrorResponse['error'];
    try {
      const json = (await response.json()) as Partial<ApiErrorResponse>;
      errorData = json.error || { code: 'UNKNOWN_ERROR', message: response.statusText };
    } catch {
      errorData = { code: 'NETWORK_ERROR', message: `HTTP Error ${response.status}: ${response.statusText}` };
    }

    const retryHeader = response.headers.get('Retry-After');
    const parsedRetry = retryHeader && /^\d+$/.test(retryHeader) ? Number(retryHeader) : undefined;
    const etag = response.headers.get('ETag') || undefined;
    const resolvedCorrelationId =
      errorData.correlationId || response.headers.get('X-Correlation-Id') || correlationId;

    const clientError = new ApiClientError(
      errorData.code,
      errorData.message,
      response.status,
      resolvedCorrelationId,
      errorData.details,
      parsedRetry,
      etag
    );

    if (response.status === 401 && isAuthenticated && !skipAuthRedirect) {
      emitErpEvent('erp:unauthorized', {
        status: response.status,
        correlationId: resolvedCorrelationId,
      });
    } else if (response.status === 409) {
      emitErpEvent('erp:conflict', {
        status: response.status,
        message: errorData.message,
        correlationId: resolvedCorrelationId,
      });
    }

    throw clientError;
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json() as Promise<T>;
}

/**
 * Safely unwraps either a canonical `{ data, page }` ListEnvelope or a plain array response.
 */
export function unwrapListResponse<T>(payload: ListEnvelope<T> | T[] | null | undefined): {
  items: T[];
  page: ListEnvelope<T>['page'];
} {
  if (!payload) {
    return { items: [], page: { number: 1, size: 0, totalElements: 0, totalPages: 0 } };
  }
  if (Array.isArray(payload)) {
    return {
      items: payload,
      page: {
        number: 1,
        size: payload.length,
        totalElements: payload.length,
        totalPages: payload.length > 0 ? 1 : 0,
      },
    };
  }
  const items = Array.isArray(payload.data) ? payload.data : [];
  return {
    items,
    page: payload.page ?? {
      number: 1,
      size: items.length,
      totalElements: items.length,
      totalPages: items.length > 0 ? 1 : 0,
    },
  };
}

export function extractApiFieldErrors(err: unknown): Record<string, string> {
  if (!(err instanceof ApiClientError) || !Array.isArray(err.details)) {
    return {};
  }
  const result: Record<string, string> = {};
  for (const detail of err.details) {
    if (detail.field) {
      result[detail.field] = detail.issue;
    }
  }
  return result;
}

/**
 * Maps HTTP 422 field-level error details from Go microservices directly onto React Hook Form fields.
 */
export function applyApiErrorsToForm<TFieldValues extends FieldValues>(
  err: unknown,
  setError: UseFormSetError<TFieldValues>
): number {
  const fieldErrors = extractApiFieldErrors(err);
  let applied = 0;
  for (const [field, message] of Object.entries(fieldErrors)) {
    setError(field as FieldPath<TFieldValues>, { type: 'server', message });
    applied += 1;
  }
  return applied;
}
