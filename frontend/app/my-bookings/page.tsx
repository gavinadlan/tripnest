'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { bookingApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import BookingCard from '@/components/bookings/BookingCard';
import toast from 'react-hot-toast';

/**
 * Daftar booking diambil dari server berdasarkan token.
 *
 * Versi sebelumnya menyimpan daftar id di localStorage (`my_bookings`),
 * sehingga riwayat hilang begitu browser dibersihkan dan tidak pernah
 * sinkron antar perangkat. Backend sudah bisa memfilter per user; endpoint
 * GET /bookings kini otomatis dibatasi pada milik pemanggil.
 */
export default function MyBookingsPage() {
  const router = useRouter();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const res = await bookingApi.get('/bookings', { params: { limit: 50 } });
        if (!cancelled) setBookings(res.data?.data ?? []);
      } catch {
        // 401 sudah ditangani interceptor (redirect ke /login).
        if (!cancelled) toast.error('Gagal memuat daftar booking');
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section className="space-y-5">
      <div>
        <h1 className="text-3xl font-bold text-[var(--color-text-primary)]">Pesanan Saya</h1>
        <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
          Pantau status booking terbaru secara real-time.
        </p>
      </div>

      {loading ? <p className="text-[var(--color-text-secondary)]">Memuat daftar booking...</p> : null}

      {!loading && bookings.length === 0 ? (
        <div className="rounded-2xl border border-[var(--color-border)] bg-white p-8 text-center text-[var(--color-text-secondary)]">
          Belum ada booking.
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2">
        {bookings.map((booking) => (
          <BookingCard
            key={booking.id}
            booking={booking}
            onOpen={(id) => router.push(`/booking-status/${id}`)}
          />
        ))}
      </div>
    </section>
  );
}
