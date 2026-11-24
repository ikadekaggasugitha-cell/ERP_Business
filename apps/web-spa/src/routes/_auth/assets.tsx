import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { AssetsWorkspace } from '../../features/asset/AssetsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/assets')({
  component: AssetsPage,
});

function AssetsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <AssetsWorkspace />
    </DomainErrorBoundary>
  );
}
