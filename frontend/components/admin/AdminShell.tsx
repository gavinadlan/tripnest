'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { Boxes, CreditCard, ExternalLink, LayoutDashboard, LogOut, MapPinned, NotebookTabs } from 'lucide-react';
import toast from 'react-hot-toast';
import { clearToken } from '@/lib/auth';
import Logo from '@/components/Logo';

const links = [
  { href: '/admin', label: 'Ringkasan', icon: LayoutDashboard },
  { href: '/admin/bookings', label: 'Booking', icon: NotebookTabs },
  { href: '/admin/payments', label: 'Pembayaran', icon: CreditCard },
  { href: '/admin/trips', label: 'Trip', icon: MapPinned },
  { href: '/admin/inventory', label: 'Inventaris', icon: Boxes },
];

const titles: Record<string, string> = {
  '/admin': 'Ringkasan sistem',
  '/admin/bookings': 'Manajemen booking',
  '/admin/payments': 'Monitoring pembayaran',
  '/admin/trips': 'Manajemen trip',
  '/admin/inventory': 'Inventaris kursi',
};

/** Kerangka admin. Halaman /admin/login tampil polos, tanpa sidebar dan tombol keluar. */
export default function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();

  if (pathname === '/admin/login') return <>{children}</>;

  const logout = () => {
    clearToken();
    toast.success('Berhasil keluar dari admin');
    router.push('/admin/login');
  };

  const isActive = (href: string) => (href === '/admin' ? pathname === href : pathname === href || pathname.startsWith(`${href}/`));

  return (
    <div className="min-h-screen lg:flex">
      <aside className="on-dark bg-dark-bg text-white lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-64 lg:shrink-0 lg:flex-col lg:p-5">
        <div className="flex items-center justify-between px-4 py-3 lg:block lg:px-0 lg:py-0">
          <Logo dark className="h-8" href="/admin" />
          <p className="mt-0 text-[11px] font-semibold uppercase tracking-[0.18em] text-biru-muda lg:mt-3">Panel admin</p>
        </div>
        <nav aria-label="Navigasi admin" className="flex gap-1 px-2 pb-3 lg:mt-8 lg:flex-1 lg:flex-col lg:px-0 lg:pb-0">
          {links.map(({ href, label, icon: Icon }) => (
            <Link
              key={href}
              href={href}
              aria-current={isActive(href) ? 'page' : undefined}
              className={`flex flex-1 flex-col items-center gap-1 rounded-xl px-1 py-2 text-[11px] font-semibold transition lg:flex-none lg:flex-row lg:gap-3 lg:px-3 lg:py-2.5 lg:text-sm ${
                isActive(href) ? 'bg-amber text-navy' : 'text-white/80 hover:bg-white/10 hover:text-white'
              }`}
            >
              <Icon className="h-4 w-4" aria-hidden />
              {label}
            </Link>
          ))}
        </nav>
        <div className="hidden space-y-1 border-t border-white/10 pt-4 lg:block">
          <Link href="/" className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-white/80 hover:bg-white/10 hover:text-white">
            <ExternalLink className="h-4 w-4" aria-hidden /> Ke aplikasi
          </Link>
          <button onClick={logout} className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-semibold text-white/80 hover:bg-white/10 hover:text-white">
            <LogOut className="h-4 w-4" aria-hidden /> Keluar
          </button>
        </div>
      </aside>

      <div className="min-w-0 flex-1">
        <header className="flex items-center justify-between gap-3 border-b border-garis px-5 py-4 lg:px-8">
          <h1 className="text-2xl font-bold text-navy">{titles[pathname] ?? 'Panel admin'}</h1>
          <button onClick={logout} className="rounded-lg border border-garis px-3 py-1.5 text-sm font-semibold text-ink lg:hidden">Keluar</button>
        </header>
        <main className="p-5 lg:p-8">{children}</main>
      </div>
    </div>
  );
}
