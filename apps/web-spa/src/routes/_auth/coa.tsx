import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { CoaWorkspace } from '../../features/coa/CoaWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/coa')({
  component: CoaPage,
});

function CoaPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');

  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <CoaWorkspace />
    </DomainErrorBoundary>
  );
}
