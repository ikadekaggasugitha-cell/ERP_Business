import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { ConsolidationWorkspace } from '../../features/consolidation/ConsolidationWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/consolidation')({
  component: ConsolidationPage,
});

function ConsolidationPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <ConsolidationWorkspace />
    </DomainErrorBoundary>
  );
}
