import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { TreasuryWorkspace } from '../../features/treasury/TreasuryWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/treasury')({
  component: TreasuryPage,
});

function TreasuryPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <TreasuryWorkspace />
    </DomainErrorBoundary>
  );
}
