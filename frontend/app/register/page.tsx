'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import toast from 'react-hot-toast';
import { Lock, Mail, User, UserPlus } from 'lucide-react';
import { userApi } from '@/lib/api';
import { serverMessage } from '@/lib/errors';
import { btnPrimary, input } from '@/lib/ui';
import AuthLayout from '@/components/AuthLayout';

export default function RegisterPage() {
  const router = useRouter();
  const [formData, setFormData] = useState({ email: '', password: '', first_name: '', last_name: '' });
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    const loadingToast = toast.loading('Membuat akunmu...');

    try {
      await userApi.post('/register', formData);
      toast.success('Akun berhasil dibuat. Silakan masuk.', { id: loadingToast });
      router.push('/login');
    } catch (error: unknown) {
      toast.error(serverMessage(error) ?? 'Gagal mendaftar, coba lagi', { id: loadingToast });
      setLoading(false);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  return (
    <AuthLayout title="Buat akun TripNest" subtitle="Daftar sekali, lalu pesan trip dan pantau statusnya kapan saja." artId="auth-register-art">
      <form className="space-y-5" onSubmit={handleSubmit}>
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="first_name" className="mb-1.5 block text-sm font-semibold text-ink">Nama depan</label>
            <div className="relative">
              <User className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-soft" aria-hidden />
              <input id="first_name" name="first_name" required className={`${input} pl-11`} placeholder="Maya" value={formData.first_name} onChange={handleChange} />
            </div>
          </div>
          <div>
            <label htmlFor="last_name" className="mb-1.5 block text-sm font-semibold text-ink">Nama belakang</label>
            <input id="last_name" name="last_name" required className={input} placeholder="Kusuma" value={formData.last_name} onChange={handleChange} />
          </div>
        </div>
        <div>
          <label htmlFor="email" className="mb-1.5 block text-sm font-semibold text-ink">Email</label>
          <div className="relative">
            <Mail className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-soft" aria-hidden />
            <input id="email" name="email" type="email" autoComplete="email" required className={`${input} pl-11`} placeholder="nama@email.com" value={formData.email} onChange={handleChange} />
          </div>
        </div>
        <div>
          <label htmlFor="password" className="mb-1.5 block text-sm font-semibold text-ink">Kata sandi</label>
          <div className="relative">
            <Lock className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-soft" aria-hidden />
            <input id="password" name="password" type="password" autoComplete="new-password" required className={`${input} pl-11`} placeholder="Buat kata sandi" value={formData.password} onChange={handleChange} />
          </div>
        </div>
        <button type="submit" disabled={loading} className={`${btnPrimary} w-full py-3`}>
          <UserPlus className="h-4 w-4" aria-hidden /> {loading ? 'Membuat akun...' : 'Daftar'}
        </button>
      </form>
      <p className="mt-6 text-sm text-ink-soft">
        Sudah punya akun?{' '}
        <Link href="/login" className="font-bold text-biru underline-offset-4 hover:underline">Masuk</Link>
      </p>
    </AuthLayout>
  );
}
