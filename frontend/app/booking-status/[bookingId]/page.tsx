'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ChevronDown } from 'lucide-react';
import toast from 'react-hot-toast';
import { bookingApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';
import { useTrip, tripLabel } from '@/lib/trips';
import { formatWaktu } from '@/lib/format';
import { statusOf } from '@/lib/errors';
import { btnGhost, btnPrimary, card } from '@/lib/ui';
import StatusBadge from '@/components/ui/StatusBadge';
import OrderTimeline from '@/components/bookings/OrderTimeline';
import TripArt from '@/components/trips/TripArt';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

/** Status yang tidak akan berubah lagi, jadi tidak perlu di-polling. */
const TERMINAL_STATUSES = new Set(['CONFIRMED', 'CANCELLED', 'EXPIRED']);

export default function BookingStatusPage() {
  const params = useParams<{ bookingId: string }>();
  const router = useRouter();
  const bookingId = typeof params?.bookingId === 'string' ? params.bookingId : '';
  const [booking, setBooking] = useState<Booking | null>(null);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const trip = useTrip(booking?.resource_id ?? '');

  const validId = uuidPattern.test(bookingId);
  const isTerminal = booking ? TERMINAL_STATUSES.has(booking.status) : false;

  useEffect(() => {
    // Id yang bentuknya salah tidak perlu memanggil API sama sekali.
    if (!validId) return;
    // Status final tidak akan berubah lagi; hentikan polling.
    if (isTerminal) return;

    const fetchStatus = async () => {
      try {
        const res = await bookingApi.get(`/bookings/${bookingId}`);
        setBooking(res.data);
        setNotFound(false);
      } catch (error: unknown) {
        if (statusOf(error) === 404) {
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
  }, [bookingId, validId, isTerminal]);

  if (!validId) return <NotFoundCard onBack={() => router.push('/')} />;

  if (loading) return <div role="status" aria-label="Memuat status booking" className="mx-auto h-96 max-w-2xl animate-pulse rounded-3xl bg-krem-tua/60" />;

  if (notFound || !booking) return <NotFoundCard onBack={() => router.push('/')} />;

  const title = tripLabel(trip, booking.resource_id);

  return (
    <section className={`${card} mx-auto max-w-2xl overflow-hidden`}>
      <div className="on-dark relative h-36">
        <TripArt id={booking.resource_id} destination={trip?.destination ?? ''} title={trip?.title ?? ''} className="h-full w-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-dark-bg/95 via-dark-bg/75 to-dark-bg/55" />
        <div className="absolute inset-x-0 bottom-0 flex items-end justify-between gap-3 p-6 text-white">
          <div>
            <p className="text-xs font-bold uppercase tracking-[0.2em] text-amber">Status booking</p>
            <h1 className="mt-1 text-2xl font-extrabold sm:text-3xl">{title}</h1>
          </div>
          <StatusBadge status={booking.status} />
        </div>
      </div>

      <div className="p-6 sm:p-8">
        <dl className="grid grid-cols-2 gap-4 text-sm">
          <div><dt className="text-ink-soft">Jumlah kursi</dt><dd className="font-semibold text-ink">{booking.quantity}</dd></div>
          <div><dt className="text-ink-soft">Total</dt><dd className="font-semibold text-ink">{formatRupiah(booking.total_amount)}</dd></div>
          <div className="col-span-2"><dt className="text-ink-soft">Nomor pesanan</dt><dd className="break-all font-mono text-xs text-ink">{booking.id}</dd></div>
        </dl>

        <h2 className="mb-5 mt-8 text-xl font-bold text-navy">Perjalanan pesanan</h2>
        <OrderTimeline status={booking.status} />

        {booking.status === 'PENDING_PAYMENT' ? (
          <p className="mt-6 rounded-xl bg-warn-bg px-4 py-3 text-sm text-warn">
            Jika kamu baru saja membayar, konfirmasi butuh beberapa detik. Halaman ini menyegarkan sendiri.
            {booking.expires_at ? ` Batas bayar: ${formatWaktu(booking.expires_at)}.` : ''}
          </p>
        ) : null}

        <details className="group mt-8 rounded-xl border border-garis bg-krem/60 p-4">
          <summary className="flex cursor-pointer list-none items-center justify-between text-sm font-semibold text-navy">
            Apa yang terjadi di balik layar
            <ChevronDown className="h-4 w-4 transition group-open:rotate-180" aria-hidden />
          </summary>
          <div className="mt-3 space-y-3 text-sm text-ink-soft">
            <p>
              Setiap layanan TripNest berbicara lewat pesan (event), bukan saling memanggil langsung. Urutan normalnya:
            </p>
            <ol className="list-decimal space-y-1.5 pl-5">
              <li><code className="rounded bg-krem-tua px-1">booking.created</code>: pesananmu dicatat dan menunggu kursi.</li>
              <li><code className="rounded bg-krem-tua px-1">inventory.reserved</code>: katalog menahan kursi, lalu pembayaran disiapkan.</li>
              <li><code className="rounded bg-krem-tua px-1">payment.success</code>: Midtrans mengabarkan pembayaran berhasil, pesanan menjadi terkonfirmasi.</li>
            </ol>
            <p>
              Bila kursi ditolak atau pembayaran gagal, pesanan dibatalkan. Bila batas waktu habis, pesanan kedaluwarsa dan kursi dilepas kembali (<code className="rounded bg-krem-tua px-1">booking.expired</code>).
            </p>
          </div>
        </details>

        <div className="mt-8 flex flex-wrap gap-3">
          {booking.status === 'PENDING_PAYMENT' ? (
            <button onClick={() => router.push(`/payment/${booking.id}`)} className={btnPrimary}>Lanjut bayar</button>
          ) : null}
          <button onClick={() => router.push('/my-bookings')} className={btnGhost}>Kembali ke Pesanan Saya</button>
        </div>
      </div>
    </section>
  );
}

function NotFoundCard({ onBack }: { onBack: () => void }) {
  return (
    <section className={`${card} mx-auto max-w-2xl p-8`}>
      <h1 className="text-2xl font-bold text-navy">Status booking</h1>
      <p className="mt-3 text-sm font-semibold text-bad">Booking tidak ditemukan</p>
      <button onClick={onBack} className={`${btnGhost} mt-5`}>Kembali ke beranda</button>
    </section>
  );
}
