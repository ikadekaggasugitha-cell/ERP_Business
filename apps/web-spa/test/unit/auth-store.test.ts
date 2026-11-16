import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { queryClient } from '../../src/lib/query-client';
import { DEFAULT_BRANCH_ORG, DEFAULT_HOLDING_ORG, useAuthStore } from '../../src/store/auth-store';

describe('useAuthStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      token: null,
      activeOrg: null,
      organizations: [DEFAULT_HOLDING_ORG, DEFAULT_BRANCH_ORG],
      isAuthenticated: false,
      isHydrating: false,
      hasHydrated: false,
      isDemoSession: false,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('hydrates session from GET /auth/me cookie without throwing when authenticated', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        user: {
          id: 'u-1',
          username: 'cfo',
          fullName: 'Chief Financial Officer',
          email: 'cfo@sinergi.example.com',
          activeOrgId: DEFAULT_HOLDING_ORG.id,
          roles: ['FINANCE_MANAGER'],
          permissions: ['gl.*', 'ap:read'],
        },
        activeOrg: DEFAULT_HOLDING_ORG,
        organizations: [DEFAULT_HOLDING_ORG, DEFAULT_BRANCH_ORG],
      }),
    });
    vi.stubGlobal('fetch', mockFetch);

    const ok = await useAuthStore.getState().hydrateSession(true);
    expect(ok).toBe(true);
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(useAuthStore.getState().hasPermission('gl.post')).toBe(true);
    expect(useAuthStore.getState().hasPermission('security:write')).toBe(false);
  });

  it('clears authentication cleanly on 401 during hydration', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
      })
    );

    const ok = await useAuthStore.getState().hydrateSession(true);
    expect(ok).toBe(false);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().hasHydrated).toBe(true);
  });

  it('clears queryClient cache on logout and invalidates queries on org switch', () => {
    const clearSpy = vi.spyOn(queryClient, 'clear');
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    useAuthStore.getState().setDemoAuth(
      {
        id: 'u-1',
        username: 'cfo',
        fullName: 'Chief Financial Officer',
        email: 'cfo@sinergi.example.com',
        activeOrgId: DEFAULT_HOLDING_ORG.id,
        roles: ['FINANCE_MANAGER'],
        permissions: ['gl.*'],
      },
      DEFAULT_HOLDING_ORG,
      [DEFAULT_HOLDING_ORG, DEFAULT_BRANCH_ORG]
    );

    useAuthStore.getState().setActiveOrg(DEFAULT_BRANCH_ORG);
    expect(invalidateSpy).toHaveBeenCalledTimes(1);

    useAuthStore.getState().logout();
    expect(clearSpy).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });
});
