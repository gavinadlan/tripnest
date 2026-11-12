'use client';

import { Suspense, useCallback, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { CalendarDays, Armchair, MapPin, Minus, Plus, ShieldCheck } from 'lucide-react';
import toast from 'react-hot-toast';
import { bookingApi, searchApi } from '@/lib/api';
import { Trip } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';
import { formatTanggal } from '@/lib/format';
import { statusOf } from '@/lib/errors';
import { btnGhost, btnPrimary, card } from '@/lib/ui';
import TripArt from '@/components/trips/TripArt';

const MAX_QUANTITY = 10;

function BookingContent() {
  const params = useSearchParams();
  const router = useRouter();

  const tripId = params.get('tripId') || '';
  const [trip, setTrip] = useState<Trip | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [loadingTrip, setLoadingTrip] = useState(Boolean(tripId));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  /**
   * Detail trip diambil dari server, bukan dari query string. Harga hanya
   * dipakai untuk ditampilkan; server menghitung ulang total dari katalog.
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
      const res = await bookingApi.post('/bookings', { resource_id: trip.id, quantity });

      const bookingId: string | undefined = res.data?.id;
      if (!bookingId) {
        toast.error('Respons booking tidak valid');
        return;
      }

      toast.success('Booking dibuat, lanjutkan pembayaran');
      router.push(`/payment/${bookingId}`);
    } catch (err: unknown) {
      const status = statusOf(err);
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
    return <div role="status" className="mx-auto h-96 max-w-3xl animate-pulse rounded-3xl bg-krem-tua/60" aria-label="Memuat detail trip" />;
  }

  const displayError = !tripId ? 'Trip tidak dipilih' : error;

  if (displayError || !trip) {
    return (
      <section className={`${card} mx-auto max-w-3xl p-8`}>
        <h1 className="text-2xl font-bold text-navy">Booking</h1>
        <p className="mt-3 text-sm font-semibold text-bad">{displayError || 'Trip tidak ditemukan'}</p>
        <button onClick={() => router.push('/')} className={`${btnGhost} mt-5`}>Kembali ke beranda</button>
      </section>
    );
  }

  const maxSelectable = Math.min(MAX_QUANTITY, Math.max(trip.available_slots, 0));
  const total = trip.price * quantity;

  return (
    <section className={`${card} mx-auto max-w-3xl overflow-hidden`}>
      <div className="on-dark relative h-52 sm:h-64">
        <TripArt id={trip.id} destination={trip.destination} title={trip.title} className="h-full w-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-dark-bg/95 via-dark-bg/75 to-dark-bg/55" />
        <div className="absolute inset-x-0 bottom-0 p-6 text-white">
          <p className="text-xs font-bold uppercase tracking-[0.2em] text-amber">Pesan trip</p>
          <h1 className="mt-1 text-3xl font-extrabold sm:text-4xl">{trip.title}</h1>
        </div>
      </div>

      <div className="p-6 sm:p-8">
        <dl className="grid gap-4 text-sm sm:grid-cols-2">
          <div className="flex items-start gap-3"><MapPin className="mt-0.5 h-4 w-4 text-biru" aria-hidden /><div><dt className="text-ink-soft">Lokasi</dt><dd className="font-semibold text-ink">{trip.destination}</dd></div></div>
          <div className="flex items-start gap-3"><CalendarDays className="mt-0.5 h-4 w-4 text-biru" aria-hidden /><div><dt className="text-ink-soft">Tanggal</dt><dd className="font-semibold text-ink">{formatTanggal(trip.date)}</dd></div></div>
          <div className="flex items-start gap-3"><span className="mt-0.5 w-4 text-center text-xs font-bold text-biru" aria-hidden>Rp</span><div><dt className="text-ink-soft">Harga per kursi</dt><dd className="font-semibold text-ink">{formatRupiah(trip.price)}</dd></div></div>
          <div className="flex items-start gap-3"><Armchair className="mt-0.5 h-4 w-4 text-biru" aria-hidden /><div><dt className="text-ink-soft">Sisa kursi</dt><dd className="font-semibold text-ink">{trip.available_slots}</dd></div></div>
        </dl>

        <div className="mt-8 flex flex-wrap items-center justify-between gap-4 rounded-2xl bg-krem p-4">
          <div>
            <p id="qty-label" className="text-sm font-semibold text-ink">Jumlah kursi</p>
            <p className="text-xs text-ink-soft">Maksimal {MAX_QUANTITY} kursi per pesanan</p>
          </div>
          <div role="group" aria-labelledby="qty-label" className="flex items-center gap-3">
            <button type="button" onClick={() => setQuantity((q) => Math.max(1, q - 1))} disabled={maxSelectable < 1 || quantity <= 1} aria-label="Kurangi jumlah kursi" className="flex h-10 w-10 items-center justify-center rounded-full border border-garis bg-surface text-ink hover:bg-krem-tua disabled:text-ink-soft/50">
              <Minus className="h-4 w-4" aria-hidden />
            </button>
            <output aria-live="polite" className="font-display w-8 text-center text-2xl font-bold text-navy">{Math.min(quantity, Math.max(maxSelectable, 1))}</output>
            <button type="button" onClick={() => setQuantity((q) => Math.min(Math.max(maxSelectable, 1), q + 1))} disabled={maxSelectable < 1 || quantity >= maxSelectable} aria-label="Tambah jumlah kursi" className="flex h-10 w-10 items-center justify-center rounded-full border border-garis bg-surface text-ink hover:bg-krem-tua disabled:text-ink-soft/50">
              <Plus className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </div>

        <div className="mt-6 flex items-end justify-between">
          <p className="text-sm text-ink-soft">Total</p>
          <p className="font-display text-4xl font-extrabold text-navy">{formatRupiah(total)}</p>
        </div>
        <p className="mt-2 flex items-start gap-2 text-xs text-ink-soft">
          <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-ok" aria-hidden />
          Total final dihitung ulang oleh server saat booking dibuat.
        </p>

        <button disabled={submitting || maxSelectable < 1} onClick={createBooking} className={`${btnPrimary} mt-6 w-full py-3 text-base`}>
          {submitting ? 'Membuat booking...' : maxSelectable < 1 ? 'Kursi habis' : 'Konfirmasi dan lanjut ke pembayaran'}
        </button>
      </div>
    </section>
  );
}

export default function BookingPage() {
  return (
    <Suspense fallback={<p className="text-ink-soft">Memuat detail booking...</p>}>
      <BookingContent />
    </Suspense>
  );
}
