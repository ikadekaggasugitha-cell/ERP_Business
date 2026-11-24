import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { MaintenanceWorkspace } from '../../features/maintenance/MaintenanceWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/maintenance')({
  component: MaintenancePage,
});

function MaintenancePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <MaintenanceWorkspace />
    </DomainErrorBoundary>
  );
}
