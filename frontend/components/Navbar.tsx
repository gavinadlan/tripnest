'use client';

import Link from 'next/link';
import { useRouter, usePathname } from 'next/navigation';
import { useCallback, useState, useSyncExternalStore } from 'react';
import { LogOut, Menu, X } from 'lucide-react';
import { clearToken, isAdmin, isLoggedIn } from '@/lib/auth';
import Logo from '@/components/Logo';
import { btnGhost, btnPrimary } from '@/lib/ui';

/**
 * Status sesi dibaca lewat useSyncExternalStore, bukan useEffect + setState.
 *
 * Cookie adalah state di luar React. useSyncExternalStore menangani
 * perbedaan render server dan client tanpa flag `mounted`.
 */
function subscribe(onChange: () => void) {
  window.addEventListener('storage', onChange);
  window.addEventListener('focus', onChange);
  return () => {
    window.removeEventListener('storage', onChange);
    window.removeEventListener('focus', onChange);
  };
}

const serverFalse = () => false;

export default function Navbar() {
  const router = useRouter();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);

  const loggedIn = useSyncExternalStore(subscribe, isLoggedIn, serverFalse);
  const admin = useSyncExternalStore(subscribe, isAdmin, serverFalse);

  const handleLogout = useCallback(() => {
    clearToken();
    setOpen(false);
    router.push('/login');
  }, [router]);

  const navLinks = [
    { name: 'Trip', href: '/' },
    { name: 'Pesanan Saya', href: '/my-bookings' },
    // Tautan admin hanya tampilan; server tetap memverifikasi role.
    ...(admin ? [{ name: 'Admin', href: '/admin' }] : []),
  ];

  const linkClass = (href: string) =>
    `inline-flex items-center border-b-2 px-1 py-1 text-sm font-semibold transition ${
      pathname === href ? 'border-amber text-navy' : 'border-transparent text-ink-soft hover:text-ink'
    }`;

  return (
    <nav className="sticky top-0 z-50 border-b border-garis bg-krem/95 backdrop-blur" aria-label="Navigasi utama">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-10">
          <Logo className="h-9" />
          <div className="hidden gap-7 sm:flex">
            {navLinks.map((l) => (
              <Link key={l.href} href={l.href} className={linkClass(l.href)} aria-current={pathname === l.href ? 'page' : undefined}>
                {l.name}
              </Link>
            ))}
          </div>
        </div>

        <div className="hidden items-center gap-3 sm:flex">
          {loggedIn ? (
            <button onClick={handleLogout} className={`${btnGhost} !py-2`}>
              <LogOut className="h-4 w-4" aria-hidden /> Keluar
            </button>
          ) : (
            <>
              <Link href="/login" className={`${btnGhost} !py-2`}>Masuk</Link>
              <Link href="/register" className={`${btnPrimary} !py-2`}>Daftar</Link>
            </>
          )}
        </div>

        <button
          className="rounded-lg p-2 text-ink sm:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label={open ? 'Tutup menu' : 'Buka menu'}
          aria-expanded={open}
        >
          {open ? <X className="h-6 w-6" aria-hidden /> : <Menu className="h-6 w-6" aria-hidden />}
        </button>
      </div>

      {open && (
        <div className="border-t border-garis bg-krem px-4 pb-4 pt-2 sm:hidden">
          {navLinks.map((l) => (
            <Link key={l.href} href={l.href} onClick={() => setOpen(false)} className="block rounded-lg px-2 py-3 text-sm font-semibold text-ink">
              {l.name}
            </Link>
          ))}
          <div className="mt-2 flex gap-3 border-t border-garis pt-4">
            {loggedIn ? (
              <button onClick={handleLogout} className={btnGhost}>Keluar</button>
            ) : (
              <>
                <Link href="/login" onClick={() => setOpen(false)} className={btnGhost}>Masuk</Link>
                <Link href="/register" onClick={() => setOpen(false)} className={btnPrimary}>Daftar</Link>
              </>
            )}
          </div>
        </div>
      )}
    </nav>
  );
}
