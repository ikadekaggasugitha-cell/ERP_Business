import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { DashboardWorkspace } from '../../features/dashboard/DashboardWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/')({
  component: DashboardRoutePage,
});

function DashboardRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');

  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <DashboardWorkspace />
    </DomainErrorBoundary>
  );
}
