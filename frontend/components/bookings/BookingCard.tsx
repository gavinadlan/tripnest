'use client';

import { Booking } from '@/lib/types';
import StatusBadge from '@/components/ui/StatusBadge';
import { formatRupiah } from '@/lib/currency';

export default function BookingCard({
  booking,
  onOpen,
}: {
  booking: Booking;
  onOpen?: (id: string) => void;
}) {
  return (
    <article className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-sm">
      <div className="mb-3 flex items-center justify-between">
        <p className="font-mono text-xs text-[var(--color-text-secondary)]">{booking.id}</p>
        <StatusBadge status={booking.status} />
      </div>
      <h3 className="text-base font-semibold text-[var(--color-text-primary)]">{booking.resource_id}</h3>
      <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
        Dibuat {new Date(booking.created_at).toLocaleString('id-ID')}
      </p>
      <p className="mt-3 text-lg font-semibold text-[var(--color-text-primary)]">{formatRupiah(booking.total_amount)}</p>
      {onOpen ? (
        <button
          onClick={() => onOpen(booking.id)}
          className="mt-4 rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm font-medium text-[var(--color-text-primary)] transition hover:bg-[var(--color-muted)]"
        >
          Lihat status
        </button>
      ) : null}
    </article>
  );
}
