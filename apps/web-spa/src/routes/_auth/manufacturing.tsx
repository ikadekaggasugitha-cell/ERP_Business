import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { useAuthStore } from '../../store/auth-store';

const LazyManufacturingWorkspace = React.lazy(() =>
  import('../../features/manufacturing/ManufacturingWorkspace').then((m) => ({
    default: m.ManufacturingWorkspace,
  }))
);

function ManufacturingRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LazyManufacturingWorkspace />
    </DomainErrorBoundary>
  );
}

export const Route = createFileRoute('/_auth/manufacturing')({
  component: ManufacturingRoutePage,
});
