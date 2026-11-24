import type { PortalMode, User } from '../types';

const ADMIN_ROLES = new Set([
  'superadmin',
  'admin',
  'SYSTEM_ADMIN',
  'SUPER_ADMIN',
  'HOLDING_ADMIN',
]);

export function getAdminTierLabel(
  user: Pick<User, 'roles' | 'adminTier'> | null | undefined
): string | null {
  if (!user) return null;
  const roles = new Set((user.roles ?? []).map((r) => r.trim().toUpperCase()));
  if (user.adminTier === 0 || roles.has('SYSTEM_ADMIN')) return 'System Admin';
  if (user.adminTier === 1 || roles.has('HOLDING_ADMIN')) return 'Head Office Admin';
  if (user.adminTier === 2 || roles.has('SUBSIDIARY_ADMIN')) return 'Subsidiary Admin';
  if (user.adminTier === 3 || roles.has('BRANCH_ADMIN')) return 'Branch Admin';
  return null;
}

const WRITE_ACTION_ALIASES: Record<string, string[]> = {
  write: ['write', 'create', 'update', 'delete', 'post', 'reverse'],
  create: ['create', 'write'],
  update: ['update', 'write'],
  delete: ['delete', 'write'],
  approve: ['approve'],
  read: ['read', 'view', 'list'],
};

function parsePerm(perm: string): { domain: string; action: string } | null {
  const clean = perm.trim().toLowerCase();
  if (!clean) return null;
  const sepIdx = clean.search(/[:.]/);
  if (sepIdx === -1) {
    return { domain: clean, action: '*' };
  }
  const domain = clean.slice(0, sepIdx);
  const action = clean.slice(sepIdx + 1) || '*';
  return { domain, action };
}

function matchesSinglePermission(grantedPerms: Set<string>, requiredPerm: string): boolean {
  const req = parsePerm(requiredPerm);
  if (!req) return false;

  if (grantedPerms.has('*') || grantedPerms.has('*:*') || grantedPerms.has('*.*')) {
    return true;
  }

  for (const granted of grantedPerms) {
    if (granted === requiredPerm.trim().toLowerCase()) return true;
    const g = parsePerm(granted);
    if (!g) continue;

    const domainMatches = g.domain === '*' || g.domain === req.domain;
    if (!domainMatches) continue;

    if (g.action === '*' || g.action === req.action) {
      return true;
    }

    const acceptedByRequired = WRITE_ACTION_ALIASES[req.action];
    if (acceptedByRequired && acceptedByRequired.includes(g.action)) {
      return true;
    }
  }

  return false;
}

/**
 * Checks whether a user possesses the required permission (or any permission in an array),
 * seamlessly matching both dot-separated (gl.read / gl.*) and colon-separated (ap:read) claims.
 */
export function hasUserPermission(
  user: Pick<User, 'roles' | 'permissions'> | null | undefined,
  required: string | string[]
): boolean {
  if (!user) return false;
  if (Array.isArray(user.roles) && user.roles.some((r) => ADMIN_ROLES.has(r))) {
    return true;
  }

  const granted = new Set((user.permissions ?? []).map((p) => p.trim().toLowerCase()));
  if (granted.size === 0) return false;

  const list = Array.isArray(required) ? required : [required];
  if (list.length === 0) return true;
  return list.some((req) => matchesSinglePermission(granted, req));
}

export type PortalNavGroup =
  | 'overview'
  | 'finance'
  | 'supply_chain'
  | 'human_capital'
  | 'governance'
  | 'platform'
  | 'bi';

export interface PortalNavItem {
  id: string;
  serviceId: string;
  to: string;
  portal: PortalMode;
  group: PortalNavGroup;
  labelKey: string;
  defaultLabel: string;
  permissions: string[];
  keywords: string[];
}

