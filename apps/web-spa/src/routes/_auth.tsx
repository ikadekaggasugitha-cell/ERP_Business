import { createFileRoute, Outlet, redirect, useNavigate, useRouterState } from '@tanstack/react-router';
import { AlertTriangle } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { MobileBottomNav } from '../components/layout/MobileBottomNav';
import { Navbar } from '../components/layout/Navbar';
import { Sidebar } from '../components/layout/Sidebar';
import { Button } from '../components/ui/Button';
import { DomainErrorBoundary } from '../components/ui/DomainErrorBoundary';
import { useRealtimeEvents } from '../lib/use-realtime-events';
import { cn } from '../lib/utils';
import { useAuthStore } from '../store/auth-store';
import { useUiStore } from '../store/ui-store';

export const Route = createFileRoute('/_auth')({
  beforeLoad: async () => {
    const store = useAuthStore.getState();
    const ok = store.isAuthenticated ? true : await store.hydrateSession();
    if (!ok) {
      throw redirect({ to: '/login' });
    }
  },
  component: AuthLayout,
});

function AuthLayout() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const { sidebarCollapsed, demoMode, setDemoMode } = useUiStore();
  const { activeOrg, isDemoSession, logout } = useAuthStore();

  useRealtimeEvents();

  const handleExitDemo = () => {
    setDemoMode(false);
    logout();
    void navigate({ to: '/login' });
  };

  return (
    <div className="min-h-screen flex flex-col bg-canvas">
      <Navbar />
      <div className="flex flex-1 min-w-0">
        <Sidebar />
        <main
          className={cn(
            'flex-1 min-w-0 p-4 sm:p-6 pb-24 md:pb-8 transition-all duration-200',
            sidebarCollapsed ? 'md:ml-16' : 'md:ml-64'
          )}
        >
          <div className="max-w-[1400px] mx-auto space-y-5">
            {isDemoSession || demoMode ? (
              <div
                role="status"
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-xs text-amber-900 dark:border-amber-800/70 dark:bg-amber-950/40 dark:text-amber-200"
              >
                <div className="flex items-center gap-2 min-w-0">
                  <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400 shrink-0" />
                  <div className="min-w-0">
                    <span className="font-bold">{t('app.demoBannerTitle')}: </span>
                    <span className="text-amber-800 dark:text-amber-300">
                      {t('app.demoBannerDesc')}
                    </span>
                  </div>
                </div>
                <Button variant="outline" size="xs" onClick={handleExitDemo}>
                  {t('app.exitDemo')}
                </Button>
              </div>
            ) : null}

            <DomainErrorBoundary resetKey={`${pathname}:${activeOrg?.id ?? ''}`}>
              <Outlet />
            </DomainErrorBoundary>
          </div>
        </main>
      </div>
      <MobileBottomNav />
    </div>
  );
}
