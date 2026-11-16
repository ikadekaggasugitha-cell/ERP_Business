import { createFileRoute } from '@tanstack/react-router';
import { OrganizationsWorkspace } from '../../features/admin/OrganizationsWorkspace';

export const Route = createFileRoute('/_auth/organizations')({
  component: OrganizationsPage,
});

function OrganizationsPage() {
  return <OrganizationsWorkspace />;
}
