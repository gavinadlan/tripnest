'use client';

import { useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import AdminSidebar from '@/components/admin/AdminSidebar';
import AdminTopbar from '@/components/admin/AdminTopbar';

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const isLoginPage = pathname === '/admin/login';
  const isLoggedIn = typeof window !== 'undefined' && localStorage.getItem('admin_logged_in') === 'true';

  useEffect(() => {
    if (!isLoginPage && !isLoggedIn) {
      router.replace('/admin/login');
    }
  }, [isLoginPage, isLoggedIn, router]);

  if (isLoginPage) {
    return <>{children}</>;
  }

  if (!isLoggedIn) {
    return <p className="p-6 text-sm text-[var(--color-text-secondary)]">Memverifikasi akses admin...</p>;
  }

  return (
    <div className="flex min-h-screen bg-[var(--color-background)]">
      <AdminSidebar />
      <div className="flex flex-1 flex-col">
        <AdminTopbar />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
