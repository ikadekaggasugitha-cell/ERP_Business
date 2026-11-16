export type SalesOrderStatus = 'Draft' | 'Confirmed' | 'Delivered' | 'Cancelled';

export interface SalesOrderLineRecord {
  id: string;
  lineNo: number;
  itemId: string;
  orderQty: string;
  unitPrice: string;
  deliveredQty: string;
  rowVersion: number;
}

export interface SalesOrderRecord {
  id: string;
  organizationId: string;
  soNo: string;
  customerId: string;
  orderDate: string;
  currencyCode: string;
  totalAmount: string;
  status: SalesOrderStatus | string;
  rowVersion: number;
}

export interface SalesOrderDetailRecord extends SalesOrderRecord {
  correlationId?: string;
  lines: SalesOrderLineRecord[];
}

export interface SalesOrderLineInput {
  lineNo: number;
  itemId: string;
  orderQty: string;
  unitPrice: string;
}

export interface UpsertSalesOrderPayload {
  soNo: string;
  customerId: string;
  orderDate: string;
  currencyCode: string;
  totalAmount: string;
  lines: SalesOrderLineInput[];
}

export interface ConfirmSalesOrderPayload {
  warehouseId?: string;
}

export interface CreateDeliveryPayload {
  deliveryNo?: string;
  warehouseId?: string;
  deliveryDate?: string;
}

export interface DeliveryLineRecord {
  id: string;
  soLineId: string;
  lineNo: number;
  itemId: string;
  deliveredQty: string;
  rowVersion: number;
}

export interface DeliveryRecord {
  id: string;
  organizationId: string;
  deliveryNo: string;
  salesOrderId: string;
  warehouseId: string;
  deliveryDate: string;
  status: string;
  rowVersion: number;
}

export interface DeliveryDetailRecord extends DeliveryRecord {
  correlationId?: string;
  lines: DeliveryLineRecord[];
}

export interface CustomerLookupRecord {
  id: string;
  code: string;
  name: string;
  partyType: string;
}

export interface SalesItemLookupRecord {
  id: string;
  code: string;
  name: string;
  baseUom: string;
}

export interface SalesWarehouseLookupRecord {
  id: string;
  code: string;
  name: string;
}
