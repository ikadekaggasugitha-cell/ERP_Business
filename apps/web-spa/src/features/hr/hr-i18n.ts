import { useUiStore } from '../../store/ui-store';

export interface HrText {
  title: string;
  subtitle: string;
  readOnlyHint: string;
  tabEmployees: string;
  tabPositions: string;
  kpiActiveEmployees: string;
  kpiTotalEmployees: string;
  kpiTerminated: string;
  kpiPositions: string;
  addEmployee: string;
  editEmployee: string;
  addPosition: string;
  searchPlaceholder: string;
  nationalIdFilterPlaceholder: string;
  statusAll: string;
  statusActive: string;
  statusTerminated: string;
  colEmployeeNo: string;
  colFullName: string;
  colNationalId: string;
  colHireDate: string;
  colEmail: string;
  colStatus: string;
  colActions: string;
  colPositionCode: string;
  colPositionTitle: string;
  colParentPosition: string;
  colOrgRef: string;
  viewDetails: string;
  terminate: string;
  terminateTitle: string;
  terminateDescription: string;
  terminateConfirm: string;
  cancel: string;
  save: string;
  create: string;
  piiMasked: string;
  piiRevealed: string;
  togglePiiShow: string;
  togglePiiHide: string;
  subTabEmployment: string;
  subTabCompensation: string;
  addEmployment: string;
  addCompensation: string;
  totalCompensationLabel: string;
  emptyEmployeesTitle: string;
  emptyEmployeesDesc: string;
  emptyPositionsTitle: string;
  emptyPositionsDesc: string;
  emptyEmployments: string;
  emptyCompensations: string;
  employeeCreated: string;
  employeeUpdated: string;
  positionCreated: string;
  employmentCreated: string;
  compensationCreated: string;
  exportCsv: string;
  resetFilters: string;
}

const EN_TEXT: HrText = {
  title: 'Human Resources & Organization Staffing',
  subtitle: 'Manage employee directory, PII-protected records, employment history, compensation grades, and positions.',
  readOnlyHint: 'Read-only mode: hr:write permission is required to create, update, or terminate records.',
  tabEmployees: 'Employees',
  tabPositions: 'Positions',
  kpiActiveEmployees: 'Active Headcount',
  kpiTotalEmployees: 'Directory Records',
  kpiTerminated: 'Terminated (Page)',
  kpiPositions: 'Defined Positions',
  addEmployee: 'New Employee',
  editEmployee: 'Edit Employee',
  addPosition: 'New Position',
  searchPlaceholder: 'Filter loaded rows by name or employee #...',
  nationalIdFilterPlaceholder: 'Exact National ID (NIK) lookup...',
  statusAll: 'All Statuses',
  statusActive: 'Active',
  statusTerminated: 'Terminated',
  colEmployeeNo: 'Employee #',
  colFullName: 'Full Name',
  colNationalId: 'National ID (NIK)',
  colHireDate: 'Hire Date',
  colEmail: 'Email',
  colStatus: 'Status',
  colActions: 'Actions',
  colPositionCode: 'Position Code',
  colPositionTitle: 'Title',
  colParentPosition: 'Reports To',
  colOrgRef: 'Org Unit Ref',
  viewDetails: 'Inspect',
  terminate: 'Terminate',
  terminateTitle: 'Terminate Employee',
  terminateDescription: 'Confirm termination for this employee. Optimistic concurrency (If-Match) is enforced.',
  terminateConfirm: 'Confirm Termination',
  cancel: 'Cancel',
  save: 'Save Changes',
  create: 'Create Record',
  piiMasked: 'PII Protected (Masked)',
  piiRevealed: 'PII Unmasked',
  togglePiiShow: 'Reveal PII',
  togglePiiHide: 'Mask PII',
  subTabEmployment: 'Employment History',
  subTabCompensation: 'Compensation Grades',
  addEmployment: 'Assign Position',
  addCompensation: 'Add Compensation Component',
  totalCompensationLabel: 'Fixed-Point Package Total',
  emptyEmployeesTitle: 'No employees found',
  emptyEmployeesDesc: 'Adjust status/NIK filters or create a new employee record.',
  emptyPositionsTitle: 'No positions defined',
  emptyPositionsDesc: 'Define structural job positions before assigning employee history.',
  emptyEmployments: 'No employment assignments recorded yet.',
  emptyCompensations: 'No compensation components recorded yet.',
  employeeCreated: 'Employee created successfully.',
  employeeUpdated: 'Employee updated successfully.',
  positionCreated: 'Position created successfully.',
  employmentCreated: 'Employment assignment added.',
  compensationCreated: 'Compensation grade added.',
  exportCsv: 'Export CSV',
  resetFilters: 'Reset Filters',
};

