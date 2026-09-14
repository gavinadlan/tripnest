'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import { bookingApi, paymentApi } from '@/lib/api';
import { Booking } from '@/lib/types';
import { formatRupiah } from '@/lib/currency';

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
    return <p className="text-[var(--color-text-secondary)]">Memuat...</p>;
  }

  const displayError = CONFIG_ERROR || error;

  if (displayError || !booking) {
    return (
      <section className="mx-auto max-w-2xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
        <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Pembayaran</h1>
        <p className="mt-3 text-sm text-[var(--color-error)]">{displayError || 'Booking tidak ditemukan'}</p>
        <button
          onClick={() => router.push('/my-bookings')}
          className="mt-5 rounded-xl border border-[var(--color-border)] px-4 py-2 text-sm font-medium"
        >
          Ke Pesanan Saya
        </button>
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

  return (
    <section className="mx-auto max-w-2xl rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-sm">
      <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Pembayaran</h1>
      <p className="mt-2 font-mono text-xs text-[var(--color-text-secondary)]">{booking.id}</p>

      {waitingSeat ? (
        <div className="mt-5 flex items-start gap-3 rounded-xl border border-[var(--color-border)] bg-[var(--color-muted)] p-4">
          <span
            aria-hidden
            className="mt-0.5 h-4 w-4 animate-spin rounded-full border-2 border-[var(--color-primary)] border-t-transparent"
          />
          <div className="text-sm">
            <p className="font-medium text-[var(--color-text-primary)]">Menyiapkan kursi...</p>
            <p className="mt-1 text-[var(--color-text-secondary)]">
              Kami sedang memastikan kursimu masih tersedia. Pembayaran baru dibuka setelah kursi
              benar-benar tertahan, jadi kamu tidak akan membayar untuk kursi yang sudah diambil
              orang lain.
            </p>
          </div>
        </div>
      ) : null}

      <div className="mt-5 space-y-1 text-sm text-[var(--color-text-secondary)]">
        <p>Trip: {booking.resource_id}</p>
        <p>Jumlah kursi: {booking.quantity}</p>
        <p className="text-lg font-semibold text-[var(--color-text-primary)]">
          Total: {formatRupiah(booking.total_amount)}
        </p>
        {booking.expires_at && !waitingSeat ? (
          <p className="text-xs">
            Bayar sebelum {new Date(booking.expires_at).toLocaleString('id-ID')}
          </p>
        ) : null}
      </div>

      <button
        onClick={pay}
        disabled={!canPay}
        className="mt-6 rounded-xl bg-[var(--color-primary)] px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-slate-300"
      >
        {buttonLabel}
      </button>
    </section>
  );
}
