'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { searchApi } from '@/lib/api';
import { Trip } from '@/lib/types';
import TripCard from '@/components/trips/TripCard';
import toast from 'react-hot-toast';

export default function HomePage() {
  const router = useRouter();
  const [trips, setTrips] = useState<Trip[]>([]);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState('');
  const [bookingTripId, setBookingTripId] = useState<string | null>(null);

  useEffect(() => {
    const fetchTrips = async () => {
      try {
        const res = await searchApi.get('/search');
        setTrips(res.data.data || []);
      } catch {
        toast.error('Gagal memuat daftar trip');
      } finally {
        setLoading(false);
      }
    };
    fetchTrips();
  }, []);

  const filteredTrips = useMemo(
    () => trips.filter((trip) => `${trip.title} ${trip.destination}`.toLowerCase().includes(query.toLowerCase())),
    [query, trips]
  );

  const onBook = (trip: Trip) => {
    setBookingTripId(trip.id);
    const params = new URLSearchParams({
      tripId: trip.id,
      title: trip.title,
      destination: trip.destination,
      price: String(trip.price),
      date: trip.date,
    });
    router.push(`/booking?${params.toString()}`);
  };

  return (
    <section className="space-y-8">
      <div className="rounded-3xl bg-gradient-to-r from-[var(--color-muted)] to-white p-8 shadow-sm">
        <p className="text-sm font-medium text-[var(--color-primary)]">Temukan Perjalanan Impianmu</p>
        <h1 className="mt-2 text-4xl font-bold text-[var(--color-text-primary)]">Booking travel modern, cepat, dan aman.</h1>
        <p className="mt-3 max-w-2xl text-[var(--color-text-secondary)]">
          Jelajahi trip yang tersedia, booking dalam hitungan detik, lalu selesaikan pembayaran aman lewat Midtrans.
        </p>
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Cari destinasi atau nama trip"
          className="mt-6 w-full max-w-lg rounded-xl border border-[var(--color-border)] bg-white px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none ring-[var(--color-primary)]/20 focus:ring-4"
        />
      </div>

      {loading ? (
        <p className="text-[var(--color-text-secondary)]">Memuat daftar trip...</p>
      ) : (
        <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
          {filteredTrips.map((trip) => (
            <TripCard key={trip.id} trip={trip} onBook={onBook} loading={bookingTripId === trip.id} />
          ))}
        </div>
      )}
    </section>
  );
}
