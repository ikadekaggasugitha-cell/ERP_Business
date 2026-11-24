import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { LocalizationWorkspace } from '../../features/localization/LocalizationWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/localization')({
  component: LocalizationPage,
});

function LocalizationPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LocalizationWorkspace />
    </DomainErrorBoundary>
  );
}
