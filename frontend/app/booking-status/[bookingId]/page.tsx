'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { bookingApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import StatusBadge from '@/components/ui/StatusBadge';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function BookingStatusPage() {
  const params = useParams<{ bookingId: string }>();
  const router = useRouter();
  const bookingId = typeof params?.bookingId === 'string' ? params.bookingId : '';
  const [booking, setBooking] = useState<Booking | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!uuidPattern.test(bookingId)) {
      setNotFound(true);
      setLoading(false);
      return;
    }

    const fetchStatus = async () => {
      try {
        const res = await bookingApi.get(`/bookings/${bookingId}`);
        setBooking(res.data);
        setNotFound(false);
      } catch (error: unknown) {
        if (
          typeof error === 'object' &&
          error !== null &&
          'response' in error &&
          typeof (error as { response?: { status?: number } }).response?.status === 'number' &&
          (error as { response?: { status?: number } }).response?.status === 404
        ) {
          setNotFound(true);
          setBooking(null);
          return;
        }
        toast.error('Gagal memuat status booking');
      } finally {
        setLoading(false);
      }
    };
    void fetchStatus();
    const interval = setInterval(() => {
      void fetchStatus();
    }, 5000);
    return () => clearInterval(interval);
  }, [bookingId]);

  if (loading) return <p className="text-[var(--color-text-secondary)]">Memuat status booking...</p>;

  if (notFound || !booking) {
    return (
      <section className="mx-auto max-w-2xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Status Booking</h1>
        <p className="mt-3 text-sm text-[var(--color-error)]">Booking tidak ditemukan</p>
        <button
          onClick={() => router.push('/')}
          className="mt-5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text-primary)]"
        >
          Kembali ke beranda
        </button>
      </section>
    );
  }

  return (
    <section className="mx-auto max-w-2xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
      <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Status Booking</h1>
      <p className="mt-2 font-mono text-xs text-[var(--color-text-secondary)]">{booking.id}</p>
      <div className="mt-5">
        <StatusBadge status={booking.status} />
      </div>
      <p className="mt-4 text-sm text-[var(--color-text-secondary)]">Trip: {booking.resource_id}</p>
      <p className="mt-1 text-sm text-[var(--color-text-secondary)]">Total: {formatRupiah(booking.total_amount)}</p>
      <div className="mt-6 flex gap-3">
        <button
          onClick={() => router.push('/my-bookings')}
          className="rounded-xl border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text-primary)]"
        >
          Kembali ke Pesanan Saya
        </button>
      </div>
    </section>
  );
}
