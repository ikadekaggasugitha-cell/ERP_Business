import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { useAuthStore } from '../../store/auth-store';

const LazyInventoryWorkspace = React.lazy(() =>
  import('../../features/inventory/InventoryWorkspace').then((m) => ({
    default: m.InventoryWorkspace,
  }))
);

function InventoryRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LazyInventoryWorkspace />
    </DomainErrorBoundary>
  );
}

export const Route = createFileRoute('/_auth/inventory')({
  component: InventoryRoutePage,
});
