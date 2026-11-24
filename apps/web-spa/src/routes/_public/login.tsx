import * as React from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import {
  AlertTriangle,
  ArrowRight,
  Building2,
  KeyRound,
  Lock,
  Moon,
  ShieldCheck,
  Sun,
  User as UserIcon,
} from 'lucide-react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { z } from 'zod';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Select } from '../../components/ui/Select';
import { apiClient, ApiClientError } from '../../lib/api';
import {
  DEFAULT_HOLDING_ORG,
  DEFAULT_ORGANIZATIONS,
  useAuthStore,
} from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';
import type { Organization, User as UserType } from '../../types';

export const Route = createFileRoute('/_public/login')({
  component: LoginPage,
});

interface DemoPersona {
  id: string;
  username: string;
  fullName: string;
  orgCode: string;
  roles: string[];
  permissions: string[];
}

const ALL_BACKEND_PERMISSIONS: string[] = [
  'gl.*',
  'gl.create',
  'gl.read',
  'gl.update',
  'gl.approve',
  'gl.post',
  'gl.reverse',
  'gl.close_period',
  'reporting.read',
  'gl:read',
  'gl:write',
  'gl:approve',
  'ap:read',
  'ap:write',
  'ap:approve',
  'ar:read',
  'ar:write',
  'ar:approve',
  'coa:read',
  'coa:write',
  'tax:read',
  'tax:write',
  'tax:approve',
  'currency:read',
  'currency:write',
  'treasury:read',
  'treasury:write',
  'treasury:approve',
  'workflow:read',
  'workflow:write',
  'workflow:approve',
  'inventory:read',
  'inventory:write',
  'procurement:read',
  'procurement:write',
  'procurement:approve',
  'sales:read',
  'sales:write',
  'manufacturing:read',
  'manufacturing:write',
  'hr:read',
  'hr:write',
  'payroll:read',
  'payroll:write',
  'payroll:approve',
  'masterdata:read',
  'masterdata:write',
  'org:read',
  'org:write',
  'user:read',
  'user:write',
  'security:read',
  'security:write',
  'audit:read',
  'fixedasset:read',
  'fixedasset:write',
  'dimension:read',
  'dimension:write',
  'budget:read',
  'budget:write',
  'budget:approve',
  'financialclose:read',
  'financialclose:write',
  'financialclose:approve',
  'consolidation:read',
  'consolidation:write',
  'consolidation:approve',
  'project:read',
  'project:write',
  'project:approve',
  'asset:read',
  'asset:write',
  'maintenance:read',
  'maintenance:write',
  'crm:read',
  'crm:write',
  'localization:read',
  'localization:write',
];

