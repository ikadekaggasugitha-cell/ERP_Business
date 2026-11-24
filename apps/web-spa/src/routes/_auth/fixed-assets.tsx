import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { FixedAssetsWorkspace } from '../../features/fixedasset/FixedAssetsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/fixed-assets')({
  component: FixedAssetsPage,
});

function FixedAssetsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <FixedAssetsWorkspace />
    </DomainErrorBoundary>
  );
}
