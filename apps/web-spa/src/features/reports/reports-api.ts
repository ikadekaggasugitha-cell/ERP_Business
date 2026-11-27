import { useQuery } from '@tanstack/react-query';
import { apiClient, resolveApiBaseUrl } from '../../lib/api';

export type FinancialStatementKind = 'income-statement' | 'balance-sheet' | 'cash-flow';

export interface FinancialStatementItem {
  accountCode: string;
  accountName: string;
  amount: string;
}

export interface FinancialStatementSection {
  code: string;
  name: string;
  items: FinancialStatementItem[];
  subtotal: string;
}

export interface FinancialStatementResponse {
  statementType: string;
  currency: string;
  asOfDate?: string;
  period?: string;
  sections: FinancialStatementSection[];
  totals: Record<string, string>;
  isBalanced?: boolean;
}

export const financialStatementKeys = {
  byKind: (orgId: string | undefined, kind: FinancialStatementKind) =>
    ['financial-statements', orgId, kind] as const,
};

export function useFinancialStatementQuery(
  orgId: string | undefined,
  kind: FinancialStatementKind,
  enabled: boolean
) {
  return useQuery<FinancialStatementResponse>({
    queryKey: financialStatementKeys.byKind(orgId, kind),
    queryFn: () => apiClient<FinancialStatementResponse>(`/financial-statements/${kind}`),
    enabled: Boolean(orgId) && enabled,
  });
}

export function buildFinancialStatementCsvUrl(kind: FinancialStatementKind): string {
  return `${resolveApiBaseUrl()}/financial-statements/${kind}?format=csv`;
}
