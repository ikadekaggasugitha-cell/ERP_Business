import { describe, expect, it } from 'vitest';
import {
  KNOWN_SSE_EVENT_TYPES,
  extractEventEnvelope,
  resolveInvalidatedQueryKeys,
} from '../../src/lib/realtime-matcher';

describe('SSE Realtime Matcher', () => {
  it('extracts normalized envelope from eventType, topic, or fallback', () => {
    const env = extractEventEnvelope(
      JSON.stringify({
        topic: 'erp.currency.rate.created.v1',
        payload: { fromCurrency: 'USD', toCurrency: 'IDR' },
      })
    );
    expect(env.eventKey).toBe('erp.currency.rate.created.v1');
    expect(env.payload.fromCurrency).toBe('USD');
  });

  it('includes all 29 domain topics and maps query keys cleanly', () => {
    expect(KNOWN_SSE_EVENT_TYPES).toContain('erp.currency.created.v1');
    expect(KNOWN_SSE_EVENT_TYPES).toContain('erp.currency.rate.created.v1');
    expect(KNOWN_SSE_EVENT_TYPES).toContain('erp.manufacturing.workorder.completed.v1');

    const taxKeys = resolveInvalidatedQueryKeys('erp.tax.invoice.issued.v1').map((k) => k[0]);
    expect(taxKeys).toContain('tax-invoices');
    expect(taxKeys).toContain('audit-events');
  });
});