const ID_TEXT: HrText = {
  title: 'Sumber Daya Manusia & Kepegawaian',
  subtitle: 'Kelola direktori karyawan, perlindungan PII, riwayat penempatan, grade kompensasi, dan posisi jabatan.',
  readOnlyHint: 'Mode baca-saja: izin hr:write diperlukan untuk membuat, mengubah, atau memberhentikan karyawan.',
  tabEmployees: 'Karyawan',
  tabPositions: 'Posisi Jabatan',
  kpiActiveEmployees: 'Karyawan Aktif',
  kpiTotalEmployees: 'Total Direktori',
  kpiTerminated: 'Diberhentikan (Halaman)',
  kpiPositions: 'Posisi Terdaftar',
  addEmployee: 'Tambah Karyawan',
  editEmployee: 'Ubah Karyawan',
  addPosition: 'Tambah Posisi',
  searchPlaceholder: 'Saring baris termuat berdasarkan nama atau NIP...',
  nationalIdFilterPlaceholder: 'Cari NIK persis...',
  statusAll: 'Semua Status',
  statusActive: 'Aktif',
  statusTerminated: 'Berhenti',
  colEmployeeNo: 'No. Karyawan',
  colFullName: 'Nama Lengkap',
  colNationalId: 'NIK (Tersamarkan)',
  colHireDate: 'Tanggal Masuk',
  colEmail: 'Email',
  colStatus: 'Status',
  colActions: 'Aksi',
  colPositionCode: 'Kode Posisi',
  colPositionTitle: 'Nama Jabatan',
  colParentPosition: 'Atasan Langsung',
  colOrgRef: 'Ref Unit Org',
  viewDetails: 'Detail',
  terminate: 'Berhentikan',
  terminateTitle: 'Berhentikan Karyawan',
  terminateDescription: 'Konfirmasi pemberhentian karyawan ini. Versi data (If-Match) divalidasi oleh server.',
  terminateConfirm: 'Konfirmasi Berhenti',
  cancel: 'Batal',
  save: 'Simpan Perubahan',
  create: 'Buat Data',
  piiMasked: 'PII Dilindungi (Tersamarkan)',
  piiRevealed: 'PII Terbuka',
  togglePiiShow: 'Tampilkan PII',
  togglePiiHide: 'Samarkan PII',
  subTabEmployment: 'Riwayat Penempatan',
  subTabCompensation: 'Grade Kompensasi',
  addEmployment: 'Tambah Penempatan',
  addCompensation: 'Tambah Komponen Kompensasi',
  totalCompensationLabel: 'Total Paket (Fixed-Point)',
  emptyEmployeesTitle: 'Data karyawan tidak ditemukan',
  emptyEmployeesDesc: 'Ubah filter status/NIK atau tambahkan data karyawan baru.',
  emptyPositionsTitle: 'Belum ada posisi jabatan',
  emptyPositionsDesc: 'Buat daftar posisi jabatan terlebih dahulu sebelum penempatan karyawan.',
  emptyEmployments: 'Belum ada riwayat penempatan jabatan.',
  emptyCompensations: 'Belum ada komponen kompensasi.',
  employeeCreated: 'Karyawan berhasil ditambahkan.',
  employeeUpdated: 'Data karyawan berhasil diperbarui.',
  positionCreated: 'Posisi jabatan berhasil dibuat.',
  employmentCreated: 'Riwayat penempatan berhasil ditambahkan.',
  compensationCreated: 'Komponen kompensasi berhasil ditambahkan.',
  exportCsv: 'Ekspor CSV',
  resetFilters: 'Reset Filter',
};

export function useHrText(): HrText {
  const locale = useUiStore((s) => s.locale);
  return locale === 'id' ? ID_TEXT : EN_TEXT;
}

export function maskEmailValue(email: string | null | undefined, masked: boolean): string {
  if (!email) return '—';
  if (!masked) return email;
  const atIndex = email.indexOf('@');
  if (atIndex <= 1) return '***';
  return `${email.slice(0, 2)}***${email.slice(atIndex)}`;
}

export function maskIsoDateValue(dateVal: string | null | undefined, masked: boolean): string {
  if (!dateVal) return '—';
  if (!masked) return dateVal;
  return '••••-••-••';
}

export function maskNationalIdValue(val: string | null | undefined, masked: boolean): string {
  if (!val) return '—';
  if (!masked) return val;
  if (val.includes('*')) return val;
  if (val.length <= 4) return '••••';
  return `${'•'.repeat(Math.max(4, val.length - 4))}${val.slice(-4)}`;
}
