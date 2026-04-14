'use client';

import { CalendarDays, MapPin, Users } from 'lucide-react';
import { Trip } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';

export default function TripCard({
  trip,
  onBook,
  loading,
}: {
  trip: Trip;
  onBook: (trip: Trip) => void;
  loading?: boolean;
}) {
  return (
    <article className="rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md">
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">{trip.title}</h3>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">{trip.destination}</p>
        </div>
        <span className="rounded-lg bg-[var(--color-muted)] px-3 py-1 text-sm font-semibold text-[var(--color-primary)]">
          {formatRupiah(trip.price)}
        </span>
      </div>

      <div className="space-y-2 text-sm text-[var(--color-text-secondary)]">
        <p className="flex items-center gap-2"><MapPin className="h-4 w-4" /> {trip.destination}</p>
        <p className="flex items-center gap-2"><CalendarDays className="h-4 w-4" /> {trip.date}</p>
        <p className="flex items-center gap-2"><Users className="h-4 w-4" /> Sisa {trip.available_slots} kursi</p>
      </div>

      <button
        onClick={() => onBook(trip)}
        disabled={loading || trip.available_slots <= 0}
        className="mt-5 w-full rounded-xl bg-[var(--color-primary)] px-4 py-2.5 text-sm font-semibold text-white transition hover:opacity-95 disabled:cursor-not-allowed disabled:bg-slate-300"
      >
        {loading ? 'Membuat booking...' : trip.available_slots > 0 ? 'Pesan Trip Ini' : 'Kursi Habis'}
      </button>
    </article>
  );
}
