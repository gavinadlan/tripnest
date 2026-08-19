'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import { userApi } from '@/lib/api';
import { decodeToken, setToken, clearToken } from '@/lib/auth';

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
      toast.error('Email dan password wajib diisi');
      return;
    }

    setLoading(true);
    const loadingToast = toast.loading('Memeriksa kredensial...');

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
      toast.error(status === 401 ? 'Kredensial tidak valid' : 'Gagal masuk, coba lagi', {
        id: loadingToast,
      });
      setLoading(false);
    }
  };

  return (
    <section className="flex min-h-screen items-center justify-center bg-[var(--color-background)] p-4">
      <div className="w-full max-w-md rounded-2xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Login Admin</h1>
        <p className="mt-2 text-sm text-[var(--color-text-secondary)]">
          Masuk dengan akun yang punya role admin.
        </p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
              placeholder="admin@tripnest.local"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
              placeholder="••••••••"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-lg bg-[var(--color-primary)] px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60"
          >
            {loading ? 'Memproses...' : 'Masuk Admin'}
          </button>
        </form>
      </div>
    </section>
  );
}
