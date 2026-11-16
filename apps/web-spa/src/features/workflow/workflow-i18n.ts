import { useUiStore } from '../../store/ui-store';

export interface WorkflowText {
  title: string;
  subtitle: string;
  readOnlyHint: string;
  tabTasks: string;
  tabInstances: string;
  tabDefinitions: string;
  kpiPendingTasks: string;
  kpiInstances: string;
  kpiDefinitions: string;
  kpiSodGuarded: string;
  filterPending: string;
  filterAll: string;
  filterDecided: string;
  approve: string;
  reject: string;
  approveTitle: string;
  rejectTitle: string;
  decisionCommentLabel: string;
  createDefinition: string;
  createInstance: string;
  inspectTimeline: string;
  sodBadge: string;
  sodBlockedHint: string;
  cancel: string;
  confirm: string;
  emptyTasksTitle: string;
  emptyTasksDesc: string;
  emptyInstancesTitle: string;
  emptyInstancesDesc: string;
  emptyDefinitionsTitle: string;
  emptyDefinitionsDesc: string;
  approvedToast: string;
  rejectedToast: string;
  definitionCreatedToast: string;
  instanceCreatedToast: string;
}

const EN_TEXT: WorkflowText = {
  title: 'Workflow Approvals & Governance Inbox',
  subtitle: 'Review pending approval tasks with Segregation-of-Duties (SoD) enforcement, inspect instance timelines, and define rules.',
  readOnlyHint: 'Limited permissions: workflow:approve is needed to decide tasks; workflow:write is needed to define workflows.',
  tabTasks: 'Approval Tasks',
  tabInstances: 'Workflow Instances',
  tabDefinitions: 'Workflow Definitions',
  kpiPendingTasks: 'Pending Tasks',
  kpiInstances: 'Workflow Instances',
  kpiDefinitions: 'Active Definitions',
  kpiSodGuarded: 'Self-Initiated (SoD)',
  filterPending: 'Pending Only',
  filterAll: 'All Tasks',
  filterDecided: 'Decided Only',
  approve: 'Approve',
  reject: 'Reject',
  approveTitle: 'Approve Workflow Task',
  rejectTitle: 'Reject Workflow Task',
  decisionCommentLabel: 'Approval / Rejection Note',
  createDefinition: 'New Definition',
  createInstance: 'Start Instance',
  inspectTimeline: 'Timeline',
  sodBadge: 'SoD: Self-Created',
  sodBlockedHint: 'Segregation of Duties prohibits the creator from approving or rejecting their own submission.',
  cancel: 'Cancel',
  confirm: 'Confirm Decision',
  emptyTasksTitle: 'No approval tasks in queue',
  emptyTasksDesc: 'Change the status filter or wait for new submissions.',
  emptyInstancesTitle: 'No workflow instances found',
  emptyInstancesDesc: 'Start a workflow instance against an active definition.',
  emptyDefinitionsTitle: 'No workflow definitions registered',
  emptyDefinitionsDesc: 'Create a workflow definition to configure multi-step approval policies.',
  approvedToast: 'Task approved.',
  rejectedToast: 'Task rejected.',
  definitionCreatedToast: 'Workflow definition created.',
  instanceCreatedToast: 'Workflow instance started.',
};

const ID_TEXT: WorkflowText = {
  title: 'Kotak Persetujuan & Tata Kelola Workflow',
  subtitle: 'Tinjau tugas persetujuan dengan penegakan Segregation of Duties (SoD), linimasa instans, dan definisi alur.',
  readOnlyHint: 'Akses terbatas: workflow:approve diperlukan untuk memutuskan tugas; workflow:write diperlukan untuk membuat definisi.',
  tabTasks: 'Tugas Persetujuan',
  tabInstances: 'Instans Workflow',
  tabDefinitions: 'Definisi Workflow',
  kpiPendingTasks: 'Menunggu Persetujuan',
  kpiInstances: 'Instans Workflow',
  kpiDefinitions: 'Definisi Aktif',
  kpiSodGuarded: 'Dibuat Sendiri (SoD)',
  filterPending: 'Hanya Menunggu',
  filterAll: 'Semua Tugas',
  filterDecided: 'Sudah Diputuskan',
  approve: 'Setujui',
  reject: 'Tolak',
  approveTitle: 'Setujui Tugas Workflow',
  rejectTitle: 'Tolak Tugas Workflow',
  decisionCommentLabel: 'Catatan Keputusan',
  createDefinition: 'Buat Definisi',
  createInstance: 'Mulai Instans',
  inspectTimeline: 'Linimasa',
  sodBadge: 'SoD: Pengaju Sendiri',
  sodBlockedHint: 'Pemisahan Tugas (SoD) melarang pembuat menyetujui atau menolak pengajuannya sendiri.',
  cancel: 'Batal',
  confirm: 'Konfirmasi Keputusan',
  emptyTasksTitle: 'Tidak ada tugas persetujuan',
  emptyTasksDesc: 'Ubah filter status atau tunggu pengajuan baru.',
  emptyInstancesTitle: 'Belum ada instans workflow',
  emptyInstancesDesc: 'Mulai instans workflow dari definisi yang aktif.',
  emptyDefinitionsTitle: 'Belum ada definisi workflow',
  emptyDefinitionsDesc: 'Buat definisi workflow untuk mengatur kebijakan persetujuan berjenjang.',
  approvedToast: 'Tugas berhasil disetujui.',
  rejectedToast: 'Tugas berhasil ditolak.',
  definitionCreatedToast: 'Definisi workflow dibuat.',
  instanceCreatedToast: 'Instans workflow dimulai.',
};

export function useWorkflowText(): WorkflowText {
  const locale = useUiStore((s) => s.locale);
  return locale === 'id' ? ID_TEXT : EN_TEXT;
}
