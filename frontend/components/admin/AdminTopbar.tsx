'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { usePathname } from 'next/navigation';
import toast from 'react-hot-toast';

export default function AdminTopbar() {
  const pathname = usePathname();
  const router = useRouter();
  const titleMap: Record<string, string> = {
    '/admin': 'Ringkasan Sistem',
    '/admin/bookings': 'Manajemen Booking',
    '/admin/payments': 'Monitoring Pembayaran',
    '/admin/trips': 'Manajemen Trip',
    '/admin/inventory': 'Manajemen Inventaris',
  };
  const title = titleMap[pathname] || 'Dashboard Admin';

  const logoutAdmin = () => {
    localStorage.removeItem('admin_logged_in');
    toast.success('Berhasil keluar dari admin');
    router.push('/admin/login');
  };

  return (
    <header className="flex items-center justify-between border-b border-[var(--color-border)] bg-white px-6 py-4">
      <div>
        <p className="text-xs font-medium text-[var(--color-text-secondary)]">TripNest</p>
        <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">{title}</h1>
      </div>
      <div className="flex items-center gap-2">
        <Link href="/" className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm font-medium text-[var(--color-text-primary)] hover:bg-slate-50">
          Ke Aplikasi
        </Link>
        <button onClick={logoutAdmin} className="rounded-lg bg-[var(--color-error)] px-3 py-2 text-sm font-medium text-white">
          Keluar Admin
        </button>
      </div>
    </header>
  );
}
