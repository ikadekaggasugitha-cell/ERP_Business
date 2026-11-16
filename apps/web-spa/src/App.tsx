import * as React from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { I18nextProvider } from 'react-i18next';
import { TooltipProvider } from './components/ui/Tooltip';
import i18n from './lib/i18n';
import { queryClient } from './lib/query-client';
import { router } from './lib/router';
import { useAuthStore } from './store/auth-store';

function InnerRouter() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const user = useAuthStore((s) => s.user);
  const activeOrg = useAuthStore((s) => s.activeOrg);

  const authContext = React.useMemo(
    () => ({ isAuthenticated, user, activeOrg }),
    [isAuthenticated, user, activeOrg]
  );

  return <RouterProvider router={router} context={{ queryClient, auth: authContext }} />;
}

export function App() {
  return (
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={250}>
          <InnerRouter />
        </TooltipProvider>
      </QueryClientProvider>
    </I18nextProvider>
  );
}
