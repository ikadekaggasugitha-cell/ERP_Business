import { createFileRoute } from '@tanstack/react-router';
import { WorkflowWorkspace } from '../../features/workflow/WorkflowWorkspace';

export const Route = createFileRoute('/_auth/workflow')({
  component: WorkflowPage,
});

function WorkflowPage() {
  return <WorkflowWorkspace />;
}
