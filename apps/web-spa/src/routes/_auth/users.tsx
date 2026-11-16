import { createFileRoute } from '@tanstack/react-router';
import { UsersWorkspace } from '../../features/admin/UsersWorkspace';

export const Route = createFileRoute('/_auth/users')({
  component: UsersPage,
});

function UsersPage() {
  return <UsersWorkspace />;
}
