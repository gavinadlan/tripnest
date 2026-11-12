'use client';

import { Suspense, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import toast from 'react-hot-toast';
import { LogIn, Mail, Lock } from 'lucide-react';
import { userApi } from '@/lib/api';
import { setToken } from '@/lib/auth';
import { serverMessage, statusOf } from '@/lib/errors';
import { btnPrimary, input } from '@/lib/ui';
import AuthLayout from '@/components/AuthLayout';

function LoginForm() {
  const searchParams = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    const loadingToast = toast.loading('Sedang masuk...');

    try {
      const response = await userApi.post('/login', { email, password });
      setToken(response.data.token);
      toast.success('Selamat datang kembali!', { id: loadingToast });

      // Reload penuh supaya Navbar dan proxy membaca cookie baru.
      const next = searchParams.get('next');
      window.location.href = next && next.startsWith('/') ? next : '/';
    } catch (error: unknown) {
      toast.error(statusOf(error) === 401 ? 'Email atau kata sandi salah' : serverMessage(error) ?? 'Gagal masuk, coba lagi', {
        id: loadingToast,
      });
      setLoading(false);
    }
  };

  return (
    <AuthLayout title="Selamat datang kembali" subtitle="Masuk untuk melihat pesananmu dan menjelajahi trip berikutnya." artId="auth-login-art">
      <form className="space-y-5" onSubmit={handleSubmit}>
        <div>
          <label htmlFor="email" className="mb-1.5 block text-sm font-semibold text-ink">Email</label>
          <div className="relative">
            <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-soft" aria-hidden />
            <input id="email" name="email" type="email" autoComplete="email" required className={`${input} pl-11`} placeholder="nama@email.com" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
        </div>
        <div>
          <label htmlFor="password" className="mb-1.5 block text-sm font-semibold text-ink">Kata sandi</label>
          <div className="relative">
            <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-soft" aria-hidden />
            <input id="password" name="password" type="password" autoComplete="current-password" required className={`${input} pl-11`} placeholder="Masukkan kata sandi" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
        </div>
        <button type="submit" disabled={loading} className={`${btnPrimary} w-full py-3`}>
          <LogIn className="h-4 w-4" aria-hidden /> {loading ? 'Sedang masuk...' : 'Masuk'}
        </button>
      </form>
      <p className="mt-6 text-sm text-ink-soft">
        Belum punya akun?{' '}
        <Link href="/register" className="font-bold text-biru underline-offset-4 hover:underline">Daftar gratis</Link>
      </p>
    </AuthLayout>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<p className="p-8 text-center text-ink-soft">Memuat...</p>}>
      <LoginForm />
    </Suspense>
  );
}
