'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import { userApi } from '@/lib/api';
import { decodeToken, setToken, clearToken } from '@/lib/auth';
import { btnPrimary, input } from '@/lib/ui';
import Logo from '@/components/Logo';
import { ShieldCheck } from 'lucide-react';

/**
 * Login admin memakai endpoint yang sama dengan login user biasa.
 *
 * Versi sebelumnya mencocokkan "admin"/"admin123" di JavaScript client dan
 * menyimpan flag `admin_logged_in` di localStorage -- flag yang ternyata
 * tidak pernah diperiksa di mana pun. Sekarang hak admin ditentukan klaim
 * `role` di dalam JWT, dan setiap endpoint admin memverifikasinya di server.
 */
export default function AdminLoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!email.trim() || !password) {
      toast.error('Email dan kata sandi wajib diisi');
      return;
    }

    setLoading(true);
    const loadingToast = toast.loading('Memeriksa akun...');

    try {
      const response = await userApi.post('/login', { email: email.trim(), password });
      const token: string = response.data.token;

      if (decodeToken(token)?.role !== 'admin') {
        clearToken();
        toast.error('Akun ini bukan admin', { id: loadingToast });
        setLoading(false);
        return;
      }

      setToken(token);
      toast.success('Login admin berhasil', { id: loadingToast });
      router.replace('/admin');
    } catch (error: unknown) {
      const status =
        typeof error === 'object' && error !== null && 'response' in error
          ? (error as { response?: { status?: number } }).response?.status
          : undefined;
      toast.error(status === 401 ? 'Email atau kata sandi salah' : 'Gagal masuk, coba lagi', {
        id: loadingToast,
      });
      setLoading(false);
    }
  };

  return (
    <section className="on-dark flex min-h-screen items-center justify-center bg-dark-bg p-4">
      <div className="w-full max-w-md rounded-2xl border border-white/10 bg-surface p-8 shadow-2xl">
        <Logo className="h-8" href={null} />
        <h1 className="mt-6 flex items-center gap-2 text-2xl font-extrabold text-navy">
          <ShieldCheck className="h-6 w-6 text-biru" aria-hidden /> Masuk admin
        </h1>
        <p className="mt-2 text-sm text-ink-soft">Gunakan akun yang memiliki peran admin.</p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="email" className="mb-1.5 block text-sm font-semibold text-ink">Email</label>
            <input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" className={input} placeholder="admin@email.com" />
          </div>
          <div>
            <label htmlFor="password" className="mb-1.5 block text-sm font-semibold text-ink">Kata sandi</label>
            <input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" className={input} placeholder="Masukkan kata sandi" />
          </div>
          <button type="submit" disabled={loading} className={`${btnPrimary} w-full py-3`}>
            {loading ? 'Memproses...' : 'Masuk admin'}
          </button>
        </form>
      </div>
    </section>
  );
}
