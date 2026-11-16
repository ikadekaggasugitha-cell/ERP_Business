export type PartyType = 'customer' | 'vendor' | 'both';

export interface PartyAddress {
  id: string;
  addressType: string;
  line1: string;
  line2?: string | null;
  city?: string | null;
  stateProvince?: string | null;
  postalCode?: string | null;
  countryCode?: string | null;
  isPrimary: boolean;
  rowVersion: number;
}

export interface PartyRecord {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  partyType: PartyType;
  taxId?: string | null;
  isShared: boolean;
  rowVersion: number;
}

export interface PartyDetailRecord extends PartyRecord {
  addresses: PartyAddress[];
}

export interface PartyAddressInput {
  addressType: string;
  line1: string;
  line2?: string;
  city?: string;
  stateProvince?: string;
  postalCode?: string;
  countryCode?: string;
  isPrimary: boolean;
}

export interface CreatePartyPayload {
  code: string;
  name: string;
  partyType: PartyType;
  taxId?: string;
  isShared: boolean;
  addresses?: PartyAddressInput[];
}

export interface UpdatePartyPayload {
  code?: string;
  name: string;
  partyType: PartyType;
  taxId?: string;
  isShared: boolean;
}

export type ItemType = 'stock' | 'service' | 'asset';

export interface ItemUom {
  id: string;
  uomCode: string;
  conversionToBase: string;
  isBase: boolean;
  rowVersion: number;
}

export interface ItemRecord {
  id: string;
  organizationId: string;
  code: string;
  name: string;
  itemType: ItemType;
  baseUom: string;
  rowVersion: number;
}

export interface ItemDetailRecord extends ItemRecord {
  uoms: ItemUom[];
}

export interface CreateItemPayload {
  code: string;
  name: string;
  itemType: ItemType;
  baseUom: string;
}

export interface UpdateItemPayload {
  code?: string;
  name: string;
  itemType: ItemType;
  baseUom: string;
}
