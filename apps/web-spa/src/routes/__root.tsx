import * as React from 'react';
import { createRootRouteWithContext, Link, Outlet, useNavigate } from '@tanstack/react-router';
import type { QueryClient } from '@tanstack/react-query';
import { ArrowLeft, FileQuestion, Home } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { CommandPalette } from '../components/ui/CommandPalette';
import { Toaster } from '../components/ui/Toaster';
import { onErpEvent } from '../lib/event-bus';
import { useAuthStore } from '../store/auth-store';
import { useUiStore } from '../store/ui-store';
import type { Organization, User } from '../types';

export interface RouterContext {
  queryClient: QueryClient;
  auth: {
    isAuthenticated: boolean;
    user: User | null;
    activeOrg: Organization | null;
  };
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
  notFoundComponent: NotFoundComponent,
});

function RootComponent() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const logout = useAuthStore((s) => s.logout);
  const setDemoMode = useUiStore((s) => s.setDemoMode);

  React.useEffect(() => {
    const offUnauthorized = onErpEvent('erp:unauthorized', () => {
      setDemoMode(false);
      logout();
      toast.error(t('auth.sessionExpired'));
      void navigate({ to: '/login' });
    });

    const offConflict = onErpEvent('erp:conflict', (detail) => {
      toast.warning(t('auth.conflictError'), {
        description: detail.message,
      });
    });

    return () => {
      offUnauthorized();
      offConflict();
    };
  }, [logout, navigate, setDemoMode, t]);

  return (
    <div className="min-h-screen bg-slate-50 font-sans text-slate-900 antialiased dark:bg-slate-950 dark:text-slate-100">
      <Outlet />
      <CommandPalette />
      <Toaster />
    </div>
  );
}

function NotFoundComponent() {
  const { t } = useTranslation();

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-50 p-6 dark:bg-slate-950">
      <div className="max-w-md w-full text-center bg-white border border-slate-200 rounded-2xl p-8 shadow-sm space-y-4 dark:bg-slate-900 dark:border-slate-800">
        <div className="inline-flex p-3 rounded-xl bg-slate-100 text-brand-600 dark:bg-slate-800 dark:text-brand-400">
          <FileQuestion className="h-8 w-8" />
        </div>
        <h2 className="text-xl font-bold text-slate-900 dark:text-slate-100">
          {t('errors.notFoundTitle')}
        </h2>
        <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
          {t('errors.notFoundDesc')}
        </p>
        <div className="pt-2 flex items-center justify-center gap-3">
          <button
            type="button"
            onClick={() => window.history.back()}
            className="flex items-center gap-2 px-3 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg transition-colors cursor-pointer dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
          >
            <ArrowLeft className="h-4 w-4" />
            <span>{t('actions.back')}</span>
          </button>
          <Link
            to="/"
            className="flex items-center gap-2 px-3 py-2 text-xs font-semibold text-white bg-brand-700 hover:bg-brand-800 rounded-lg transition-colors cursor-pointer dark:bg-brand-600 dark:hover:bg-brand-500"
          >
            <Home className="h-4 w-4" />
            <span>{t('actions.dashboard')}</span>
          </Link>
        </div>
      </div>
    </div>
  );
}
