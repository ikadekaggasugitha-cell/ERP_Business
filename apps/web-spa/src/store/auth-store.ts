import { create } from 'zustand';
import { emitErpEvent } from '../lib/event-bus';
import { hasUserPermission } from '../lib/permissions';
import { queryClient } from '../lib/query-client';
import type { AuthSessionResponse, Organization, User } from '../types';

export const DEFAULT_HOLDING_ORG: Organization = {
  id: '11111111-1111-1111-1111-111111111111',
  code: 'HLD-01',
  name: 'PT Sinergi Investasi Properti (Head Office / Holding)',
  type: 'HOLDING',
  parentId: null,
  functionalCurrency: 'IDR',
};

export const DEFAULT_SUB_PRO_ORG: Organization = {
  id: '22222222-2222-2222-2222-222222222222',
  code: 'SUB-PRO',
  name: 'PT Sinergi Properti Jakarta (Anak Perusahaan)',
  type: 'COMPANY',
  parentId: '11111111-1111-1111-1111-111111111111',
  functionalCurrency: 'IDR',
};

export const DEFAULT_SUB_LOG_ORG: Organization = {
  id: '33333333-3333-3333-3333-333333333333',
  code: 'SUB-LOG',
  name: 'PT Sinergi Logistik Nusantara (Anak Perusahaan)',
  type: 'COMPANY',
  parentId: '11111111-1111-1111-1111-111111111111',
  functionalCurrency: 'IDR',
};

export const DEFAULT_BRANCH_ORG: Organization = {
  id: '44444444-4444-4444-4444-444444444444',
  code: 'BRC-JKT',
  name: 'Cabang Jakarta Selatan (Branch)',
  type: 'BRANCH',
  parentId: '22222222-2222-2222-2222-222222222222',
  functionalCurrency: 'IDR',
};

export const DEFAULT_DIVISIONS: Organization[] = [
  {
    id: '11111111-0000-4000-8000-000000000001',
    code: 'DIV-FIN',
    name: 'Divisi Finance & Accounting',
    type: 'DIVISION',
    parentId: '11111111-1111-1111-1111-111111111111',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000002',
    code: 'DIV-TAX',
    name: 'Divisi Tax & Treasury',
    type: 'DIVISION',
    parentId: '11111111-1111-1111-1111-111111111111',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000003',
    code: 'DIV-PRC',
    name: 'Divisi Procurement & Purchasing',
    type: 'DIVISION',
    parentId: '22222222-2222-2222-2222-222222222222',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000004',
    code: 'DIV-LOG',
    name: 'Divisi Warehouse & Logistics',
    type: 'DIVISION',
    parentId: '33333333-3333-3333-3333-333333333333',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000005',
    code: 'DIV-SLS',
    name: 'Divisi Sales & CRM',
    type: 'DIVISION',
    parentId: '22222222-2222-2222-2222-222222222222',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000006',
    code: 'DIV-OPS',
    name: 'Divisi Operations, Project & Maintenance',
    type: 'DIVISION',
    parentId: '22222222-2222-2222-2222-222222222222',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000007',
    code: 'DIV-HR',
    name: 'Divisi HR & Payroll',
    type: 'DIVISION',
    parentId: '11111111-1111-1111-1111-111111111111',
    functionalCurrency: 'IDR',
  },
  {
    id: '11111111-0000-4000-8000-000000000008',
    code: 'DIV-AUD',
    name: 'Divisi IT & Internal Audit',
    type: 'DIVISION',
    parentId: '11111111-1111-1111-1111-111111111111',
    functionalCurrency: 'IDR',
  },
];

export const DEFAULT_ORGANIZATIONS: Organization[] = [
  DEFAULT_HOLDING_ORG,
  DEFAULT_SUB_PRO_ORG,
  DEFAULT_SUB_LOG_ORG,
  DEFAULT_BRANCH_ORG,
  ...DEFAULT_DIVISIONS,
];

const ACTIVE_ORG_STORAGE_KEY = 'erp_active_org_id';

function resolveApiBase(): string {
  const envUrl =
    typeof import.meta !== 'undefined' && import.meta.env
      ? (import.meta.env.VITE_API_URL as string | undefined)
      : undefined;
  if (!envUrl) return '/api/v1';
  const trimmed = envUrl.replace(/\/+$/, '');
  return trimmed.endsWith('/api/v1') ? trimmed : `${trimmed}/api/v1`;
}

function getSavedOrgId(): string | null {
  if (typeof window === 'undefined') return null;
  try {
    return window.sessionStorage.getItem(ACTIVE_ORG_STORAGE_KEY);
  } catch {
    return null;
  }
}

function saveOrgId(orgId: string | null): void {
  if (typeof window === 'undefined') return;
  try {
    if (orgId) {
      window.sessionStorage.setItem(ACTIVE_ORG_STORAGE_KEY, orgId);
    } else {
      window.sessionStorage.removeItem(ACTIVE_ORG_STORAGE_KEY);
    }
  } catch {
    // Ignore storage errors
  }
}

