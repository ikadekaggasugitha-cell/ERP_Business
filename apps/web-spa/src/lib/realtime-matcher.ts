export interface NormalizedSseEvent {
  eventKey: string;
  payload: Record<string, unknown>;
}

/**
 * Normalizes an incoming SSE message payload and event type into a canonical eventKey + payload object.
 */
export function extractEventEnvelope(
  rawData: string | undefined,
  eventType?: string
): NormalizedSseEvent {
  let parsed: Record<string, unknown> = {};
  if (rawData) {
    try {
      const json = JSON.parse(rawData);
      if (json && typeof json === 'object') {
        parsed = json as Record<string, unknown>;
      }
    } catch {
      parsed = {};
    }
  }

  const eventKey = String(parsed.eventType || parsed.topic || eventType || '').trim();
  const innerPayload =
    (parsed.payload as Record<string, unknown>) ||
    (parsed.data as Record<string, unknown>) ||
    parsed;

  return {
    eventKey,
    payload: innerPayload && typeof innerPayload === 'object' ? innerPayload : {},
  };
}

/**
 * Matches both `erp.<domain>.*` (all 29 DomainTopics) and `<domain>.*` event patterns to TanStack Query keys.
 */
export function resolveInvalidatedQueryKeys(eventKey: string): string[][] {
  const key = eventKey.toLowerCase().trim();
  if (!key || key === 'connected' || key === 'message') {
    return [['dashboard-metrics']];
  }

  const invalidated = new Set<string>();

  if (/^(?:erp\.)?gl\./.test(key)) {
    invalidated.add('journals');
    invalidated.add('balances');
    invalidated.add('gl-balances');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?coa\./.test(key)) {
    invalidated.add('coa-accounts');
    invalidated.add('fiscal-periods');
    invalidated.add('ledgers');
    invalidated.add('balances');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?workflow\./.test(key)) {
    invalidated.add('approvals');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?ap\./.test(key)) {
    invalidated.add('ap-invoices');
    invalidated.add('balances');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?ar\./.test(key)) {
    invalidated.add('ar-invoices');
    invalidated.add('balances');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?tax\./.test(key)) {
    invalidated.add('tax-invoices');
    invalidated.add('tax-codes');
    invalidated.add('balances');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?(?:treasury|cashbank)\./.test(key)) {
    invalidated.add('bank-accounts');
    invalidated.add('cashbank-transactions');
    invalidated.add('balances');
    invalidated.add('ap-invoices');
    invalidated.add('ar-invoices');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?inventory\./.test(key)) {
    invalidated.add('inventory-items');
    invalidated.add('stock-movements');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?masterdata\./.test(key)) {
    invalidated.add('inventory-items');
    invalidated.add('business-partners');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?procurement\./.test(key)) {
    invalidated.add('purchase-orders');
    invalidated.add('purchase-requisitions');
    invalidated.add('goods-receipts');
    invalidated.add('inventory-items');
    invalidated.add('ap-invoices');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?sales\./.test(key)) {
    invalidated.add('sales-orders');
    invalidated.add('ar-invoices');
    invalidated.add('inventory-items');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?manufacturing\./.test(key)) {
    invalidated.add('work-orders');
    invalidated.add('boms');
    invalidated.add('inventory-items');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?hr\./.test(key)) {
    invalidated.add('employees');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?payroll\./.test(key)) {
    invalidated.add('payroll-runs');
    invalidated.add('journals');
    invalidated.add('bank-accounts');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?currency\./.test(key)) {
    invalidated.add('currencies');
    invalidated.add('exchange-rates');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?org\./.test(key)) {
    invalidated.add('organizations');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?user\./.test(key)) {
    invalidated.add('users');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?security\./.test(key)) {
    invalidated.add('roles');
    invalidated.add('users');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?audit\./.test(key)) {
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?fixedasset\./.test(key)) {
    invalidated.add('fixed-assets');
    invalidated.add('asset-categories');
    invalidated.add('journals');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?dimension\./.test(key)) {
    invalidated.add('cost-centers');
    invalidated.add('profit-centers');
    invalidated.add('allocation-rules');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?budget\./.test(key)) {
    invalidated.add('budgets');
    invalidated.add('budget-policies');
    invalidated.add('budget-commitments');
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?financialclose\./.test(key)) {
    invalidated.add('closing-runs');
    invalidated.add('closing-templates');
    invalidated.add('period-statuses');
    invalidated.add('period-reopenings');
    invalidated.add('fiscal-periods');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?consolidation\./.test(key)) {
    invalidated.add('consolidation-groups');
    invalidated.add('elimination-rules');
    invalidated.add('consolidation-runs');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?project\./.test(key)) {
    invalidated.add('projects');
    invalidated.add('project-billings');
    invalidated.add('ar-invoices');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?asset\./.test(key)) {
    invalidated.add('operational-assets');
    invalidated.add('asset-movements');
    invalidated.add('asset-verifications');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?maintenance\./.test(key)) {
    invalidated.add('maintenance-schedules');
    invalidated.add('maintenance-work-orders');
    invalidated.add('inventory-items');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?crm\./.test(key)) {
    invalidated.add('crm-leads');
    invalidated.add('crm-opportunities');
    invalidated.add('crm-segments');
    invalidated.add('crm-support-cases');
    invalidated.add('sales-orders');
    invalidated.add('audit-events');
  }

  if (/^(?:erp\.)?localization\./.test(key)) {
    invalidated.add('localization-languages');
    invalidated.add('localization-translations');
    invalidated.add('localization-formats');
    invalidated.add('audit-events');
  }

  if (invalidated.size === 0) {
    invalidated.add('dashboard-metrics');
    invalidated.add('audit-events');
  }

  return Array.from(invalidated).map((k) => [k]);
}

