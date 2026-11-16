export interface CurrencyDto {
  id: string;
  code: string;
  numericCode?: string | null;
  name: string;
  precision: number;
  isActive: boolean;
  rowVersion: number;
}

export interface CreateCurrencyInput {
  code: string;
  numericCode?: string;
  name: string;
  precision: number;
  isActive: boolean;
}

export interface UpdateCurrencyInput {
  code?: string;
  numericCode?: string;
  name: string;
  precision: number;
  isActive: boolean;
}

export interface ExchangeRateTypeDto {
  id: string;
  code: string;
  name: string;
  rowVersion: number;
}

export interface CreateExchangeRateTypeInput {
  code: string;
  name: string;
}

export interface ExchangeRateDto {
  id: string;
  fromCurrency: string;
  toCurrency: string;
  rateTypeId: string;
  rateTypeCode: string;
  rateTypeName: string;
  rate: string;
  validFrom: string;
  validTo?: string | null;
  rowVersion: number;
}

export interface CreateExchangeRateInput {
  fromCurrency: string;
  toCurrency: string;
  rateTypeId: string;
  rate: string;
  validFrom: string;
  validTo?: string;
}

export interface UpdateExchangeRateInput {
  rate: string;
  validTo?: string;
}

export interface ConvertCurrencyInput {
  fromCurrency: string;
  toCurrency: string;
  rateTypeCode?: string;
  rateTypeId?: string;
  amount: string;
  onDate?: string;
  anchor?: string;
}

export interface ConvertCurrencyResultDto {
  fromCurrency: string;
  toCurrency: string;
  rateTypeId: string;
  originalAmount: string;
  convertedAmount: string;
  rate: string;
  rateSource: 'IDENTITY' | 'DIRECT' | 'INVERSE' | 'TRIANGULATED' | string;
  anchorCurrency?: string | null;
  effectiveDate: string;
}
