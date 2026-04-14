'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { BarChart3, Boxes, CreditCard, MapPin, NotebookTabs } from 'lucide-react';

const links = [
  { href: '/admin', label: 'Ringkasan', icon: BarChart3 },
  { href: '/admin/bookings', label: 'Booking', icon: NotebookTabs },
  { href: '/admin/payments', label: 'Pembayaran', icon: CreditCard },
  { href: '/admin/trips', label: 'Trip', icon: MapPin },
  { href: '/admin/inventory', label: 'Inventaris', icon: Boxes },
];

export default function AdminSidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 border-r border-[var(--color-border)] bg-white p-4">
      <h2 className="px-3 py-2 text-sm font-semibold text-[var(--color-text-secondary)]">
        Dashboard Admin
      </h2>

      <nav className="mt-2 space-y-1">
        {links.map(({ href, label, icon: Icon }) => {
          const active =
            href === '/admin'
              ? pathname === href
              : pathname === href || pathname.startsWith(`${href}/`);

          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition ${
                active
                  ? 'bg-[var(--color-muted)] text-[var(--color-primary)]'
                  : 'text-[var(--color-text-secondary)] hover:bg-slate-50 hover:text-[var(--color-text-primary)]'
              }`}
            >
              <Icon className="h-4 w-4" />
              {label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}