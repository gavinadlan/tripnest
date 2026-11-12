'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Ticket } from 'lucide-react';
import toast from 'react-hot-toast';
import { bookingApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import { useTrips } from '@/lib/trips';
import { btnPrimary } from '@/lib/ui';
import BookingCard from '@/components/bookings/BookingCard';

const ACTIVE = new Set(['PENDING_SEAT', 'PENDING_PAYMENT']);

/**
 * Daftar booking diambil dari server berdasarkan token; GET /bookings
 * otomatis dibatasi pada milik pemanggil.
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

  const trips = useTrips(bookings.map((b) => b.resource_id));

  const { aktif, riwayat } = useMemo(() => {
    const sorted = [...bookings].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
    return { aktif: sorted.filter((b) => ACTIVE.has(b.status)), riwayat: sorted.filter((b) => !ACTIVE.has(b.status)) };
  }, [bookings]);

  const list = (items: Booking[]) => (
    <div className="grid gap-4 lg:grid-cols-2">
      {items.map((b) => (
        <BookingCard key={b.id} booking={b} trip={trips[b.resource_id]} onOpen={(id) => router.push(`/booking-status/${id}`)} />
      ))}
    </div>
  );

  return (
    <section className="space-y-8">
      <div>
        <h1 className="text-4xl font-extrabold text-navy">Pesanan Saya</h1>
        <p className="mt-1 text-ink-soft">Pantau status pesananmu. Halaman status menyegarkan dirinya sendiri.</p>
      </div>

      {loading ? (
        <div role="status" aria-label="Memuat daftar booking" className="grid gap-4 lg:grid-cols-2">
          {[0, 1, 2, 3].map((i) => <div key={i} className="h-44 animate-pulse rounded-2xl bg-krem-tua/60" />)}
        </div>
      ) : null}

      {!loading && bookings.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-garis bg-surface px-6 py-14 text-center">
          <Ticket className="mx-auto h-10 w-10 text-ink-soft" aria-hidden />
          <h2 className="mt-4 text-2xl font-bold text-navy">Belum ada pesanan</h2>
          <p className="mt-2 text-ink-soft">Pilih trip pertamamu dan kursinya langsung kami tahan.</p>
          <Link href="/" className={`${btnPrimary} mt-5`}>Jelajahi trip</Link>
        </div>
      ) : null}

      {aktif.length > 0 ? (
        <div>
          <h2 className="mb-3 text-xl font-bold text-navy">Aktif</h2>
          {list(aktif)}
        </div>
      ) : null}
      {riwayat.length > 0 ? (
        <div>
          <h2 className="mb-3 text-xl font-bold text-navy">Riwayat</h2>
          {list(riwayat)}
        </div>
      ) : null}
    </section>
  );
}
