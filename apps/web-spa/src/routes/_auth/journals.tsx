import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { JournalsWorkspace } from '../../features/journals/JournalsWorkspace';
import { useAuthStore } from '../../store/auth-store';

const journalsSearchSchema = z.object({
  search: z.string().default('').catch(''),
  status: z.enum(['ALL', 'DRAFT', 'POSTED', 'REVERSED']).default('ALL').catch('ALL'),
  page: z.number().int().positive().default(1).catch(1),
});

export const Route = createFileRoute('/_auth/journals')({
  validateSearch: journalsSearchSchema,
  component: JournalsPage,
});

function JournalsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  const { search, status, page } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <JournalsWorkspace
        search={search}
        status={status}
        page={page}
        onSearchStateChange={(next) => {
          void navigate({
            search: (prev) => ({
              search: next.search !== undefined ? next.search : prev.search,
              status: next.status !== undefined ? next.status : prev.status,
              page: next.page !== undefined ? next.page : prev.page,
            }),
            replace: true,
          });
        }}
      />
    </DomainErrorBoundary>
  );
}
