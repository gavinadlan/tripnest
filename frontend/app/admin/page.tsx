'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminBookingApi, adminPaymentApi } from '@/lib/adminApi';
import StatCard from '@/components/admin/StatCard';
import CountUp from '@/components/ui/CountUp';

type Booking = { status: string };
type Payment = { status: string };

export default function AdminOverviewPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);

  useEffect(() => {
    const load = async () => {
      const [b, p] = await Promise.allSettled([adminBookingApi.get('/bookings'), adminPaymentApi.get('/payments')]);

      if (b.status === 'fulfilled') setBookings(b.value.data.data || b.value.data || []);
      if (p.status === 'fulfilled') setPayments(p.value.data.data || p.value.data || []);
    };
    load();
  }, []);

  const stats = useMemo(() => {
    const total = bookings.length;
    const confirmed = bookings.filter((b) => b.status === 'CONFIRMED').length;
    const failed = bookings.filter((b) => ['CANCELLED', 'EXPIRED'].includes(b.status)).length;
    const active = bookings.filter((b) => b.status === 'PENDING_PAYMENT').length;
    const expired = bookings.filter((b) => b.status === 'EXPIRED').length;

    const payTotal = payments.length || 1;
    const paySuccess = payments.filter((p) => p.status === 'SUCCESS').length;
    const payRate = Math.round((paySuccess / payTotal) * 100);

    return { total, confirmed, failed, active, expired, payRate, payCount: payments.length, paySuccess };
  }, [bookings, payments]);

  const pct = (n: number) => (stats.total ? (n / stats.total) * 100 : 0);

  return (
    <section className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <StatCard  label="Total booking" value={<CountUp value={stats.total} />} hint="Semua booking, apa pun statusnya." />
        <StatCard  tone="success" label="Terkonfirmasi" value={<CountUp value={stats.confirmed} />} hint="Pembayaran diterima dan kursi aman." />
        <StatCard  tone="warning" label="Menunggu pembayaran" value={<CountUp value={stats.active} />} hint="Kursi tertahan, belum dibayar." />
        <StatCard  tone="error" label="Dibatalkan atau kedaluwarsa" value={<CountUp value={stats.failed} />} hint="Gabungan dibatalkan dan kedaluwarsa; kursinya sudah dilepas." />
        <StatCard  tone="error" label="Kedaluwarsa" value={<CountUp value={stats.expired} />} hint="Bagian dari angka di sebelahnya: batas waktu bayar habis." />
        <StatCard
          
          label="Rasio pembayaran berhasil"
          value={<CountUp value={stats.payRate} format={(n) => `${n}%`} />}
          hint={`${stats.paySuccess} berhasil dari ${stats.payCount} catatan pembayaran, termasuk yang masih menunggu.`}
        />
      </div>

      <div className="rounded-2xl border border-garis bg-surface p-6">
        <h2 className="text-xl font-bold text-navy">Komposisi booking</h2>
        <div className="mt-5 space-y-4">
          {[
            ['Terkonfirmasi', stats.confirmed, 'bg-ok'],
            ['Menunggu pembayaran', stats.active, 'bg-amber'],
            ['Dibatalkan atau kedaluwarsa', stats.failed, 'bg-bad'],
          ].map(([label, n, color]) => (
            <div key={label as string}>
              <div className="mb-1 flex items-center justify-between text-sm text-ink-soft">
                <span>{label}</span>
                <span className="font-semibold text-ink">{n as number}</span>
              </div>
              <div className="h-2.5 rounded-full bg-krem-tua">
                <div className={`h-2.5 rounded-full ${color}`} style={{ width: `${pct(n as number)}%` }} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
