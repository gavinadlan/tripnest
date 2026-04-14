'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';

const API_BOOKING = 'http://localhost:8081';

export default function PaymentPage() {
  const params = useParams();
  const router = useRouter();

  const bookingId =
    typeof params?.bookingId === 'string' ? params.bookingId : '';

  const [booking, setBooking] = useState<any>(null);
  const [snapToken, setSnapToken] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // 🔥 LOAD MIDTRANS SCRIPT
  useEffect(() => {
    if (document.getElementById('midtrans-script')) return;

    const script = document.createElement('script');
    script.id = 'midtrans-script';
    script.src = 'https://app.sandbox.midtrans.com/snap/snap.js';
    script.setAttribute(
      'data-client-key',
      'Mid-client-5fq_PtORQ2LIktjf'
    );
    script.async = true;

    document.body.appendChild(script);
  }, []);

  // 🔥 FETCH BOOKING
  useEffect(() => {
    if (!bookingId) return;

    const fetchBooking = async () => {
      try {
        setLoading(true);
        setError('');

        console.log('BOOKING ID:', bookingId);

        const res = await fetch(
          `${API_BOOKING}/bookings/${bookingId}`
        );

        if (res.status === 404) {
          setError('Booking tidak ditemukan');
          return;
        }

        const json = await res.json();
        console.log('BOOKING RESPONSE:', json);

        // 🔥 FIX: HANDLE nested response
        const data = json.data || json;

        if (!data || !data.id) {
          setError('Booking tidak valid');
          return;
        }

        setBooking(data);
      } catch (err) {
        console.error(err);
        setError('Gagal mengambil booking');
      } finally {
        setLoading(false);
      }
    };

    fetchBooking();
  }, [bookingId]);

  // 🔥 FETCH SNAP TOKEN
  useEffect(() => {
    if (!booking) return;

    const fetchToken = async () => {
      try {
        const res = await fetch(
          `/api/payments/snap-token?booking_id=${booking.id}`
        );

        const json = await res.json();
        console.log('SNAP TOKEN:', json);

        const token = json.token || json.snap_token;

        if (!token) {
          setError('Token pembayaran tidak ditemukan');
          return;
        }

        setSnapToken(token);
      } catch (err) {
        console.error(err);
        setError('Gagal mengambil token pembayaran');
      }
    };

    fetchToken();
  }, [booking]);

  // 🔥 LOADING
  if (!bookingId || loading) {
    return <p className="p-4">Memuat...</p>;
  }

  // 🔥 ERROR
  if (error) {
    return (
      <div className="p-4">
        <h1 className="text-xl font-bold">Pembayaran</h1>
        <p className="text-red-500 mt-2">{error}</p>
        <button
          onClick={() => router.push('/')}
          className="mt-4 bg-gray-200 px-4 py-2 rounded"
        >
          Kembali ke beranda
        </button>
      </div>
    );
  }

  // 🔥 UI NORMAL
  return (
    <div className="p-4">
      <h1 className="text-xl font-bold">Pembayaran</h1>

      <p className="mt-2">Booking ID: {booking.id}</p>
      <p>Resource: {booking.resource_id}</p>
      <p>Total: Rp {booking.total_amount}</p>

      <button
        onClick={() => {
          const snap = (window as any).snap;

          if (!snap) {
            alert('Snap belum siap, refresh dulu');
            return;
          }

          if (!snapToken) {
            alert('Token belum ada');
            return;
          }

          console.log('PAY:', snapToken);

          snap.pay(snapToken, {
            onSuccess: () => {
              alert('Pembayaran berhasil');
              router.push(`/booking-status/${booking.id}`);
            },
            onPending: () => {
              alert('Pembayaran pending');
            },
            onError: () => {
              alert('Pembayaran gagal');
            },
            onClose: () => {
              alert('Pembayaran dibatalkan');
            },
          });
        }}
        className="mt-4 bg-blue-600 text-white px-4 py-2 rounded"
      >
        Bayar Sekarang
      </button>
    </div>
  );
}