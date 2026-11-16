import { createFileRoute } from '@tanstack/react-router';
import { HrWorkspace } from '../../features/hr/HrWorkspace';

export const Route = createFileRoute('/_auth/hr')({
  component: HrPage,
});

function HrPage() {
  return <HrWorkspace />;
}
