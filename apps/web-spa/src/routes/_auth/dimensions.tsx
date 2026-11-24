import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { DimensionsWorkspace } from '../../features/dimension/DimensionsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/dimensions')({
  component: DimensionsPage,
});

function DimensionsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <DimensionsWorkspace />
    </DomainErrorBoundary>
  );
}
