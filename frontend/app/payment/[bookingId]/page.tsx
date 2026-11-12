'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import { bookingApi, paymentApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';
import { useTrip, tripLabel } from '@/lib/trips';
import { formatSisa, useCountdown } from '@/lib/countdown';
import { btnGhost, btnPrimary, card } from '@/lib/ui';
import { Clock, Lock } from 'lucide-react';
import TripArt from '@/components/trips/TripArt';
import MultiStepLoader from '@/components/ui/MultiStepLoader';

const SNAP_SCRIPT_URL =
  process.env.NEXT_PUBLIC_MIDTRANS_SNAP_URL || 'https://app.sandbox.midtrans.com/snap/snap.js';
const SNAP_CLIENT_KEY = process.env.NEXT_PUBLIC_MIDTRANS_CLIENT_KEY || '';

// Dihitung sekali di level modul, bukan lewat setState di dalam effect.
const CONFIG_ERROR = SNAP_CLIENT_KEY ? '' : 'NEXT_PUBLIC_MIDTRANS_CLIENT_KEY belum diatur';

// Booking lahir sebagai PENDING_SEAT dan baru bisa dibayar setelah
// catalog-service menahan kursinya. Normalnya beberapa detik; batas di
// bawah ini sedikit lebih longgar daripada sapuan booking nyangkut di
// server, supaya halaman tidak menyerah lebih dulu daripada servernya.
const SEAT_POLL_INTERVAL_MS = 2000;
const SEAT_POLL_TIMEOUT_MS = 150000;

type SnapResult = { transaction_status?: string };
type SnapApi = {
  pay: (
    token: string,
    callbacks: {
      onSuccess?: (result: SnapResult) => void;
      onPending?: (result: SnapResult) => void;
      onError?: (result: SnapResult) => void;
      onClose?: () => void;
    }
  ) => void;
};

