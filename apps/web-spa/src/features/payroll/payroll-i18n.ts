import { useUiStore } from '../../store/ui-store';

export interface PayrollText {
  title: string;
  subtitle: string;
  readOnlyHint: string;
  newRun: string;
  searchPlaceholder: string;
  statusAll: string;
  kpiTotalRuns: string;
  kpiDraft: string;
  kpiCalculatedApproved: string;
  kpiPosted: string;
  colRunNo: string;
  colPeriod: string;
  colStatus: string;
  colGlRef: string;
  colVersion: string;
  colActions: string;
  inspect: string;
  calculate: string;
  approve: string;
  postToGl: string;
  deleteDraft: string;
  cancel: string;
  create: string;
  confirmDeleteTitle: string;
  confirmDeleteDesc: string;
  totalGross: string;
  totalDeductions: string;
  totalTax: string;
  totalNet: string;
  emptyTitle: string;
  emptyDesc: string;
  runCreated: string;
  runCalculated: string;
  runApproved: string;
  runPosted: string;
  runDeleted: string;
}

const EN_TEXT: PayrollText = {
  title: 'Payroll Runs & Statutory Calculation',
  subtitle: 'Create payroll batches, compute PPh 21 / BPJS statutory contributions, approve, and post to General Ledger.',
  readOnlyHint: 'Read-only mode: payroll:write permission is required to create, calculate, approve, post, or delete runs.',
  newRun: 'New Payroll Run',
  searchPlaceholder: 'Search loaded runs by run # or GL ref...',
  statusAll: 'All Statuses',
  kpiTotalRuns: 'Total Runs',
  kpiDraft: 'Draft Batches',
  kpiCalculatedApproved: 'Ready / Approved',
  kpiPosted: 'Posted to GL',
  colRunNo: 'Run #',
  colPeriod: 'Period',
  colStatus: 'Status',
  colGlRef: 'GL Journal Ref',
  colVersion: 'Version',
  colActions: 'Actions',
  inspect: 'Manage Run',
  calculate: 'Calculate Payroll',
  approve: 'Approve Run',
  postToGl: 'Post to GL',
  deleteDraft: 'Delete Draft',
  cancel: 'Cancel',
  create: 'Create Run',
  confirmDeleteTitle: 'Delete Draft Payroll Run',
  confirmDeleteDesc: 'Only Draft runs can be deleted. Optimistic concurrency (If-Match) is sent.',
  totalGross: 'Total Gross',
  totalDeductions: 'Total Deductions',
  totalTax: 'Total PPh 21 Tax',
  totalNet: 'Total Net Pay',
  emptyTitle: 'No payroll runs found',
  emptyDesc: 'Create a new payroll run for the active entity or clear status filters.',
  runCreated: 'Payroll run created.',
  runCalculated: 'Payroll calculation completed.',
  runApproved: 'Payroll run approved.',
  runPosted: 'Payroll posted to General Ledger.',
  runDeleted: 'Draft payroll run deleted.',
};

const ID_TEXT: PayrollText = {
  title: 'Penggajian (Payroll) & Kalkulasi Pajak',
  subtitle: 'Buat periode penggajian, hitung PPh 21 / iuran BPJS, setujui batch, dan posting ke Buku Besar.',
  readOnlyHint: 'Mode baca-saja: izin payroll:write diperlukan untuk membuat, menghitung, menyetujui, memposting, atau menghapus batch.',
  newRun: 'Buat Payroll Run',
  searchPlaceholder: 'Cari nomor run atau referensi jurnal GL...',
  statusAll: 'Semua Status',
  kpiTotalRuns: 'Total Payroll Run',
  kpiDraft: 'Batch Draf',
  kpiCalculatedApproved: 'Terhitung / Disetujui',
  kpiPosted: 'Diposting ke GL',
  colRunNo: 'No. Run',
  colPeriod: 'Periode',
  colStatus: 'Status',
  colGlRef: 'Ref Jurnal GL',
  colVersion: 'Versi',
  colActions: 'Aksi',
  inspect: 'Kelola Run',
  calculate: 'Hitung Payroll',
  approve: 'Setujui Run',
  postToGl: 'Posting ke GL',
  deleteDraft: 'Hapus Draf',
  cancel: 'Batal',
  create: 'Buat Run',
  confirmDeleteTitle: 'Hapus Draf Payroll Run',
  confirmDeleteDesc: 'Hanya batch berstatus Draft yang dapat dihapus. Versi data (If-Match) divalidasi.',
  totalGross: 'Total Bruto',
  totalDeductions: 'Total Potongan',
  totalTax: 'Total Pajak PPh 21',
  totalNet: 'Total Take-Home Pay',
  emptyTitle: 'Belum ada payroll run',
  emptyDesc: 'Buat payroll run baru untuk entitas aktif atau ubah filter status.',
  runCreated: 'Payroll run berhasil dibuat.',
  runCalculated: 'Kalkulasi payroll selesai.',
  runApproved: 'Payroll run disetujui.',
  runPosted: 'Payroll berhasil diposting ke GL.',
  runDeleted: 'Draf payroll run dihapus.',
};

export function usePayrollText(): PayrollText {
  const locale = useUiStore((s) => s.locale);
  return locale === 'id' ? ID_TEXT : EN_TEXT;
}

export function formatPayrollPeriod(month: number, year: number): string {
  return `${year}-${String(month).padStart(2, '0')}`;
}
