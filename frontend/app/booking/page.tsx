'use client';

import { Suspense, useCallback, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { bookingApi, searchApi } from '@/lib/api';
import { Trip } from '@/lib/types';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

const MAX_QUANTITY = 10;

function BookingContent() {
  const params = useSearchParams();
  const router = useRouter();

  const tripId = params.get('tripId') || '';
  const [trip, setTrip] = useState<Trip | null>(null);
  const [quantity, setQuantity] = useState(1);
  // Tanpa tripId tidak ada yang perlu dimuat, jadi jangan mulai dari
  // keadaan "loading" lalu mematikannya lewat setState di dalam effect.
  const [loadingTrip, setLoadingTrip] = useState(Boolean(tripId));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  /**
   * Detail trip diambil dari server, bukan dari query string.
   *
   * Versi sebelumnya menerima judul, tanggal, dan HARGA lewat URL lalu
   * mengirim harga itu apa adanya ke booking-service. Artinya siapa pun
   * bisa mengubah ?price= dan membayar trip mahal seharga seribu rupiah.
   * Sekarang harga hanya dipakai untuk ditampilkan; server menghitung
   * ulang totalnya dari katalog.
   */
  useEffect(() => {
    if (!tripId) return;

    let cancelled = false;
    const load = async () => {
      try {
        const res = await searchApi.get(`/trips/${tripId}`);
        if (!cancelled) setTrip(res.data);
      } catch {
        if (!cancelled) setError('Trip tidak ditemukan');
      } finally {
        if (!cancelled) setLoadingTrip(false);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [tripId]);

  const createBooking = useCallback(async () => {
    if (!trip) return;

    setSubmitting(true);
    try {
      const res = await bookingApi.post('/bookings', {
        resource_id: trip.id,
        quantity,
      });

      const bookingId: string | undefined = res.data?.id;
      if (!bookingId) {
        toast.error('Respons booking tidak valid');
        return;
      }

      toast.success('Booking dibuat, lanjutkan pembayaran');
      router.push(`/payment/${bookingId}`);
    } catch (err: unknown) {
      const status =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { status?: number } }).response?.status
          : undefined;

      if (status === 409) {
        toast.error('Kursi tidak mencukupi untuk jumlah ini');
      } else if (status === 404) {
        toast.error('Trip sudah tidak tersedia');
      } else if (status === 503) {
        toast.error('Layanan katalog sedang bermasalah, coba lagi sebentar');
      } else {
        toast.error('Gagal membuat booking');
      }
    } finally {
      setSubmitting(false);
    }
  }, [trip, quantity, router]);

  if (loadingTrip) {
    return <p className="text-[var(--color-text-secondary)]">Memuat detail trip...</p>;
  }

  const displayError = !tripId ? 'Trip tidak dipilih' : error;

  if (displayError || !trip) {
    return (
      <section className="mx-auto max-w-3xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Booking</h1>
        <p className="mt-3 text-sm text-[var(--color-error)]">{displayError || 'Trip tidak ditemukan'}</p>
        <button
          onClick={() => router.push('/')}
          className="mt-5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-sm font-medium"
        >
          Kembali ke beranda
        </button>
      </section>
    );
  }

  const maxSelectable = Math.min(MAX_QUANTITY, Math.max(trip.available_slots, 0));
  const total = trip.price * quantity;

  return (
    <section className="mx-auto max-w-3xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
      <p className="text-sm font-medium text-[var(--color-primary)]">Booking</p>
      <h1 className="mt-2 text-2xl font-bold text-[var(--color-text-primary)]">{trip.title}</h1>

      <div className="mt-6 grid gap-3 text-sm text-[var(--color-text-secondary)]">
        <p><span className="font-semibold text-[var(--color-text-primary)]">Lokasi:</span> {trip.destination}</p>
        <p><span className="font-semibold text-[var(--color-text-primary)]">Tanggal:</span> {trip.date}</p>
        <p><span className="font-semibold text-[var(--color-text-primary)]">Harga per kursi:</span> {formatRupiah(trip.price)}</p>
        <p><span className="font-semibold text-[var(--color-text-primary)]">Sisa kursi:</span> {trip.available_slots}</p>
      </div>

      <div className="mt-6">
        <label htmlFor="quantity" className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">
          Jumlah kursi
        </label>
        <select
          id="quantity"
          value={quantity}
          onChange={(e) => setQuantity(Number(e.target.value))}
          disabled={maxSelectable < 1}
          className="w-32 rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
        >
          {Array.from({ length: Math.max(maxSelectable, 1) }, (_, i) => i + 1).map((n) => (
            <option key={n} value={n}>{n}</option>
          ))}
        </select>
      </div>

      <p className="mt-6 text-lg font-semibold text-[var(--color-text-primary)]">
        Total: {formatRupiah(total)}
      </p>
      <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
        Total final dihitung ulang oleh server saat booking dibuat.
      </p>

      <button
        disabled={submitting || maxSelectable < 1}
        onClick={createBooking}
        className="mt-6 rounded-xl bg-[var(--color-primary)] px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-slate-300"
      >
        {submitting ? 'Membuat booking...' : maxSelectable < 1 ? 'Kursi habis' : 'Konfirmasi dan lanjut ke pembayaran'}
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
