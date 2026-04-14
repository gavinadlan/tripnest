'use client';

import { useRouter } from 'next/navigation';

export default function AdminNavbar() {
  const router = useRouter();

  const handleLogout = () => {
    localStorage.removeItem('admin_logged_in');
    router.push('/admin/login');
  };

  return (
    <header className="flex h-16 items-center justify-between border-b border-[var(--color-border)] bg-white px-6">
      <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">
        Admin Dashboard
      </h1>
      <button
        type="button"
        onClick={handleLogout}
        className="rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white transition hover:bg-red-600"
      >
        Logout
      </button>
    </header>
  );
}
