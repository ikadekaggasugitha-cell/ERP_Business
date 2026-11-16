export type PurchaseOrderStatus = 'Draft' | 'Approved' | 'Received';
export type PurchaseRequisitionStatus = 'Draft' | 'Submitted' | 'Approved';

export interface PurchaseOrderLineRecord {
  id: string;
  lineNo: number;
  itemId: string;
  orderQty: string;
  receivedQty: string;
  unitPrice: string;
}

export interface PurchaseOrderRecord {
  id: string;
  poNo: string;
  vendorId: string;
  purchaseRequisitionId?: string;
  orderDate: string;
  currencyCode: string;
  totalAmount: string;
  status: PurchaseOrderStatus | string;
  rowVersion: number;
}

export interface PurchaseOrderDetailRecord extends PurchaseOrderRecord {
  lines: PurchaseOrderLineRecord[];
}

export interface PurchaseOrderLineInput {
  lineNo: number;
  itemId: string;
  orderQty: string;
  unitPrice: string;
}

export interface UpsertPurchaseOrderPayload {
  poNo: string;
  vendorId: string;
  purchaseRequisitionId?: string;
  orderDate: string;
  currencyCode: string;
  totalAmount: string;
  lines: PurchaseOrderLineInput[];
}

export interface ReceivePurchaseOrderPayload {
  grNo?: string;
  warehouseId?: string;
  receiptDate?: string;
}

export interface PurchaseRequisitionLineRecord {
  id: string;
  lineNo: number;
  itemId: string;
  requestQty: string;
  neededDate?: string;
}

export interface PurchaseRequisitionRecord {
  id: string;
  prNo: string;
  requesterId: string;
  workflowRef?: string;
  neededDate?: string;
  status: PurchaseRequisitionStatus | string;
  rowVersion: number;
}

export interface PurchaseRequisitionDetailRecord extends PurchaseRequisitionRecord {
  lines: PurchaseRequisitionLineRecord[];
}

export interface PurchaseRequisitionLineInput {
  lineNo: number;
  itemId: string;
  requestQty: string;
  neededDate?: string;
}

export interface CreatePurchaseRequisitionPayload {
  prNo: string;
  requesterId?: string;
  workflowRef?: string;
  neededDate?: string;
  lines: PurchaseRequisitionLineInput[];
}

export interface GoodsReceiptLineRecord {
  id: string;
  poLineId: string;
  lineNo: number;
  receivedQty: string;
}

export interface GoodsReceiptRecord {
  id: string;
  grNo: string;
  purchaseOrderId: string;
  warehouseId: string;
  receiptDate: string;
  status: string;
  rowVersion: number;
}

export interface GoodsReceiptDetailRecord extends GoodsReceiptRecord {
  lines: GoodsReceiptLineRecord[];
}

export interface VendorLookupRecord {
  id: string;
  code: string;
  name: string;
  partyType: string;
}

export interface ProcurementItemLookupRecord {
  id: string;
  code: string;
  name: string;
  baseUom: string;
}

export interface WarehouseLookupRecord {
  id: string;
  code: string;
  name: string;
}
