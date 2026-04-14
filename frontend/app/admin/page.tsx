'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminBookingApi, adminPaymentApi } from '@/lib/adminApi';
import StatCard from '@/components/admin/StatCard';

type Booking = { status: string };
type Payment = { status: string };

export default function AdminOverviewPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);

  useEffect(() => {
    const load = async () => {
      const [b, p] = await Promise.allSettled([
        adminBookingApi.get('/bookings'),
        adminPaymentApi.get('/payments'),
      ]);

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

    return { total, confirmed, failed, active, expired, payRate };
  }, [bookings, payments]);

  return (
    <section className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <StatCard label="Total Booking" value={stats.total} />
        <StatCard label="Booking Terkonfirmasi" value={stats.confirmed} tone="success" />
        <StatCard label="Gagal / Dibatalkan" value={stats.failed} tone="error" />
        <StatCard label="Rasio Pembayaran Sukses" value={`${stats.payRate}%`} tone="warning" />
        <StatCard label="Booking Menunggu" value={stats.active} />
        <StatCard label="Booking Kedaluwarsa" value={stats.expired} tone="error" />
      </div>

      <div className="rounded-2xl border border-[var(--color-border)] bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Kesehatan Booking</h2>
        <div className="mt-5 space-y-3">
          <div>
            <div className="mb-1 flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
              <span>Terkonfirmasi</span>
              <span>{stats.confirmed}</span>
            </div>
            <div className="h-2 rounded-full bg-slate-100">
              <div className="h-2 rounded-full bg-[var(--color-success)]" style={{ width: `${stats.total ? (stats.confirmed / stats.total) * 100 : 0}%` }} />
            </div>
          </div>
          <div>
            <div className="mb-1 flex items-center justify-between text-sm text-[var(--color-text-secondary)]">
              <span>Gagal / Dibatalkan</span>
              <span>{stats.failed}</span>
            </div>
            <div className="h-2 rounded-full bg-slate-100">
              <div className="h-2 rounded-full bg-[var(--color-error)]" style={{ width: `${stats.total ? (stats.failed / stats.total) * 100 : 0}%` }} />
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
