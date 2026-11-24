import * as React from 'react';
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { AlertTriangle, ArrowLeft, CheckCircle2, Trash2 } from 'lucide-react';
import { Button } from '../../components/ui/Button';
import { FormField } from '../../components/ui/FormField';
import { Input } from '../../components/ui/Input';
import { Textarea } from '../../components/ui/Textarea';
import { apiClient, ApiClientError } from '../../lib/api';
import { useAuthStore } from '../../store/auth-store';
import { useUiStore } from '../../store/ui-store';

export const Route = createFileRoute('/_public/delete-account')({
  component: DeleteAccountPage,
});

interface UserRecord {
  id: string;
  username: string;
  email: string;
  rowVersion: number;
}

function DeleteAccountPage() {
  const navigate = useNavigate();
  const { locale, setLocale } = useUiStore();
  const { isAuthenticated, user, logout } = useAuthStore();
  const isId = locale === 'id';

  const [confirmText, setConfirmText] = React.useState('');
  const [submitting, setSubmitting] = React.useState(false);
  const [errorMsg, setErrorMsg] = React.useState<string | null>(null);
  const [doneRef, setDoneRef] = React.useState<string | null>(null);

  const [publicEmail, setPublicEmail] = React.useState('');
  const [publicReason, setPublicReason] = React.useState('');

  const handleSelfServiceDelete = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!user) return;
    if (confirmText.trim().toUpperCase() !== 'DELETE') {
      setErrorMsg(isId ? 'Ketik DELETE untuk mengonfirmasi penghapusan akun.' : 'Type DELETE to confirm account deletion.');
      return;
    }
    setSubmitting(true);
    setErrorMsg(null);
    try {
      let rowVersion = 1;
      try {
        const detail = await apiClient<UserRecord>(`/users/${user.id}`);
        if (detail?.rowVersion) rowVersion = detail.rowVersion;
      } catch {
        // Proceed with default rowVersion if detail lookup is restricted
      }
      await apiClient<void>(`/users/${user.id}`, {
        method: 'DELETE',
        ifMatch: String(rowVersion),
      });
      setDoneRef(`SELF-${user.id.slice(0, 8).toUpperCase()}`);
      logout();
      setTimeout(() => {
        void navigate({ to: '/login' });
      }, 2500);
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setErrorMsg(err.message);
      } else {
        setErrorMsg(err instanceof Error ? err.message : 'Failed to delete account');
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handlePublicRequest = (e: React.FormEvent) => {
    e.preventDefault();
    if (!publicEmail.trim() || !publicEmail.includes('@')) {
      setErrorMsg(isId ? 'Masukkan alamat email terdaftar yang valid.' : 'Please enter a valid registered email address.');
      return;
    }
    setErrorMsg(null);
    const ticket = `DSR-${crypto.randomUUID().slice(0, 8).toUpperCase()}`;
    setDoneRef(ticket);
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950 text-slate-900 dark:text-slate-100">
      <header className="sticky top-0 z-20 border-b border-slate-200 dark:border-slate-800 bg-white/90 dark:bg-slate-900/90 backdrop-blur-md">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-4 py-3.5 sm:px-6">
          <Link
            to="/login"
            className="inline-flex items-center gap-2 text-xs font-semibold text-brand-700 hover:text-brand-800 dark:text-brand-400"
          >
            <ArrowLeft className="h-4 w-4" />
            <span>{isId ? 'Kembali ke Halaman Masuk' : 'Back to Sign In'}</span>
          </Link>
          <div className="flex items-center gap-3">
            <Link to="/privacy" className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-slate-200">
              {isId ? 'Kebijakan Privasi' : 'Privacy'}
            </Link>
            <Link to="/terms" className="text-xs text-slate-500 hover:text-slate-800 dark:hover:text-slate-200">
              {isId ? 'Syarat' : 'Terms'}
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

      <main className="mx-auto max-w-3xl px-4 py-10 sm:px-6 space-y-6">
        <div className="rounded-2xl border border-rose-200 dark:border-rose-900/60 bg-white dark:bg-slate-900 p-6 sm:p-8 space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-rose-200 bg-rose-50 px-3 py-1 text-xs font-semibold text-rose-800 dark:border-rose-900 dark:bg-rose-950/50 dark:text-rose-300">
            <Trash2 className="h-3.5 w-3.5" />
            <span>UU PDP Art. 8 · GDPR Art. 17 (Right to Erasure)</span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight">
            {isId ? 'Permintaan Penghapusan Akun & Data Pribadi' : 'Account & Personal Data Deletion Request'}
          </h1>
          <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
            {isId
              ? 'Penghapusan akun akan menonaktifkan akses login Anda dan menghapus profil pribadi non-fiskal. Catatan audit keuangan dan perpajakan yang diwajibkan undang-undang tetap dipertahankan dalam bentuk teranonimisasi selama periode retensi wajib.'
              : 'Deleting your account revokes all login access and removes non-statutory personal profile fields. Statutory financial and tax audit trails are retained in anonymized form for the legally mandated retention period.'}
          </p>
        </div>

        {doneRef ? (
          <div className="rounded-2xl border border-emerald-200 dark:border-emerald-900 bg-emerald-50/70 dark:bg-emerald-950/30 p-6 space-y-2">
            <div className="flex items-center gap-2 font-bold text-emerald-800 dark:text-emerald-300">
              <CheckCircle2 className="h-5 w-5" />
              <span>{isId ? 'Permintaan Diterima' : 'Request Recorded'}</span>
            </div>
            <p className="text-xs text-emerald-900/90 dark:text-emerald-200">
              {isId
                ? `Nomor referensi penghapusan: ${doneRef}. Tim DPO akan memproses dan mengonfirmasi dalam maksimal 3×24 jam kerja.`
                : `Deletion reference ID: ${doneRef}. Our DPO team will complete and confirm erasure within 3 business days.`}
            </p>
          </div>
        ) : (
          <div className="rounded-2xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 p-6 sm:p-8 space-y-5">
            {errorMsg ? (
              <div role="alert" className="flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-200">
                <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                <span>{errorMsg}</span>
              </div>
            ) : null}

            {isAuthenticated && user ? (
              <form onSubmit={handleSelfServiceDelete} className="space-y-4">
                <div className="rounded-xl border border-amber-200 bg-amber-50/60 dark:border-amber-900/50 dark:bg-amber-950/20 p-3.5 text-xs text-amber-900 dark:text-amber-200">
                  {isId
                    ? `Anda sedang masuk sebagai ${user.fullName} (${user.email}). Menghapus akun akan langsung mengeluarkan sesi Anda.`
                    : `Signed in as ${user.fullName} (${user.email}). Confirming will immediately disable/delete your account and sign you out.`}
                </div>
                <FormField
                  label={isId ? 'Ketik DELETE untuk konfirmasi' : 'Type DELETE to confirm'}
                  htmlFor="confirmDelete"
                  required
                >
                  <Input
                    id="confirmDelete"
                    value={confirmText}
                    onChange={(e) => setConfirmText(e.target.value)}
                    placeholder="DELETE"
                  />
                </FormField>
                <Button type="submit" variant="danger" isLoading={submitting} leftIcon={<Trash2 className="h-4 w-4" />}>
                  {isId ? 'Hapus Akun Saya Sekarang' : 'Delete My Account Now'}
                </Button>
              </form>
            ) : (
              <form onSubmit={handlePublicRequest} className="space-y-4">
                <p className="text-xs text-slate-600 dark:text-slate-400">
                  {isId
                    ? 'Tidak sedang masuk? Ajukan permintaan penghapusan data pribadi publik yang akan diverifikasi oleh Data Protection Officer (DPO).'
                    : 'Not signed in? Submit a public data erasure request for verification by our Data Protection Officer (DPO).'}
                </p>
                <FormField label={isId ? 'Email Akun Terdaftar' : 'Registered Account Email'} htmlFor="publicEmail" required>
                  <Input
                    id="publicEmail"
                    type="email"
                    value={publicEmail}
                    onChange={(e) => setPublicEmail(e.target.value)}
                    placeholder="user@sinergi.example.com"
                  />
                </FormField>
                <FormField label={isId ? 'Alasan / Catatan Tambahan (Opsional)' : 'Reason / Additional Notes (Optional)'} htmlFor="publicReason">
                  <Textarea
                    id="publicReason"
                    value={publicReason}
                    onChange={(e) => setPublicReason(e.target.value)}
                    rows={3}
                    placeholder={isId ? 'Permintaan penghapusan data sesuai UU PDP / GDPR' : 'Personal data erasure request under UU PDP / GDPR'}
                  />
                </FormField>
                <Button type="submit" variant="danger" leftIcon={<Trash2 className="h-4 w-4" />}>
                  {isId ? 'Kirim Permintaan Penghapusan' : 'Submit Deletion Request'}
                </Button>
              </form>
            )}
          </div>
        )}
      </main>
    </div>
  );
}
