import { describe, expect, it } from 'vitest';
import { filterNavByPortalAndPermission, hasUserPermission } from '../../src/lib/permissions';

describe('RBAC Permission Helper', () => {
  it('normalizes dot and colon permissions accurately', () => {
    const accountant = { roles: ['ACCOUNTANT'], permissions: ['gl.*', 'ap:read', 'coa.write'] };
    expect(hasUserPermission(accountant, 'gl.post')).toBe(true);
    expect(hasUserPermission(accountant, 'gl:read')).toBe(true);
    expect(hasUserPermission(accountant, 'ap.read')).toBe(true);
    expect(hasUserPermission(accountant, 'ap:write')).toBe(false);
    expect(hasUserPermission(accountant, 'coa:write')).toBe(true);
  });

  it('filters sidebar modules by portal and RBAC permissions', () => {
    const viewer = { roles: ['VIEWER'], permissions: ['reporting.read'] };
    const reportingNav = filterNavByPortalAndPermission('reporting', viewer);
    expect(reportingNav.some((m) => m.id === 'reports')).toBe(true);

    const adminNav = filterNavByPortalAndPermission('admin', viewer);
    expect(adminNav.some((m) => m.id === 'security')).toBe(false);
    expect(adminNav.some((m) => m.id === 'settings')).toBe(true);
  });

  it('prevents write permissions from escalating to approve permissions', () => {
    const accountant = {
      roles: ['ACCOUNTANT'],
      permissions: ['workflow:write', 'gl.update', 'gl.write'],
    };
    expect(hasUserPermission(accountant, 'workflow:write')).toBe(true);
    expect(hasUserPermission(accountant, 'workflow:approve')).toBe(false);
    expect(hasUserPermission(accountant, 'gl.approve')).toBe(false);

    const manager = {
      roles: ['FINANCE_MANAGER'],
      permissions: ['workflow:approve', 'gl.approve'],
    };
    expect(hasUserPermission(manager, 'workflow:approve')).toBe(true);
    expect(hasUserPermission(manager, 'gl.approve')).toBe(true);
  });
});