/**
 * Synchronized with services/api-gateway/internal/sse/hub.go DomainTopics (all 29 canonical topics)
 * plus short domain aliases for resilience.
 */
export const KNOWN_SSE_EVENT_TYPES = [
  'erp.gl.journal.posted.v1',
  'erp.gl.journal.reversed.v1',
  'gl.journal.posted',
  'erp.coa.account.upserted.v1',
  'erp.coa.period.updated.v1',
  'erp.ap.invoice.recorded.v1',
  'ap.invoice.recorded',
  'erp.ar.invoice.issued.v1',
  'ar.invoice.issued',
  'erp.tax.invoice.issued.v1',
  'erp.tax.efaktur.reported.v1',
  'erp.treasury.payment.settled.v1',
  'treasury.payment.settled',
  'erp.cashbank.payment.settled.v1',
  'cashbank.payment.settled',
  'erp.hr.employee.updated.v1',
  'erp.payroll.run.completed.v1',
  'erp.procurement.po.approved.v1',
  'erp.procurement.goods.received.v1',
  'procurement.po.approved',
  'procurement.goods.received',
  'erp.inventory.stock.reserved.v1',
  'erp.inventory.goods.issued.v1',
  'erp.inventory.reservation.failed.v1',
  'inventory.stock.reserved',
  'inventory.goods.issued',
  'erp.sales.order.confirmed.v1',
  'erp.manufacturing.workorder.released.v1',
  'erp.manufacturing.workorder.completed.v1',
  'erp.masterdata.party.upserted.v1',
  'masterdata.party.upserted',
  'masterdata.item.upserted',
  'erp.org.unit.upserted.v1',
  'erp.security.role.upserted.v1',
  'erp.user.account.upserted.v1',
  'erp.workflow.approval.requested.v1',
  'erp.workflow.approval.completed.v1',
  'workflow.approval.requested',
  'workflow.approval.completed',
  'erp.currency.created.v1',
  'erp.currency.rate.created.v1',
  'erp.currency.rate.updated.v1',
  'currency.created',
  'currency.rate.updated',
  'currency.exchange_rate.created',
  'currency.exchange_rate.updated',
  'erp.audit.event.recorded.v1',
  'audit.event.recorded',
  'erp.dimension.cost_center.upserted.v1',
  'erp.dimension.profit_center.upserted.v1',
  'erp.dimension.allocation_rule.upserted.v1',
  'erp.dimension.allocation_rule.executed.v1',
  'erp.localization.language.updated.v1',
  'erp.localization.bundle.updated.v1',
  'erp.localization.locale_format.updated.v1',
  'erp.fixedasset.asset.acquired.v1',
  'erp.fixedasset.depreciation.posted.v1',
  'erp.fixedasset.asset.disposed.v1',
  'erp.budget.approved.v1',
  'erp.budget.commitment.created.v1',
  'erp.financialclose.period.closed.v1',
  'erp.financialclose.period.locked.v1',
  'erp.financialclose.period.reopened.v1',
  'erp.consolidation.run.finalized.v1',
  'erp.project.billing.approved.v1',
  'erp.asset.movement.completed.v1',
  'erp.asset.verification.recorded.v1',
  'erp.maintenance.workorder.parts_requested.v1',
  'erp.maintenance.workorder.completed.v1',
  'erp.crm.opportunity.won.v1',
];
