import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { CrmWorkspace } from '../../features/crm/CrmWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/crm')({
  component: CrmPage,
});

function CrmPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <CrmWorkspace />
    </DomainErrorBoundary>
  );
}
