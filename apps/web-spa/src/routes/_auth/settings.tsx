import { createFileRoute } from '@tanstack/react-router';
import { SettingsWorkspace } from '../../features/admin/SettingsWorkspace';

export const Route = createFileRoute('/_auth/settings')({
  component: SettingsPage,
});

function SettingsPage() {
  return <SettingsWorkspace />;
}