interface AuthState {
  user: User | null;
  token: string | null;
  activeOrg: Organization | null;
  organizations: Organization[];
  isAuthenticated: boolean;
  isHydrating: boolean;
  hasHydrated: boolean;
  isDemoSession: boolean;
  setAuth: (user: User, token: string, activeOrg: Organization, organizations?: Organization[]) => void;
  setDemoAuth: (user: User, activeOrg: Organization, organizations?: Organization[]) => void;
  setActiveOrg: (org: Organization) => void;
  setOrganizations: (organizations: Organization[]) => void;
  hydrateSession: (force?: boolean) => Promise<boolean>;
  logout: () => void;
  hasPermission: (permission: string | string[]) => boolean;
}

let inflightHydration: Promise<boolean> | null = null;

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  token: null,
  activeOrg: null,
  organizations: DEFAULT_ORGANIZATIONS,
  isAuthenticated: false,
  isHydrating: false,
  hasHydrated: false,
  isDemoSession: false,

  setAuth: (user, token, activeOrg, organizations) => {
    const prevUserId = get().user?.id;
    if (prevUserId && prevUserId !== user.id) {
      queryClient.clear();
    }
    const isDemo = token === 'dev-jwt-offline-token';
    saveOrgId(activeOrg.id);
    emitErpEvent('erp:org-changed', { org: activeOrg });
    set({
      user,
      token: isDemo ? null : token || null,
      activeOrg,
      organizations: organizations && organizations.length > 0 ? organizations : [activeOrg],
      isAuthenticated: true,
      hasHydrated: true,
      isHydrating: false,
      isDemoSession: isDemo,
    });
  },

  setDemoAuth: (user, activeOrg, organizations) => {
    const prevUserId = get().user?.id;
    if (prevUserId && prevUserId !== user.id) {
      queryClient.clear();
    }
    saveOrgId(activeOrg.id);
    emitErpEvent('erp:org-changed', { org: activeOrg });
    set({
      user,
      token: 'dev-jwt-offline-token',
      activeOrg,
      organizations: organizations && organizations.length > 0 ? organizations : [activeOrg],
      isAuthenticated: true,
      hasHydrated: true,
      isHydrating: false,
      isDemoSession: true,
    });
  },

  setActiveOrg: (activeOrg) => {
    const prevOrgId = get().activeOrg?.id;
    saveOrgId(activeOrg.id);
    emitErpEvent('erp:org-changed', { org: activeOrg });
    set((state) => ({
      activeOrg,
      user: state.user ? { ...state.user, activeOrgId: activeOrg.id } : null,
    }));
    if (prevOrgId && prevOrgId !== activeOrg.id) {
      void queryClient.invalidateQueries();
    }
  },

  setOrganizations: (organizations) =>
    set({
      organizations,
    }),

  hydrateSession: async (force = false) => {
    const current = get();
    if (current.isDemoSession && current.isAuthenticated) {
      if (!current.hasHydrated) {
        set({ hasHydrated: true, isHydrating: false });
      }
      return true;
    }

    if (!force && current.hasHydrated && !current.isHydrating) {
      return current.isAuthenticated;
    }

    if (inflightHydration) {
      return inflightHydration;
    }

    set({ isHydrating: true });

    inflightHydration = (async () => {
      try {
        const res = await fetch(`${resolveApiBase()}/auth/me`, {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        });

        if (!res.ok) {
          set({
            user: null,
            token: null,
            activeOrg: null,
            isAuthenticated: false,
            isHydrating: false,
            hasHydrated: true,
            isDemoSession: false,
          });
          return false;
        }

        const data = (await res.json()) as AuthSessionResponse;
        const orgs =
          data.organizations && data.organizations.length > 0
            ? data.organizations
            : data.activeOrg
              ? [data.activeOrg]
              : [DEFAULT_HOLDING_ORG];

        const preferredOrgId = getSavedOrgId();
        const resolvedOrg =
          (preferredOrgId ? orgs.find((o) => o.id === preferredOrgId) : undefined) ||
          data.activeOrg ||
          orgs[0] ||
          DEFAULT_HOLDING_ORG;

        saveOrgId(resolvedOrg.id);
        emitErpEvent('erp:org-changed', { org: resolvedOrg });

        set((state) => ({
          user: { ...data.user, activeOrgId: resolvedOrg.id },
          token: data.token ?? state.token,
          activeOrg: resolvedOrg,
          organizations: orgs,
          isAuthenticated: true,
          isHydrating: false,
          hasHydrated: true,
          isDemoSession: false,
        }));
        return true;
      } catch {
        set({
          user: null,
          token: null,
          activeOrg: null,
          isAuthenticated: false,
          isHydrating: false,
          hasHydrated: true,
          isDemoSession: false,
        });
        return false;
      } finally {
        inflightHydration = null;
      }
    })();

    return inflightHydration;
  },

  logout: () => {
    const wasDemo = get().isDemoSession;
    saveOrgId(null);
    queryClient.clear();
    emitErpEvent('erp:org-changed', { org: null });
    set({
      user: null,
      token: null,
      activeOrg: null,
      isAuthenticated: false,
      isHydrating: false,
      hasHydrated: true,
      isDemoSession: false,
    });

    if (!wasDemo && typeof fetch === 'function') {
      void fetch(`${resolveApiBase()}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      }).catch(() => {
        // Ignore network errors during best-effort cookie logout
      });
    }
  },

  hasPermission: (permission) => {
    return hasUserPermission(get().user, permission);
  },
}));
