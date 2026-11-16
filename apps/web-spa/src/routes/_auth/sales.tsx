import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { useAuthStore } from '../../store/auth-store';

const LazySalesWorkspace = React.lazy(() =>
  import('../../features/sales/SalesWorkspace').then((m) => ({
    default: m.SalesWorkspace,
  }))
);

function SalesRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LazySalesWorkspace />
    </DomainErrorBoundary>
  );
}

export const Route = createFileRoute('/_auth/sales')({
  component: SalesRoutePage,
});
