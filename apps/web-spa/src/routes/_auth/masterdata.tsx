import * as React from 'react';
import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { useAuthStore } from '../../store/auth-store';

const LazyMasterdataWorkspace = React.lazy(() =>
  import('../../features/masterdata/MasterdataWorkspace').then((m) => ({
    default: m.MasterdataWorkspace,
  }))
);

function MasterdataRoutePage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default-org');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <LazyMasterdataWorkspace />
    </DomainErrorBoundary>
  );
}

export const Route = createFileRoute('/_auth/masterdata')({
  component: MasterdataRoutePage,
});
