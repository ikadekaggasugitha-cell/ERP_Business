import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_public')({
  component: PublicLayout,
});

function PublicLayout() {
  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Outlet />
    </div>
  );
}