export const PORTAL_MODULES: PortalNavItem[] = [
  // Backoffice Core & Finance
  {
    id: 'dashboard',
    serviceId: 'GW-000',
    to: '/',
    portal: 'backoffice',
    group: 'overview',
    labelKey: 'nav.dashboard',
    defaultLabel: 'Dashboard Eksekutif',
    permissions: [],
    keywords: ['dashboard', 'overview', 'kpi', 'ringkasan'],
  },
  {
    id: 'journals',
    serviceId: 'SVC-001',
    to: '/journals',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.journals',
    defaultLabel: 'Buku Besar & Jurnal',
    permissions: ['gl:read', 'gl.read', 'gl.*'],
    keywords: ['gl', 'journal', 'ledger', 'jurnal', 'buku besar'],
  },
  {
    id: 'coa',
    serviceId: 'SVC-002',
    to: '/coa',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.coa',
    defaultLabel: 'Bagan Akun & Periode',
    permissions: ['coa:read', 'coa.read'],
    keywords: ['coa', 'accounts', 'fiscal', 'periode', 'akun'],
  },
  {
    id: 'ap',
    serviceId: 'SVC-003',
    to: '/ap',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.ap',
    defaultLabel: 'Hutang Usaha (AP)',
    permissions: ['ap:read', 'ap.read'],
    keywords: ['ap', 'payable', 'vendor', 'hutang', 'tagihan'],
  },
  {
    id: 'ar',
    serviceId: 'SVC-004',
    to: '/ar',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.ar',
    defaultLabel: 'Piutang Usaha (AR)',
    permissions: ['ar:read', 'ar.read'],
    keywords: ['ar', 'receivable', 'customer', 'piutang', 'faktur'],
  },
  {
    id: 'tax',
    serviceId: 'SVC-005',
    to: '/tax',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.tax',
    defaultLabel: 'Pajak & e-Faktur',
    permissions: ['tax:read', 'tax.read'],
    keywords: ['tax', 'ppn', 'pph', 'efaktur', 'pajak'],
  },
  {
    id: 'treasury',
    serviceId: 'SVC-006',
    to: '/treasury',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.treasury',
    defaultLabel: 'Kas, Bank & Valas',
    permissions: ['treasury:read', 'currency:read', 'cashbank:read'],
    keywords: ['treasury', 'cashbank', 'bank', 'kas', 'reconciliation'],
  },
  {
    id: 'currency',
    serviceId: 'SVC-007',
    to: '/currency',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.currency',
    defaultLabel: 'Mata Uang & Kurs FX',
    permissions: ['currency:read', 'currency.read'],
    keywords: ['currency', 'fx', 'exchange rate', 'kurs', 'valas'],
  },
  {
    id: 'fixedAssets',
    serviceId: 'SVC-021',
    to: '/fixed-assets',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.fixedAssets',
    defaultLabel: 'Aset Tetap (Fixed Assets)',
    permissions: ['fixedasset:read', 'fixedasset.read'],
    keywords: ['fixed asset', 'depreciation', 'penyusutan', 'kapitalisasi', 'aset tetap'],
  },
  {
    id: 'dimensions',
    serviceId: 'SVC-022',
    to: '/dimensions',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.dimensions',
    defaultLabel: 'Dimensi & Pusat Biaya',
    permissions: ['dimension:read', 'dimension.read'],
    keywords: ['dimension', 'cost center', 'profit center', 'allocation', 'pusat biaya'],
  },
  {
    id: 'budgets',
    serviceId: 'SVC-023',
    to: '/budgets',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.budgets',
    defaultLabel: 'Anggaran & Kontrol Budget',
    permissions: ['budget:read', 'budget.read'],
    keywords: ['budget', 'anggaran', 'commitment', 'policy', 'pagu'],
  },
  {
    id: 'financialClose',
    serviceId: 'SVC-024',
    to: '/financial-close',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.financialClose',
    defaultLabel: 'Tutup Buku (Financial Close)',
    permissions: ['financialclose:read', 'financialclose.read'],
    keywords: ['financial close', 'tutup buku', 'closing', 'revaluation', 'period lock'],
  },
  {
    id: 'consolidation',
    serviceId: 'SVC-025',
    to: '/consolidation',
    portal: 'backoffice',
    group: 'finance',
    labelKey: 'nav.consolidation',
    defaultLabel: 'Konsolidasi Grup',
    permissions: ['consolidation:read', 'consolidation.read'],
    keywords: ['consolidation', 'konsolidasi', 'elimination', 'holding', 'group'],
  },
  // Backoffice Supply Chain & Manufacturing
  {
    id: 'procurement',
    serviceId: 'SVC-009',
    to: '/procurement',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.procurement',
    defaultLabel: 'Pengadaan (PR/PO)',
    permissions: ['procurement:read', 'procurement.read'],
    keywords: ['procurement', 'po', 'pr', 'grn', 'pembelian'],
  },
  {
    id: 'inventory',
    serviceId: 'SVC-008',
    to: '/inventory',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.inventory',
    defaultLabel: 'Persediaan & Gudang',
    permissions: ['inventory:read', 'inventory.read'],
    keywords: ['inventory', 'stock', 'warehouse', 'gudang', 'persediaan'],
  },
  {
    id: 'sales',
    serviceId: 'SVC-011',
    to: '/sales',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.sales',
    defaultLabel: 'Penjualan (SO)',
    permissions: ['sales:read', 'sales.read'],
    keywords: ['sales', 'so', 'order', 'penjualan'],
  },
  {
    id: 'manufacturing',
    serviceId: 'SVC-010',
    to: '/manufacturing',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.manufacturing',
    defaultLabel: 'Manufaktur & BOM',
    permissions: ['manufacturing:read', 'manufacturing.read'],
    keywords: ['manufacturing', 'bom', 'work order', 'produksi'],
  },
  {
    id: 'projects',
    serviceId: 'SVC-026',
    to: '/projects',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.projects',
    defaultLabel: 'Manajemen Proyek & WBS',
    permissions: ['project:read', 'project.read'],
    keywords: ['project', 'wbs', 'proyek', 'billing', 'change order'],
  },
  {
    id: 'assets',
    serviceId: 'SVC-027',
    to: '/assets',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.assets',
    defaultLabel: 'Aset Operasional & Lokasi',
    permissions: ['asset:read', 'asset.read'],
    keywords: ['asset', 'movement', 'verification', 'barcode', 'aset operasional'],
  },
  {
    id: 'maintenance',
    serviceId: 'SVC-028',
    to: '/maintenance',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.maintenance',
    defaultLabel: 'Pemeliharaan (CMMS)',
    permissions: ['maintenance:read', 'maintenance.read'],
    keywords: ['maintenance', 'cmms', 'work order', 'preventive', 'pemeliharaan'],
  },
  {
    id: 'crm',
    serviceId: 'SVC-029',
    to: '/crm',
    portal: 'backoffice',
    group: 'supply_chain',
    labelKey: 'nav.crm',
    defaultLabel: 'CRM & Pipeline Pelanggan',
    permissions: ['crm:read', 'crm.read'],
    keywords: ['crm', 'lead', 'opportunity', 'pipeline', 'support case', 'segment'],
  },
  // Backoffice People
  {
    id: 'hr',
    serviceId: 'SVC-012',
    to: '/hr',
    portal: 'backoffice',
    group: 'human_capital',
    labelKey: 'nav.hr',
    defaultLabel: 'Sumber Daya Manusia',
    permissions: ['hr:read', 'hr.read'],
    keywords: ['hr', 'employee', 'sdm', 'karyawan'],
  },
  {
    id: 'payroll',
    serviceId: 'SVC-013',
    to: '/payroll',
    portal: 'backoffice',
    group: 'human_capital',
    labelKey: 'nav.payroll',
    defaultLabel: 'Penggajian (Payroll)',
    permissions: ['payroll:read', 'payroll.read'],
    keywords: ['payroll', 'salary', 'gaji', 'penggajian'],
  },
  // Governance & Master Data
  {
    id: 'workflow',
    serviceId: 'SVC-015',
    to: '/workflow',
    portal: 'backoffice',
    group: 'governance',
    labelKey: 'nav.workflow',
    defaultLabel: 'Otorisasi & Persetujuan',
    permissions: ['workflow:read', 'workflow.read'],
    keywords: ['workflow', 'approval', 'persetujuan', 'otorisasi'],
  },
  {
    id: 'masterdata',
    serviceId: 'SVC-014',
    to: '/masterdata',
    portal: 'backoffice',
    group: 'governance',
    labelKey: 'nav.masterdata',
    defaultLabel: 'Master Partner & UoM',
    permissions: ['masterdata:read', 'masterdata.read'],
    keywords: ['masterdata', 'vendor', 'customer', 'uom', 'partner'],
  },
  // Platform Admin Portal
  {
    id: 'organizations',
    serviceId: 'SVC-016',
    to: '/organizations',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.organizations',
    defaultLabel: 'Struktur Organisasi',
    permissions: ['org:read', 'org.read'],
    keywords: ['organization', 'company', 'branch', 'holding', 'organisasi'],
  },
  {
    id: 'users',
    serviceId: 'SVC-017',
    to: '/users',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.users',
    defaultLabel: 'Manajemen Pengguna',
    permissions: ['user:read', 'user.read'],
    keywords: ['user', 'account', 'pengguna', 'iam'],
  },
  {
    id: 'security',
    serviceId: 'SVC-018',
    to: '/security',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.security',
    defaultLabel: 'Peran & Hak Akses RBAC',
    permissions: ['security:read', 'security.read'],
    keywords: ['security', 'rbac', 'roles', 'permissions', 'keamanan'],
  },
  {
    id: 'audit',
    serviceId: 'SVC-019',
    to: '/audit',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.audit',
    defaultLabel: 'Jejak Audit Sistem',
    permissions: ['audit:read', 'audit.read'],
    keywords: ['audit', 'trail', 'compliance', 'jejak'],
  },
  {
    id: 'localization',
    serviceId: 'SVC-030',
    to: '/localization',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.localization',
    defaultLabel: 'Bahasa & Lokalisasi',
    permissions: ['localization:read', 'localization.read'],
    keywords: ['localization', 'language', 'translation', 'locale', 'bahasa'],
  },
  {
    id: 'settings',
    serviceId: 'GW-000',
    to: '/settings',
    portal: 'admin',
    group: 'platform',
    labelKey: 'nav.settings',
    defaultLabel: 'Pengaturan Sistem',
    permissions: [],
    keywords: ['settings', 'preferences', 'pengaturan', 'tema', 'bahasa'],
  },
  // BI Reporting Portal
  {
    id: 'reports',
    serviceId: 'SVC-020',
    to: '/reports',
    portal: 'reporting',
    group: 'bi',
    labelKey: 'nav.reports',
    defaultLabel: 'Laporan Keuangan & BI',
    permissions: ['reporting.read', 'reporting:read', 'gl:read'],
    keywords: ['reporting', 'trial balance', 'income statement', 'neraca', 'laporan'],
  },
];

export function filterNavByPortalAndPermission(
  portal: PortalMode,
  user: Pick<User, 'roles' | 'permissions'> | null | undefined
): PortalNavItem[] {
  return PORTAL_MODULES.filter((item) => {
    if (item.portal !== portal) return false;
    if (item.permissions.length === 0) return true;
    return hasUserPermission(user, item.permissions);
  });
}
