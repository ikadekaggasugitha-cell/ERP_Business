import { createFileRoute, Link } from '@tanstack/react-router';
import { ArrowLeft, FileText } from 'lucide-react';
import { useUiStore } from '../../store/ui-store';

export const Route = createFileRoute('/_public/terms')({
  component: TermsOfServicePage,
});

function TermsOfServicePage() {
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
            <Link to="/privacy" className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-slate-200">
              {isId ? 'Kebijakan Privasi' : 'Privacy Policy'}
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
          <div className="inline-flex items-center gap-2 rounded-full border border-brand-200 bg-brand-50 px-3 py-1 text-xs font-semibold text-brand-800 dark:border-brand-900 dark:bg-brand-950/50 dark:text-brand-300">
            <FileText className="h-3.5 w-3.5" />
            <span>{isId ? 'Ketentuan Penggunaan Internal & Korporasi' : 'Enterprise Terms of Service'}</span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight">
            {isId ? 'Syarat & Ketentuan Penggunaan Platform ERP' : 'ERP Platform Terms of Service'}
          </h1>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            {isId
              ? 'Pembaruan terakhir: 24 November 2026 · PT Sinergi Investasi Properti Tbk'
              : 'Last updated: November 24, 2026 · PT Sinergi Investasi Properti Tbk'}
          </p>
        </div>

        <section className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6 sm:p-8 space-y-6 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              1. {isId ? 'Akses Terotorisasi & Pemisahan Tugas (SoD)' : 'Authorized Access & Segregation of Duties (SoD)'}
            </h2>
            <p>
              {isId
                ? 'Akses ke platform ini terbatas bagi personel PT Sinergi Investasi Properti Tbk dan entitas anak yang telah ditetapkan melalui kebijakan Role-Based Access Control (RBAC). Pengguna dilarang membagikan kredensial, memintas kontrol persetujuan berjenjang (SoD), atau mengakses data entitas di luar cakupan otorisasinya.'
                : 'Access is restricted to authorized personnel of PT Sinergi Investasi Properti Tbk and its subsidiaries governed by Role-Based Access Control (RBAC). Users must not share credentials, bypass Segregation of Duties (SoD) approval controls, or access entity data outside their assigned scope.'}
            </p>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              2. {isId ? 'Integritas Data Keuangan & Audit' : 'Financial Data Integrity & Auditability'}
            </h2>
            <p>
              {isId
                ? 'Seluruh transaksi jurnal, faktur, pembayaran, penggajian, tutup buku, dan konsolidasi dicatat secara permanen pada Audit Trail. Pengguna bertanggung jawab atas kebenaran data sumber yang dimasukkan serta kepatuhan terhadap PSAK/IFRS dan ketentuan perpajakan yang berlaku.'
                : 'All journal postings, invoices, payments, payroll runs, period closes, and consolidations are permanently recorded in the Audit Trail. Users are responsible for the accuracy of source records and compliance with PSAK/IFRS and applicable statutory regulations.'}
            </p>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              3. {isId ? 'Kepemilikan Data & Ketersediaan Layanan' : 'Data Ownership & Service Availability'}
            </h2>
            <p>
              {isId
                ? 'Seluruh catatan bisnis, laporan keuangan, dan konfigurasi yang tersimpan dalam sistem merupakan milik penuh PT Sinergi Investasi Properti Tbk. Pemeliharaan terjadwal dan pencadangan basis data dilakukan sesuai kebijakan operasional TI perusahaan.'
                : 'All business records, financial statements, and configurations stored in the platform remain the sole property of PT Sinergi Investasi Properti Tbk. Scheduled maintenance and backups are executed in accordance with enterprise IT operations policy.'}
            </p>
          </div>

          <div className="space-y-2">
            <h2 className="text-base font-bold text-slate-900 dark:text-slate-100">
              4. {isId ? 'Pengakhiran Akses & Penghapusan Akun' : 'Access Termination & Account Deletion'}
            </h2>
            <p>
              {isId
                ? 'Hak akses dapat dinonaktifkan seketika apabila terjadi pelanggaran keamanan atau berakhirnya hubungan kerja. Permintaan penghapusan akun dan data pribadi dapat diajukan melalui '
                : 'Access may be suspended immediately upon security violations or termination of employment. Account and personal data deletion requests may be submitted via '}
              <Link to="/delete-account" className="font-semibold text-brand-700 underline dark:text-brand-400">
                /delete-account
              </Link>
              .
            </p>
          </div>
        </section>
      </main>
    </div>
  );
}
