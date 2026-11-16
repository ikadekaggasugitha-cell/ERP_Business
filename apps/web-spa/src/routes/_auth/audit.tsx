import { createFileRoute } from '@tanstack/react-router';
import { AuditWorkspace } from '../../features/audit/AuditWorkspace';
import type { AuditFilterState } from '../../features/audit/audit-api';

interface AuditSearchParams {
  entityType?: string;
  action?: string;
  actor?: string;
  dateFrom?: string;
  dateTo?: string;
  page?: number;
}

export const Route = createFileRoute('/_auth/audit')({
  validateSearch: (search: Record<string, unknown>): AuditSearchParams => ({
    entityType: typeof search.entityType === 'string' ? search.entityType : undefined,
    action: typeof search.action === 'string' ? search.action : undefined,
    actor: typeof search.actor === 'string' ? search.actor : undefined,
    dateFrom: typeof search.dateFrom === 'string' ? search.dateFrom : undefined,
    dateTo: typeof search.dateTo === 'string' ? search.dateTo : undefined,
    page: typeof search.page === 'number' && search.page >= 1 ? search.page : undefined,
  }),
  component: AuditPage,
});

function AuditPage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();

  const handleFiltersChange = (next: AuditFilterState) => {
    void navigate({
      search: {
        entityType: next.entityType || undefined,
        action: next.action || undefined,
        actor: next.actor || undefined,
        dateFrom: next.dateFrom || undefined,
        dateTo: next.dateTo || undefined,
        page: next.page > 1 ? next.page : undefined,
      },
      replace: true,
    });
  };

  return <AuditWorkspace initialFilters={search} onFiltersChange={handleFiltersChange} />;
}
