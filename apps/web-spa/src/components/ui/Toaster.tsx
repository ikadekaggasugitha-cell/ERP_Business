import { Toaster as SonnerToaster } from 'sonner';
import { useUiStore } from '../../store/ui-store';

export function Toaster() {
  const { theme } = useUiStore();

  return (
    <SonnerToaster
      theme={theme}
      position="bottom-right"
      richColors
      closeButton
      duration={4500}
      toastOptions={{
        classNames: {
          toast:
            'font-sans border border-slate-200 dark:border-slate-800 shadow-lg rounded-xl',
        },
      }}
    />
  );
}