const DEMO_PERSONAS: DemoPersona[] = [
  {
    id: '00000000-0000-0000-0000-000000000101',
    username: 'admin.finance',
    fullName: 'Chief Financial Officer',
    orgCode: 'HLD-01',
    roles: ['SYSTEM_ADMIN', 'FINANCE_MANAGER', 'ACCOUNTANT'],
    permissions: ALL_BACKEND_PERMISSIONS,
  },
  {
    id: '00000000-0000-0000-0000-000000000102',
    username: 'admin.ho',
    fullName: 'Admin Utama Head Office (Holding)',
    orgCode: 'HLD-01',
    roles: ['HOLDING_ADMIN', 'FINANCE_MANAGER'],
    permissions: ALL_BACKEND_PERMISSIONS,
  },
  {
    id: '00000000-0000-0000-0000-000000000103',
    username: 'admin.properti',
    fullName: 'Admin Anak Perusahaan (Properti)',
    orgCode: 'SUB-PRO',
    roles: ['SUBSIDIARY_ADMIN'],
    permissions: ALL_BACKEND_PERMISSIONS,
  },
  {
    id: '00000000-0000-0000-0000-000000000104',
    username: 'admin.cabang',
    fullName: 'Admin Cabang Jakarta Selatan',
    orgCode: 'BRC-JKT',
    roles: ['BRANCH_ADMIN'],
    permissions: [
      'gl.read', 'gl.create', 'ap:read', 'ap:write', 'ar:read', 'ar:write',
      'inventory:read', 'inventory:write', 'sales:read', 'sales:write',
      'procurement:read', 'procurement:write', 'user:read', 'user:write', 'org:read',
    ],
  },
  {
    id: '00000000-0000-0000-0000-000000000201',
    username: 'finance.mgr',
    fullName: 'Kepala Divisi Finance & Accounting',
    orgCode: 'HLD-01',
    roles: ['FINANCE_MANAGER', 'ACCOUNTANT'],
    permissions: [
      'gl.create',
      'gl.read',
      'gl.update',
      'gl.approve',
      'ap.create',
      'ap.read',
      'ap.update',
      'ap.approve',
      'ar.create',
      'ar.read',
      'ar.update',
      'ar.approve',
      'tax.create',
      'tax.read',
      'tax.update',
      'tax.approve',
      'treasury.create',
      'treasury.read',
      'treasury.update',
      'treasury.approve',
      'procurement.read',
      'procurement.approve',
      'inventory.read',
      'sales.read',
      'audit.read',
      'reporting.read',
      'org.read',
      'ap:read',
      'ap:write',
      'ar:read',
      'ar:write',
      'coa:read',
      'coa:write',
      'tax:read',
      'tax:write',
      'currency:read',
      'currency:write',
      'treasury:read',
      'treasury:write',
      'workflow:read',
      'workflow:write',
      'workflow:approve',
      'inventory:read',
      'procurement:read',
      'sales:read',
      'manufacturing:read',
      'payroll:read',
      'payroll:write',
      'masterdata:read',
      'masterdata:write',
      'org:read',
      'audit:read',
    ],
  },
  {
    id: '00000000-0000-0000-0000-000000000203',
    username: 'procurement.mgr',
    fullName: 'Kepala Divisi Procurement',
    orgCode: 'SUB-PRO',
    roles: ['PROCUREMENT_MANAGER'],
    permissions: ['procurement:read', 'procurement:write', 'procurement:approve', 'inventory:read', 'inventory:write', 'ap:read', 'masterdata:read', 'org:read'],
  },
  {
    id: '00000000-0000-0000-0000-000000000204',
    username: 'warehouse.mgr',
    fullName: 'Kepala Divisi Warehouse & Logistik',
    orgCode: 'SUB-LOG',
    roles: ['WAREHOUSE_MANAGER'],
    permissions: ['inventory:read', 'inventory:write', 'procurement:read', 'sales:read', 'masterdata:read', 'org:read'],
  },
  {
    id: '00000000-0000-0000-0000-000000000205',
    username: 'sales.mgr',
    fullName: 'Kepala Divisi Sales & CRM',
    orgCode: 'SUB-PRO',
    roles: ['SALES_MANAGER'],
    permissions: ['sales:read', 'sales:write', 'ar:read', 'ar:write', 'inventory:read', 'masterdata:read', 'org:read'],
  },
  {
    id: '00000000-0000-0000-0000-000000000207',
    username: 'hr.mgr',
    fullName: 'Kepala Divisi HR & Payroll',
    orgCode: 'HLD-01',
    roles: ['HR_MANAGER'],
    permissions: ['hr:read', 'hr:write', 'payroll:read', 'payroll:write', 'payroll:approve', 'workflow:read', 'workflow:approve', 'org:read', 'user:read'],
  },
  {
    id: '00000000-0000-0000-0000-000000000208',
    username: 'auditor.it',
    fullName: 'Internal Auditor & IT Governance',
    orgCode: 'HLD-01',
    roles: ['AUDITOR'],
    permissions: [
      'gl.read',
      'reporting.read',
      'ap:read',
      'ar:read',
      'coa:read',
      'tax:read',
      'currency:read',
      'treasury:read',
      'workflow:read',
      'inventory:read',
      'procurement:read',
      'sales:read',
      'manufacturing:read',
      'hr:read',
      'payroll:read',
      'masterdata:read',
      'org:read',
      'user:read',
      'security:read',
      'audit:read',
    ],
  },
];

