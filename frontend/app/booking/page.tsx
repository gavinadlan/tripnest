'use client';

import { Suspense, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { bookingApi } from '@/lib/api';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

function extractBookingID(response: unknown): string {
  if (!response || typeof response !== 'object') return '';
  const root = response as Record<string, unknown>;

  const responseID = root.id;
  if (typeof responseID === 'string') return responseID;

  const data = root.data;
  if (!data || typeof data !== 'object') return '';
  const dataObj = data as Record<string, unknown>;

  if (typeof dataObj.id === 'string') return dataObj.id;

  const nestedData = dataObj.data;
  if (!nestedData || typeof nestedData !== 'object') return '';
  const nestedDataObj = nestedData as Record<string, unknown>;

  return typeof nestedDataObj.id === 'string' ? nestedDataObj.id : '';
}

function BookingContent() {
  const params = useSearchParams();
  const router = useRouter();
  const [loading, setLoading] = useState(false);

  const tripId = params.get('tripId') || '';
  const title = params.get('title') || 'Selected Trip';
  const destination = params.get('destination') || '-';
  const price = Number(params.get('price') || '0');
  const date = params.get('date') || '-';

  const createBooking = async () => {
    const userId = localStorage.getItem('user_id');
    const token = localStorage.getItem('token');

    if (!token || !userId) {
      toast.error('Silakan login terlebih dahulu');
      router.push('/login');
      return;
    }

    setLoading(true);
    try {
      const res = await bookingApi.post('/bookings', {
        user_id: userId,
        resource_id: tripId,
        total_amount: price,
      });
      const bookingId = extractBookingID(res);
      if (!bookingId || !uuidPattern.test(bookingId)) {
        toast.error('Respons booking tidak valid');
        return;
      }
      const existing = JSON.parse(localStorage.getItem('my_bookings') || '[]');
      localStorage.setItem('my_bookings', JSON.stringify([...new Set([...existing, bookingId])]));
      localStorage.setItem('last_booking_id', bookingId);
      toast.success('Booking berhasil dibuat');
      console.log('bookingId before redirect:', bookingId);
      router.push(`/payment/${bookingId}`);
    } catch {
      toast.error('Gagal membuat booking');
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="mx-auto max-w-3xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
      <p className="text-sm font-medium text-[var(--color-primary)]">Booking</p>
      <h1 className="mt-2 text-2xl font-bold text-[var(--color-text-primary)]">{title}</h1>
      <div className="mt-6 grid gap-3 text-sm text-[var(--color-text-secondary)]">
        <p><span className="font-semibold text-[var(--color-text-primary)]">Lokasi:</span> {destination}</p>
        <p><span className="font-semibold text-[var(--color-text-primary)]">Tanggal:</span> {date}</p>
        <p><span className="font-semibold text-[var(--color-text-primary)]">Total:</span> {formatRupiah(price)}</p>
      </div>

      <button
        disabled={loading || !tripId}
        onClick={createBooking}
        className="mt-8 rounded-xl bg-[var(--color-primary)] px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-slate-300"
      >
        {loading ? 'Membuat booking...' : 'Konfirmasi dan lanjut ke pembayaran'}
      </button>
    </section>
  );
}

export default function BookingPage() {
  return (
    <Suspense fallback={<p className="text-[var(--color-text-secondary)]">Memuat detail booking...</p>}>
      <BookingContent />
    </Suspense>
  );
}
