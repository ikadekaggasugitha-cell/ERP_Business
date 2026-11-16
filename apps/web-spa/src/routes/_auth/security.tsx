import { createFileRoute } from '@tanstack/react-router';
import { SecurityWorkspace } from '../../features/admin/SecurityWorkspace';

export const Route = createFileRoute('/_auth/security')({
  component: SecurityPage,
});

function SecurityPage() {
  return <SecurityWorkspace />;
}
