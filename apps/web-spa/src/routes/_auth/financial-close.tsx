import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { FinancialCloseWorkspace } from '../../features/financialclose/FinancialCloseWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/financial-close')({
  component: FinancialClosePage,
});

function FinancialClosePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <FinancialCloseWorkspace />
    </DomainErrorBoundary>
  );
}
