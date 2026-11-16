import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { CurrencyWorkspace } from '../../features/currency/CurrencyWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/currency' as unknown as '/_auth/ap')({
  component: CurrencyPage,
});

function CurrencyPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <CurrencyWorkspace />
    </DomainErrorBoundary>
  );
}
