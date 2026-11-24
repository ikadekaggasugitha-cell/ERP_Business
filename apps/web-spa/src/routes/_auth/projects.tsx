import { createFileRoute } from '@tanstack/react-router';
import { DomainErrorBoundary } from '../../components/ui/DomainErrorBoundary';
import { ProjectsWorkspace } from '../../features/project/ProjectsWorkspace';
import { useAuthStore } from '../../store/auth-store';

export const Route = createFileRoute('/_auth/projects')({
  component: ProjectsPage,
});

function ProjectsPage() {
  const activeOrgId = useAuthStore((s) => s.activeOrg?.id ?? 'default');
  return (
    <DomainErrorBoundary resetKey={activeOrgId}>
      <ProjectsWorkspace />
    </DomainErrorBoundary>
  );
}
