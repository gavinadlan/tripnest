'use client';

import { CalendarDays, MapPin, Users } from 'lucide-react';
import { Trip } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';
import { formatTanggal } from '@/lib/format';
import { btnPrimary } from '@/lib/ui';
import TripArt from '@/components/trips/TripArt';

/** Hampir penuh bila tersisa sedikit: maksimal 3 kursi atau 20% dari kapasitas. */
export function almostFull(trip: Pick<Trip, 'available_slots' | 'total_slots'>): boolean {
  return trip.available_slots > 0 && (trip.available_slots <= 3 || trip.available_slots <= trip.total_slots * 0.2);
}

export default function TripCard({ trip, onBook, loading }: { trip: Trip; onBook: (trip: Trip) => void; loading?: boolean }) {
  const soldOut = trip.available_slots <= 0;
  const few = almostFull(trip);

  return (
    <article className="flex h-full flex-col overflow-hidden rounded-2xl border border-garis bg-surface">
      <div className="relative h-40 overflow-hidden">
        <TripArt id={trip.id} destination={trip.destination} title={trip.title} className="h-full w-full" />
        <span className="font-display absolute right-3 top-3 rounded-full bg-amber px-3 py-1 text-sm font-bold text-navy">
          {formatRupiah(trip.price)}
        </span>
        {soldOut || few ? (
          <span className={`absolute left-3 top-3 rounded-full px-3 py-1 text-xs font-bold ${soldOut ? 'bg-ink text-white' : 'bg-surface text-warn'}`}>
            {soldOut ? 'Kursi habis' : 'Hampir penuh'}
          </span>
        ) : null}
      </div>

      <div className="flex flex-1 flex-col p-5">
        <h3 className="text-xl font-bold leading-tight text-navy">{trip.title}</h3>
        <ul className="mt-3 space-y-1.5 text-sm text-ink-soft">
          <li className="flex items-center gap-2"><MapPin className="h-4 w-4 shrink-0" aria-hidden /> {trip.destination}</li>
          <li className="flex items-center gap-2"><CalendarDays className="h-4 w-4 shrink-0" aria-hidden /> {formatTanggal(trip.date)}</li>
          <li className="flex items-center gap-2"><Users className="h-4 w-4 shrink-0" aria-hidden /> Sisa {trip.available_slots} kursi</li>
        </ul>
        <button onClick={() => onBook(trip)} disabled={loading || soldOut} className={`${btnPrimary} mt-5 w-full`}>
          {loading ? 'Membuat booking...' : soldOut ? 'Kursi habis' : 'Pesan trip ini'}
        </button>
      </div>
    </article>
  );
}
