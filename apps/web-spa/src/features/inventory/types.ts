export type MovementType = 'Receipt' | 'Issue' | 'Adjustment';

export interface WarehouseRecord {
  id: string;
  code: string;
  name: string;
  organizationRefId: string;
  rowVersion: number;
}

export interface CreateWarehousePayload {
  code: string;
  name: string;
  organizationRefId?: string;
}

export interface StockBalanceRecord {
  id: string;
  warehouseId: string;
  itemId: string;
  onHandQty: string;
  reservedQty: string;
  availableQty: string;
  avgCost: string;
  rowVersion: number;
}

export interface StockMovementRecord {
  id: string;
  movementDate: string;
  warehouseId: string;
  itemId: string;
  movementType: MovementType;
  quantity: string;
  sourceDocumentRef?: string;
  rowVersion: number;
}

export interface CreateStockMovementPayload {
  movementDate?: string;
  warehouseId: string;
  itemId: string;
  movementType: MovementType;
  quantity: string;
  unitCost?: string;
  sourceDocumentRef?: string;
}

export type ReservationStatus = 'Reserved' | 'Consumed' | 'Released';

export interface ReservationRecord {
  id: string;
  itemId: string;
  warehouseId: string;
  reservedQty: string;
  salesOrderRef: string;
  status: ReservationStatus;
  rowVersion: number;
}

export interface CreateReservationPayload {
  itemId: string;
  warehouseId: string;
  reservedQty: string;
  salesOrderRef?: string;
}

export interface IssueReservationPayload {
  amount?: string;
}

export interface ItemLookupRecord {
  id: string;
  code: string;
  name: string;
  itemType: string;
  baseUom: string;
}