interface LoginResponse {
  token: string;
  streamTicket?: string;
  expiresIn?: number;
  user: UserType;
  activeOrg: Organization;
  organizations: Organization[];
}

function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { setAuth, setDemoAuth } = useAuthStore();
  const { theme, toggleTheme, locale, setLocale, setDemoMode } = useUiStore();

  const loginSchema = React.useMemo(
    () =>
      z.object({
        orgCode: z.string().min(1, t('auth.validationOrgRequired')),
        username: z.string().min(2, t('auth.validationUsernameMin')),
        password: z.string().min(4, t('auth.validationPasswordMin')),
      }),
    [t]
  );

  type LoginFormValues = z.infer<typeof loginSchema>;

  const [selectedPersonaId, setSelectedPersonaId] = React.useState(DEMO_PERSONAS[0].id);
  const [serverError, setServerError] = React.useState<{
    message: string;
    correlationId?: string;
  } | null>(null);
  const [logoFailed, setLogoFailed] = React.useState(false);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      orgCode: 'HLD-01',
      username: '',
      password: '',
    },
  });

  const orgCodeValue = watch('orgCode');
  const activePersona =
    DEMO_PERSONAS.find((p) => p.id === selectedPersonaId) ?? DEMO_PERSONAS[0];

  const onSubmit = async (values: LoginFormValues) => {
    setServerError(null);
    setDemoMode(false);

    try {
      const res = await apiClient<LoginResponse>('/auth/login', {
        method: 'POST',
        skipAuthRedirect: true,
        body: JSON.stringify({
          orgCode: values.orgCode,
          username: values.username.trim(),
          password: values.password,
        }),
      });

      setAuth(res.user, res.token, res.activeOrg, res.organizations);
      void navigate({ to: '/' });
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setServerError({
          message: err.message,
          correlationId: err.correlationId,
        });
      } else {
        setServerError({
          message: err instanceof Error ? err.message : t('auth.loginFailed'),
        });
      }
    }
  };

  const handleOfflineDemoLogin = () => {
    const targetOrg =
      DEFAULT_ORGANIZATIONS.find((o) => o.code === activePersona.orgCode) ?? DEFAULT_HOLDING_ORG;
    setDemoMode(true);
    setDemoAuth(
      {
        id: activePersona.id,
        username: activePersona.username,
        fullName: activePersona.fullName,
        email: `${activePersona.username}@sinergi.example.com`,
        activeOrgId: targetOrg.id,
        roles: activePersona.roles,
        permissions: activePersona.permissions,
      },
      targetOrg,
      DEFAULT_ORGANIZATIONS
    );
    void navigate({ to: '/' });
  };

  return (
    <div className="min-h-screen grid lg:grid-cols-2 bg-slate-50 dark:bg-slate-950">
      {/* Left Enterprise Brand Panel */}
      <div className="hidden lg:flex flex-col justify-between bg-brand-900 text-white p-10 relative overflow-hidden">
        <div
          aria-hidden="true"
          className="absolute inset-0 opacity-20 bg-[radial-gradient(circle_at_20%_20%,#3b82f6,transparent_55%),radial-gradient(circle_at_80%_80%,#0ea5e9,transparent_45%)]"
        />

        <div className="relative z-10 flex items-center gap-3">
          {!logoFailed ? (
            <img
              src="/brand-logo.png"
              alt={t('app.title')}
              onError={() => setLogoFailed(true)}
              className="h-10 w-10 rounded-xl bg-white/10 p-1 object-contain"
            />
          ) : (
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/15 font-bold text-sm">
              ERP
            </div>
          )}
          <div>
            <div className="text-base font-bold tracking-tight">{t('app.title')}</div>
            <div className="text-xs text-brand-200">{t('app.subtitle')}</div>
          </div>
        </div>

        <div className="relative z-10 max-w-md space-y-5">
          <div className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-medium text-brand-100">
            <ShieldCheck className="h-3.5 w-3.5 text-emerald-400" />
            <span>30 Microservices · Row-Level Security · Fixed-Point BigInt</span>
          </div>
          <h2 className="text-3xl font-bold tracking-tight leading-tight">
            {t('auth.welcomeBack')}
          </h2>
          <p className="text-sm text-brand-100/85 leading-relaxed">
            {t('auth.subtitle')}
          </p>
        </div>

        <div className="relative z-10 flex items-center justify-between text-xs text-brand-200/80 border-t border-white/10 pt-5">
          <span>PT Sinergi Investasi Properti Tbk</span>
          <span className="font-mono">API Gateway v1</span>
        </div>
      </div>

      {/* Right Login Card */}
      <div className="flex flex-col justify-center items-center p-5 sm:p-10 relative">
        {/* Top-right Utility Toggles */}
        <div className="absolute top-4 right-4 sm:top-6 sm:right-6 flex items-center gap-2">
          <button
            type="button"
            onClick={() => setLocale(locale === 'id' ? 'en' : 'id')}
            className="inline-flex h-8 items-center rounded-lg border border-slate-200 bg-white px-2.5 text-xs font-mono font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200 cursor-pointer"
          >
            {locale.toUpperCase()}
          </button>
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={theme === 'dark' ? t('app.lightMode') : t('app.darkMode')}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 cursor-pointer"
          >
            {theme === 'dark' ? (
              <Sun className="h-4 w-4 text-amber-400" />
            ) : (
              <Moon className="h-4 w-4" />
            )}
          </button>
        </div>

        <div className="w-full max-w-md space-y-6">
          <div className="rounded-2xl border border-slate-200 bg-white p-6 sm:p-8 shadow-sm dark:border-slate-800 dark:bg-slate-900 space-y-5">
            <div className="space-y-1">
              <div className="inline-flex p-2.5 rounded-xl bg-brand-50 text-brand-700 dark:bg-brand-950/60 dark:text-brand-400 mb-1">
                <Lock className="h-5 w-5" />
              </div>
              <h1 className="text-xl font-bold text-slate-900 dark:text-slate-100">
                {t('auth.welcomeBack')}
              </h1>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {t('app.subtitle')}
              </p>
            </div>

            {serverError ? (
              <div
                role="alert"
                className="rounded-xl border border-rose-200 bg-rose-50 px-3.5 py-2.5 text-xs text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200 space-y-1"
              >
                <div className="font-semibold">{serverError.message}</div>
                {serverError.correlationId ? (
                  <div className="font-mono text-[10px] opacity-80">
                    CID: {serverError.correlationId}
                  </div>
                ) : null}
              </div>
            ) : null}

            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
              <FormField
                label={t('app.orgSwitch')}
                htmlFor="orgCode"
                required
                error={errors.orgCode?.message}
              >
                <Select
                  id="orgCode"
                  value={orgCodeValue}
                  onValueChange={(v) => setValue('orgCode', v, { shouldValidate: true })}
                  options={[
                    {
                      value: 'HLD-01',
                      label: 'HLD-01 · PT Sinergi Investasi Properti (Head Office / Holding)',
                    },
                    {
                      value: 'SUB-PRO',
                      label: 'SUB-PRO · PT Sinergi Properti Jakarta (Anak Perusahaan)',
                    },
                    {
                      value: 'SUB-LOG',
                      label: 'SUB-LOG · PT Sinergi Logistik Nusantara (Anak Perusahaan)',
                    },
                    {
                      value: 'BRC-JKT',
                      label: 'BRC-JKT · Cabang Jakarta Selatan (Branch)',
                    },
                  ]}
                />
              </FormField>

              <FormField
                label={t('auth.usernameLabel')}
                htmlFor="username"
                required
                error={errors.username?.message}
              >
                <Input
                  id="username"
                  autoComplete="username"
                  placeholder={t('auth.usernamePlaceholder')}
                  leftIcon={<UserIcon className="h-4 w-4" />}
                  error={Boolean(errors.username)}
                  {...register('username')}
                />
              </FormField>

              <FormField
                label={t('auth.passwordLabel')}
                htmlFor="password"
                required
                error={errors.password?.message}
              >
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder={t('auth.passwordPlaceholder')}
                  leftIcon={<KeyRound className="h-4 w-4" />}
                  error={Boolean(errors.password)}
                  {...register('password')}
                />
              </FormField>

              <Button
                type="submit"
                variant="primary"
                size="lg"
                isLoading={isSubmitting}
                rightIcon={<ArrowRight className="h-4 w-4" />}
                className="w-full"
              >
                {isSubmitting ? t('auth.signingIn') : t('auth.signIn')}
              </Button>
            </form>

            {/* Explicit Offline Demo Sandbox Section */}
            <div className="relative pt-2">
              <div className="relative flex justify-center text-[11px] uppercase">
                <span className="bg-white dark:bg-slate-900 px-2 text-slate-400">
                  {t('auth.orDivider')}
                </span>
              </div>
            </div>

            <div className="rounded-xl border border-amber-200/80 bg-amber-50/50 p-3.5 space-y-3 dark:border-amber-900/50 dark:bg-amber-950/20">
              <div className="flex items-start gap-2 text-[11px] text-amber-800 dark:text-amber-300">
                <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5 text-amber-600 dark:text-amber-400" />
                <span>{t('auth.demoHint')}</span>
              </div>

              <div className="space-y-1.5">
                <span className="block text-[10px] font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  {t('auth.chooseDemoPersona')}
                </span>
                <div className="grid grid-cols-3 gap-1.5">
                  {DEMO_PERSONAS.map((persona) => {
                    const selected = persona.id === activePersona.id;
                    return (
                      <button
                        key={persona.id}
                        type="button"
                        onClick={() => setSelectedPersonaId(persona.id)}
                        className={
                          selected
                            ? 'rounded-lg border border-brand-600 bg-brand-50 px-2 py-1.5 text-[11px] font-semibold text-brand-800 dark:border-brand-500 dark:bg-brand-950/60 dark:text-brand-300 cursor-pointer truncate'
                            : 'rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-[11px] font-medium text-slate-600 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300 cursor-pointer truncate'
                        }
                        title={persona.fullName}
                      >
                        <Building2 className="inline h-3 w-3 mr-1 opacity-70" />
                        {persona.username.split('.')[0]}
                      </button>
                    );
                  })}
                </div>
              </div>

              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleOfflineDemoLogin}
                className="w-full border-amber-300 bg-white hover:bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-slate-900 dark:text-amber-200 dark:hover:bg-slate-800"
              >
                {t('auth.demoButton')} ({activePersona.username})
              </Button>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-[11px] text-slate-500 dark:text-slate-400">
            <a href="/privacy" className="hover:text-brand-600 dark:hover:text-brand-400 underline-offset-2 hover:underline">
              Privacy Policy
            </a>
            <span aria-hidden="true">·</span>
            <a href="/terms" className="hover:text-brand-600 dark:hover:text-brand-400 underline-offset-2 hover:underline">
              Terms of Service
            </a>
            <span aria-hidden="true">·</span>
            <a href="/delete-account" className="hover:text-brand-600 dark:hover:text-brand-400 underline-offset-2 hover:underline">
              Delete Account
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
