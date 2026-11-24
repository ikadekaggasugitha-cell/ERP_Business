import { createFileRoute, Link } from '@tanstack/react-router';
import { ArrowLeft, Lock, ShieldCheck } from 'lucide-react';
import { useUiStore } from '../../store/ui-store';

export const Route = createFileRoute('/_public/privacy')({
  component: PrivacyPolicyPage,
});

function PrivacyPolicyPage() {
  const { locale, setLocale } = useUiStore();
  const isId = locale === 'id';

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100">
      <header className="sticky top-0 z-20 border-b border-slate-200 dark:border-slate-800 bg-white/90 dark:bg-slate-900/90 backdrop-blur-md">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-4 py-3.5 sm:px-6">
          <Link
            to="/login"
            className="inline-flex items-center gap-2 text-xs font-semibold text-brand-700 hover:text-brand-800 dark:text-brand-400"
          >
            <ArrowLeft className="h-4 w-4" />
            <span>{isId ? 'Kembali ke Halaman Masuk' : 'Back to Sign In'}</span>
          </Link>
          <div className="flex items-center gap-3">
            <Link to="/terms" className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-slate-200">
              {isId ? 'Syarat & Ketentuan' : 'Terms'}
            </Link>
            <Link to="/delete-account" className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-slate-200">
              {isId ? 'Hapus Akun' : 'Delete Account'}
            </Link>
            <button
              type="button"
              onClick={() => setLocale(isId ? 'en' : 'id')}
              className="rounded-lg border border-slate-200 dark:border-slate-700 px-2.5 py-1 text-xs font-mono font-semibold cursor-pointer"
            >
              {locale.toUpperCase()}
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-10 sm:px-6 space-y-8">
        <div className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6 sm:p-8 space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-semibold text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/50 dark:text-emerald-300">
            <ShieldCheck className="h-3.5 w-3.5" />
            <span>UU PDP No. 27/2022 · GDPR Art. 13–17</span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight">
            {isId ? 'Kebijakan Privasi Platform ERP Enterprise' : 'Enterprise ERP Platform Privacy Policy'}
          </h1>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            {isId
              ? 'Berlaku efektif: 24 November 2026 · PT Sinergi Investasi Properti Tbk'
              : 'Effective date: November 24, 2026 · PT Sinergi Investasi Properti Tbk'}
          </p>
        </div>

        <section className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6 sm:p-8 space-y-6 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              1. {isId ? 'Ruang Lingkup & Pengendali Data' : 'Scope & Data Controller'}
            </h2>
            <p>
              {isId
                ? 'Kebijakan ini menjelaskan bagaimana PT Sinergi Investasi Properti Tbk mengumpulkan, memproses, menyimpan, dan melindungi data pribadi pengguna internal, karyawan, vendor, serta pelanggan pada aplikasi Web Backoffice dan ekstensi Chrome Enterprise Companion.'
                : 'This policy explains how PT Sinergi Investasi Properti Tbk collects, processes, stores, and protects personal data of internal users, employees, vendors, and customers across the Web Backoffice SPA and the Enterprise Chrome Companion extension.'}
            </p>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              2. {isId ? 'Kategori Data yang Diproses' : 'Categories of Personal Data Processed'}
            </h2>
            <ul className="list-disc pl-5 space-y-1">
              <li>{isId ? 'Identitas akun & otorisasi: nama lengkap, alamat email korporasi, nama pengguna, peran RBAC, dan cakupan organisasi (RLS).' : 'Account & authorization identity: full name, corporate email, username, RBAC roles, and organization scope (RLS).'}</li>
              <li>{isId ? 'Data SDM & Penggajian terenkripsi: NIK/NPWP, data kepegawaian, komponen gaji, BPJS, dan PPh 21 (dienkripsi AES-256-GCM pada lapisan aplikasi).' : 'Encrypted HR & Payroll records: national ID/tax ID, employment history, compensation, BPJS, and PPh 21 (AES-256-GCM application-level encryption).'}</li>
              <li>{isId ? 'Jejak audit & keamanan: correlation ID, stempel waktu aktivitas, perubahan sebelum/sesudah dokumen keuangan, dan identitas pelaku persetujuan (SoD).' : 'Audit & security trails: correlation IDs, activity timestamps, before/after financial document diffs, and approval actor identities (SoD).'}</li>
            </ul>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              3. {isId ? 'Keamanan, Isolasi Tenant & Retensi' : 'Security, Tenant Isolation & Retention'}
            </h2>
            <p>
              {isId
                ? 'Setiap tabel operasional dilindungi PostgreSQL Row-Level Security (RLS) berbasis hierarki organisasi. Sesi autentikasi menggunakan HttpOnly cookie dan token OIDC/JWT berumur pendek. Data transaksi akuntansi dan perpajakan disimpan minimal 10 tahun sesuai peraturan perpajakan Indonesia; data akun non-fiskal dihapus atau dianonimkan sesuai permintaan sah subjek data.'
                : 'All operational tables enforce PostgreSQL Row-Level Security (RLS) scoped to organization hierarchy. Authentication sessions use HttpOnly cookies and short-lived OIDC/JWT tokens. Accounting and statutory tax records are retained for 10 years as required by Indonesian tax law; non-statutory personal account data is erased or anonymized upon verified request.'}
            </p>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              4. {isId ? 'Hak Subjek Data (UU PDP & GDPR)' : 'Data Subject Rights (UU PDP & GDPR)'}
            </h2>
            <p>
              {isId
                ? 'Anda berhak mengakses, memperbaiki, membatasi pemrosesan, atau meminta penghapusan data pribadi Anda. Untuk penghapusan akun mandiri atau pengajuan permohonan penghapusan data, gunakan halaman '
                : 'You have the right to access, rectify, restrict processing of, or request erasure of your personal data. To initiate self-service account deletion or submit a formal erasure request, visit '}
              <Link to="/delete-account" className="font-semibold text-brand-700 underline dark:text-brand-400">
                /delete-account
              </Link>
              .
            </p>
          </div>

          <div className="flex items-center gap-2 rounded-xl border border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-800/50 p-4 text-xs">
            <Lock className="h-4 w-4 text-brand-600 dark:text-brand-400 shrink-0" />
            <span>
              {isId
                ? 'Petugas Pelindungan Data Pribadi (DPO): dpo@sinergi.example.com · PT Sinergi Investasi Properti Tbk.'
                : 'Data Protection Officer (DPO): dpo@sinergi.example.com · PT Sinergi Investasi Properti Tbk.'}
            </span>
          </div>
        </section>
      </main>
    </div>
  );
}
