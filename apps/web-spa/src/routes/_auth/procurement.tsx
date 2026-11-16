import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { useAuthStore } from '../../store/auth-store';

const LazyProcurementWorkspace = React.lazy(() =>
  import('../../features/procurement/ProcurementWorkspace').then((m) => ({
    default: m.ProcurementWorkspace,
  }))
);

function ProcurementRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LazyProcurementWorkspace />
    </DomainErrorBoundary>
  );
}

export const Route = createFileRoute('/_auth/procurement')({
  component: ProcurementRoutePage,
});