export default function PaymentPage() {
  const params = useParams<{ bookingId: string }>();
  const router = useRouter();
  const bookingId = typeof params?.bookingId === 'string' ? params.bookingId : '';

  const [booking, setBooking] = useState<Booking | null>(null);
  const [snapToken, setSnapToken] = useState('');
  const [loading, setLoading] = useState(true);
  const [waitingSeat, setWaitingSeat] = useState(false);
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState('');
  const trip = useTrip(booking?.resource_id ?? '');
  const sisa = useCountdown(booking?.expires_at);

  // Muat skrip Snap sekali, dan tandai kapan benar-benar siap dipakai.
  useEffect(() => {
    if (CONFIG_ERROR) return;

    // Kesiapan Snap dicek saat tombol ditekan lewat window.snap, bukan
    // disimpan di state. Skrip ini hidup di luar React dan bisa sudah
    // termuat dari kunjungan sebelumnya.
    if (document.getElementById('midtrans-script')) return;

    const script = document.createElement('script');
    script.id = 'midtrans-script';
    script.src = SNAP_SCRIPT_URL;
    script.setAttribute('data-client-key', SNAP_CLIENT_KEY);
    script.async = true;
    script.onerror = () => setError('Gagal memuat halaman pembayaran Midtrans');
    document.body.appendChild(script);
  }, []);

  useEffect(() => {
    if (!bookingId) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const deadline = Date.now() + SEAT_POLL_TIMEOUT_MS;

    const preparePayment = async (data: Booking) => {
      // GET hanya membaca. Kalau transaksi Snap belum pernah dibuat,
      // barulah kita memintanya lewat POST -- endpoint GET tidak boleh
      // punya efek samping.
      try {
        const tokenRes = await paymentApi.get('/payments/snap-token', {
          params: { booking_id: data.id },
        });
        if (!cancelled) setSnapToken(tokenRes.data?.snap_token ?? tokenRes.data?.token ?? '');
      } catch {
        const initRes = await paymentApi.post('/payments/snap-token', { booking_id: data.id });
        if (!cancelled) setSnapToken(initRes.data?.snap_token ?? initRes.data?.token ?? '');
      }
    };

    const poll = async () => {
      try {
        const bookingRes = await bookingApi.get(`/bookings/${bookingId}`);
        if (cancelled) return;

        const data: Booking = bookingRes.data;
        setBooking(data);
        setLoading(false);

        // Kursi belum tertahan. Belum ada transaksi Snap yang bisa
        // diminta -- payment-service sendiri baru bergerak setelah
        // inventory.reserved terbit -- jadi tunggu, jangan bertanya.
        if (data.status === 'PENDING_SEAT') {
          if (Date.now() > deadline) {
            setWaitingSeat(false);
            setError('Kursi tidak kunjung dipastikan. Coba buka lagi pesananmu sebentar lagi.');
            return;
          }
          setWaitingSeat(true);
          timer = setTimeout(() => void poll(), SEAT_POLL_INTERVAL_MS);
          return;
        }

        setWaitingSeat(false);

        if (data.status !== 'PENDING_PAYMENT') {
          setError(`Booking ini berstatus ${data.status} dan tidak bisa dibayar`);
          return;
        }

        await preparePayment(data);
      } catch (err: unknown) {
        if (cancelled) return;
        const status =
          typeof err === 'object' && err !== null && 'response' in err
            ? (err as { response?: { status?: number } }).response?.status
            : undefined;
        setWaitingSeat(false);
        setLoading(false);
        setError(status === 404 ? 'Booking tidak ditemukan' : 'Gagal menyiapkan pembayaran');
      }
    };

    void poll();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, [bookingId]);

  const pay = useCallback(() => {
    const snap = (window as unknown as { snap?: SnapApi }).snap;
    if (!snap) {
      toast('Halaman pembayaran masih dimuat, coba lagi sebentar');
      return;
    }
    if (!snapToken) return;

    setPaying(true);
    snap.pay(snapToken, {
      onSuccess: () => {
        // Konfirmasi belum tentu selesai saat ini: webhook Midtrans,
        // outbox, dan Kafka butuh beberapa detik. Halaman status yang
        // menunggu perubahannya.
        toast.success('Pembayaran diterima, sedang dikonfirmasi');
        router.push(`/booking-status/${bookingId}`);
      },
      onPending: () => {
        toast('Pembayaran menunggu penyelesaian');
        router.push(`/booking-status/${bookingId}`);
      },
      onError: () => {
        toast.error('Pembayaran gagal');
        setPaying(false);
      },
      onClose: () => {
        toast('Pembayaran dibatalkan. Kamu bisa mencoba lagi sebelum booking kedaluwarsa.');
        setPaying(false);
      },
    });
  }, [snapToken, bookingId, router]);

  if (loading) {
    return <div role="status" aria-label="Memuat" className="mx-auto h-80 max-w-2xl animate-pulse rounded-3xl bg-krem-tua/60" />;
  }

  const displayError = CONFIG_ERROR || error;

  if (displayError || !booking) {
    return (
      <section className={`${card} mx-auto max-w-2xl p-8`}>
        <h1 className="text-2xl font-bold text-navy">Pembayaran</h1>
        <p className="mt-3 text-sm font-semibold text-bad">{displayError || 'Booking tidak ditemukan'}</p>
        <button onClick={() => router.push('/my-bookings')} className={`${btnGhost} mt-5`}>Ke Pesanan Saya</button>
      </section>
    );
  }

  const canPay = Boolean(snapToken) && !paying && !waitingSeat;

  const buttonLabel = paying
    ? 'Memproses...'
    : waitingSeat
      ? 'Menyiapkan kursi...'
      : !snapToken
        ? 'Menyiapkan pembayaran...'
        : 'Bayar sekarang';

  const preparing = waitingSeat || !snapToken;
  const title = tripLabel(trip, booking.resource_id);

  return (
    <section className={`${card} mx-auto max-w-2xl overflow-hidden`}>
      <div className="on-dark relative h-36">
        <TripArt id={booking.resource_id} destination={trip?.destination ?? ''} title={trip?.title ?? ''} className="h-full w-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-dark-bg/95 via-dark-bg/75 to-dark-bg/55" />
        <div className="absolute inset-x-0 bottom-0 p-6 text-white">
          <p className="text-xs font-bold uppercase tracking-[0.2em] text-amber">Pembayaran</p>
          <h1 className="mt-1 text-2xl font-extrabold sm:text-3xl">{title}</h1>
        </div>
      </div>

      <div className="p-6 sm:p-8">
        {preparing ? (
          <div className="mb-6 rounded-xl bg-krem p-4">
            <MultiStepLoader steps={['Pesanan dibuat', 'Menahan kursi di katalog', 'Menyiapkan pembayaran']} active={waitingSeat ? 1 : 2} />
            {waitingSeat ? (
              <p className="mt-3 text-xs text-ink-soft">
                Pembayaran baru dibuka setelah kursi benar-benar tertahan, jadi kamu tidak membayar untuk kursi yang sudah diambil orang lain.
              </p>
            ) : null}
          </div>
        ) : null}

        <dl className="space-y-3 text-sm">
          <div className="flex justify-between gap-4"><dt className="text-ink-soft">Jumlah kursi</dt><dd className="font-semibold text-ink">{booking.quantity}</dd></div>
          <div className="flex items-end justify-between gap-4 border-t border-garis pt-3"><dt className="text-ink-soft">Total</dt><dd className="font-display text-3xl font-extrabold text-navy">{formatRupiah(booking.total_amount)}</dd></div>
        </dl>

        {booking.expires_at && !waitingSeat ? (
          <p className="mt-4 flex items-center gap-2 rounded-xl bg-warn-bg px-4 py-3 text-sm font-semibold text-warn">
            <Clock className="h-4 w-4" aria-hidden />
            {sisa === null ? 'Hitung mundur...' : sisa > 0 ? `Bayar dalam ${formatSisa(sisa)}` : 'Batas waktu pembayaran habis'}
          </p>
        ) : null}

        <button onClick={pay} disabled={!canPay} className={`${btnPrimary} mt-6 w-full py-3 text-base`}>
          <Lock className="h-4 w-4" aria-hidden /> {buttonLabel}
        </button>
        <p className="mt-3 text-center text-xs text-ink-soft">Pembayaran diproses aman lewat Midtrans.</p>
      </div>
    </section>
  );
}
