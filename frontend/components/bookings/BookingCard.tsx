'use client';

import { Booking, Trip } from '@/lib/types';
import StatusBadge from '@/components/ui/StatusBadge';
import TripArt from '@/components/trips/TripArt';
import { formatRupiah } from '@/lib/currency';
import { formatWaktu } from '@/lib/format';
import { tripLabel } from '@/lib/trips';
import { btnGhost } from '@/lib/ui';

const STEPS = ['PENDING_SEAT', 'PENDING_PAYMENT', 'CONFIRMED'];

/** Tiga titik progres kecil. CANCELLED/EXPIRED ditandai satu titik akhir berbeda. */
function MiniSteps({ status }: { status: string }) {
  const failed = status === 'CANCELLED' || status === 'EXPIRED';
  const cur = STEPS.indexOf(status);
  return (
    <div className="flex items-center gap-1.5" aria-hidden>
      {STEPS.map((s, i) => (
        <span
          key={s}
          className={`h-1.5 w-8 rounded-full ${
            failed ? (i < 2 ? 'bg-ink-soft/30' : status === 'CANCELLED' ? 'bg-bad' : 'bg-mute') : i <= cur ? (status === 'CONFIRMED' ? 'bg-ok' : 'bg-biru') : 'bg-garis'
          }`}
        />
      ))}
    </div>
  );
}

export default function BookingCard({
  booking,
  trip,
  onOpen,
}: {
  booking: Booking;
  trip: Trip | null | undefined;
  onOpen?: (id: string) => void;
}) {
  return (
    <article className="flex overflow-hidden rounded-2xl border border-garis bg-surface">
      <div className="relative hidden w-32 shrink-0 sm:block">
        <TripArt id={booking.resource_id} destination={trip?.destination ?? ''} title={trip?.title ?? ''} className="absolute inset-0 h-full w-full" />
      </div>
      <div className="min-w-0 flex-1 p-5">
        <div className="flex items-start justify-between gap-3">
          <h3 className="text-lg font-bold leading-tight text-navy">{tripLabel(trip, booking.resource_id)}</h3>
          <StatusBadge status={booking.status} />
        </div>
        <p className="mt-1 text-sm text-ink-soft">
          {booking.quantity} kursi &middot; dibuat {formatWaktu(booking.created_at)}
        </p>
        <div className="mt-3 flex items-center justify-between gap-3">
          <p className="font-display text-2xl font-extrabold text-navy">{formatRupiah(booking.total_amount)}</p>
          <MiniSteps status={booking.status} />
        </div>
        {onOpen ? (
          <button onClick={() => onOpen(booking.id)} className={`${btnGhost} mt-4 !px-4 !py-2`}>
            Lihat status
          </button>
        ) : null}
      </div>
    </article>
  );
}
