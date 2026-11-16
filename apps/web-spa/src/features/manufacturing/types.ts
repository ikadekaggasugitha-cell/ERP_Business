export type WorkOrderStatus = 'Planned' | 'Released' | 'Completed';

export interface BomLineRecord {
  id: string;
  lineNo: number;
  componentItemId: string;
  quantity: string;
  uom: string;
}

export interface BomRecord {
  id: string;
  code: string;
  productItemId: string;
  version: number;
  isActive: boolean;
  rowVersion: number;
}

export interface BomDetailRecord extends BomRecord {
  lines: BomLineRecord[];
}

export interface CreateBomLineInput {
  lineNo: number;
  componentItemId: string;
  quantity: string;
  uom: string;
}

export interface CreateBomPayload {
  code: string;
  productItemId: string;
  version?: number;
  lines: CreateBomLineInput[];
}

export interface WorkOrderRecord {
  id: string;
  woNo: string;
  productItemId: string;
  bomId: string;
  plannedQty: string;
  producedQty: string;
  salesOrderRef?: string;
  status: WorkOrderStatus | string;
  rowVersion: number;
}

export interface UpsertWorkOrderPayload {
  woNo: string;
  productItemId: string;
  bomId: string;
  plannedQty: string;
  salesOrderRef?: string;
}

export interface CompleteWorkOrderPayload {
  producedQty?: string;
}

export interface ManufacturingItemLookupRecord {
  id: string;
  code: string;
  name: string;
  baseUom: string;
}
