import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { BudgetsWorkspace } from '../../features/budget/BudgetsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/budgets')({
  component: BudgetsPage,
});

function BudgetsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <BudgetsWorkspace />
    </DomainErrorBoundary>
  );
}
