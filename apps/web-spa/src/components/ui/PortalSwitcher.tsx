import { useNavigate } from '@tanstack/react-router';
import { Briefcase, ShieldCheck, BarChart3 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '../../lib/utils';
import { filterNavByPortalAndPermission } from '../../lib/permissions';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { PortalMode } from '../../types';

export interface PortalSwitcherProps {
  compact?: boolean;
  onSwitched?: (portal: PortalMode) => void;
  className?: string;
}

const PORTAL_META: Array<{
  id: PortalMode;
  labelKey: string;
  shortKey: string;
  icon: typeof Briefcase;
}> = [
  {
    id: 'backoffice',
    labelKey: 'portals.backoffice',
    shortKey: 'portals.backofficeShort',
    icon: Briefcase,
  },
  {
    id: 'admin',
    labelKey: 'portals.admin',
    shortKey: 'portals.adminShort',
    icon: ShieldCheck,
  },
  {
    id: 'reporting',
    labelKey: 'portals.reporting',
    shortKey: 'portals.reportingShort',
    icon: BarChart3,
  },
];

export function PortalSwitcher({ compact = false, onSwitched, className }: PortalSwitcherProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { portal, setPortal } = useUiStore();
  const { user } = useAuthStore();

  const handleSelect = (nextPortal: PortalMode) => {
    setPortal(nextPortal);
    const items = filterNavByPortalAndPermission(nextPortal, user);
    const landing = items[0]?.to;
    if (landing) {
      void navigate({ to: landing as '/' });
    }
    onSwitched?.(nextPortal);
  };

  return (
    <div
      role="tablist"
      aria-label={t('app.portalSwitch')}
      className={cn(
        'inline-flex items-center gap-0.5 rounded-xl bg-surface-muted p-1 border border-border',
        compact && 'w-full',
        className
      )}
    >
      {PORTAL_META.map((item) => {
        const Icon = item.icon;
        const active = portal === item.id;
        return (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => handleSelect(item.id)}
            title={t(item.labelKey)}
            className={cn(
              'inline-flex items-center justify-center gap-1.5 rounded-lg px-2.5 py-1 text-xs font-semibold transition-all cursor-pointer',
              compact && 'flex-1 px-2',
              active
                ? 'bg-surface text-brand-700 border border-border shadow-2xs dark:text-brand-300'
                : 'border border-transparent text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-200'
            )}
          >
            <Icon className="h-3.5 w-3.5 shrink-0" />
            <span>{compact ? t(item.shortKey) : t(item.shortKey)}</span>
          </button>
        );
      })}
    </div>
  );
}
