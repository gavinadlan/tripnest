'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { bookingApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import BookingCard from '@/components/bookings/BookingCard';
import toast from 'react-hot-toast';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function MyBookingsPage() {
  const router = useRouter();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      const idsRaw: string[] = JSON.parse(localStorage.getItem('my_bookings') || '[]');
      const ids = idsRaw.filter((id) => uuidPattern.test(id));
      if (ids.length !== idsRaw.length) {
        localStorage.setItem('my_bookings', JSON.stringify(ids));
      }
      if (!ids.length) {
        setLoading(false);
        return;
      }
      try {
        const results = await Promise.all(ids.map((id) => bookingApi.get(`/bookings/${id}`).catch(() => null)));
        const valid = results.filter(Boolean).map((res) => (res as { data: Booking }).data);
        const validIDs = new Set(valid.map((booking) => booking.id));
        localStorage.setItem(
          'my_bookings',
          JSON.stringify(ids.filter((id) => validIDs.has(id)))
        );
        valid.sort((a: Booking, b: Booking) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
        setBookings(valid);
      } catch {
        toast.error('Gagal memuat daftar booking');
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  return (
    <section className="space-y-5">
      <div>
        <h1 className="text-3xl font-bold text-[var(--color-text-primary)]">Pesanan Saya</h1>
        <p className="mt-1 text-sm text-[var(--color-text-secondary)]">Pantau status booking terbaru secara real-time.</p>
      </div>

      {loading ? <p className="text-[var(--color-text-secondary)]">Memuat daftar booking...</p> : null}
      {!loading && bookings.length === 0 ? (
        <div className="rounded-2xl border border-[var(--color-border)] bg-white p-8 text-center text-[var(--color-text-secondary)]">
          Belum ada booking.
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2">
        {bookings.map((booking) => (
          <BookingCard key={booking.id} booking={booking} onOpen={(id) => router.push(`/booking-status/${id}`)} />
        ))}
      </div>
    </section>
  );
}
