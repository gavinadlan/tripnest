'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import toast from 'react-hot-toast';

export default function AdminLoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    if (!username.trim() || !password.trim()) {
      toast.error('Username dan password wajib diisi');
      setLoading(false);
      return;
    }

    if (username.trim() !== 'admin' || password !== 'admin123') {
      toast.error('Kredensial admin tidak valid');
      setLoading(false);
      return;
    }

    localStorage.setItem('admin_logged_in', 'true');
    toast.success('Login admin berhasil');
    router.replace('/admin');
  };

  return (
    <section className="flex min-h-screen items-center justify-center bg-[var(--color-background)] p-4">
      <div className="w-full max-w-md rounded-2xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Login Admin</h1>
        <p className="mt-2 text-sm text-[var(--color-text-secondary)]">Masuk untuk mengelola dashboard TripNest.</p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Username</label>
            <input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
              placeholder="admin"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
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
