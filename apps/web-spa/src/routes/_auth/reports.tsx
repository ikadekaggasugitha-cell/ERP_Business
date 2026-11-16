import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { ReportsWorkspace } from '../../features/reports/ReportsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/reports')({
  component: ReportsPage,
});

function ReportsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');

  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <ReportsWorkspace />
    </DomainErrorBoundary>
  );
}
