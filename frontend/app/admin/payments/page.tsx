'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminBookingApi, adminPaymentApi } from '@/lib/adminApi';
import StatusBadge, { STATUS_LABEL } from '@/components/ui/StatusBadge';
import AdminModal from '@/components/admin/AdminModal';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';
import { idPendek } from '@/lib/format';
import { tripLabel, useTrips } from '@/lib/trips';
import { btnGhost, btnSmall, select, tableWrap, td, th } from '@/lib/ui';

type Payment = {
  id: string;
  booking_id: string;
  amount: number;
  status: string;
  payment_type?: string;
  transaction_status?: string;
  transaction_id?: string;
  created_at: string;
};

const statuses = ['ALL', 'SUCCESS', 'FAILED', 'PENDING'];

export default function AdminPaymentsPage() {
  const [payments, setPayments] = useState<Payment[]>([]);
  // Pembayaran hanya mengenal booking_id; trip-nya dicari lewat daftar booking.
  const [tripByBooking, setTripByBooking] = useState<Record<string, string>>({});
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [selected, setSelected] = useState<Payment | null>(null);
  const trips = useTrips(Object.values(tripByBooking));

  useEffect(() => {
    const load = async () => {
      const [p, b] = await Promise.allSettled([adminPaymentApi.get('/payments'), adminBookingApi.get('/bookings')]);
      if (p.status === 'fulfilled') setPayments(p.value.data.data || p.value.data || []);
      else toast.error('Gagal memuat pembayaran');
      if (b.status === 'fulfilled') {
        const rows: { id: string; resource_id: string }[] = b.value.data.data || b.value.data || [];
        setTripByBooking(Object.fromEntries(rows.map((r) => [r.id, r.resource_id])));
      }
    };
    load();
  }, []);

  const filtered = useMemo(() => payments.filter((p) => statusFilter === 'ALL' || p.status === statusFilter), [payments, statusFilter]);
  const label = (p: Payment) => {
    const rid = tripByBooking[p.booking_id];
    return rid ? tripLabel(trips[rid], rid) : `Booking ${idPendek(p.booking_id)}`;
  };

  return (
    <section className="space-y-4">
      <label className="sr-only" htmlFor="f-status">Filter status</label>
      <select id="f-status" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className={select}>
        {statuses.map((s) => <option key={s} value={s}>{s === 'ALL' ? 'Semua status' : STATUS_LABEL[s]}</option>)}
      </select>

      <div className={tableWrap}>
        <table className="min-w-full text-sm">
          <thead className="bg-krem/70">
            <tr>
              <th className={th}>Trip</th>
              <th className={th}>Pembayaran</th>
              <th className={th}>Total</th>
              <th className={th}>Status</th>
              <th className={th}>Metode</th>
              <th className={th}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((p) => (
              <tr key={p.id} className="border-t border-garis">
                <td className={`${td} font-semibold text-navy`} title={`Booking ${p.booking_id}`}>{label(p)}</td>
                <td className={`${td} font-mono text-xs text-ink-soft`} title={p.id}>{idPendek(p.id)}</td>
                <td className={`${td} whitespace-nowrap`}>{formatRupiah(p.amount)}</td>
                <td className={td}><StatusBadge status={p.status} /></td>
                <td className={td}>{p.payment_type ? p.payment_type.replaceAll('_', ' ') : '-'}</td>
                <td className={td}><button onClick={() => setSelected(p)} className={`${btnGhost} ${btnSmall}`}>Detail</button></td>
              </tr>
            ))}
            {filtered.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-10 text-center text-ink-soft">Tidak ada pembayaran yang cocok.</td></tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <AdminModal open={!!selected} title="Detail pembayaran" onClose={() => setSelected(null)}>
        {selected ? (
          <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
            <div><dt className="text-ink-soft">Trip</dt><dd className="font-semibold text-ink">{label(selected)}</dd></div>
            <div><dt className="text-ink-soft">Status</dt><dd><StatusBadge status={selected.status} /></dd></div>
            <div><dt className="text-ink-soft">Total</dt><dd className="font-semibold text-ink">{formatRupiah(selected.amount)}</dd></div>
            <div><dt className="text-ink-soft">Metode</dt><dd className="font-semibold text-ink">{selected.payment_type || '-'}</dd></div>
            <div><dt className="text-ink-soft">Status transaksi</dt><dd className="font-semibold text-ink">{selected.transaction_status || '-'}</dd></div>
            <div><dt className="text-ink-soft">ID transaksi</dt><dd className="break-all font-mono text-xs">{selected.transaction_id || '-'}</dd></div>
            <div className="sm:col-span-2"><dt className="text-ink-soft">ID pembayaran</dt><dd className="break-all font-mono text-xs">{selected.id}</dd></div>
            <div className="sm:col-span-2"><dt className="text-ink-soft">ID booking</dt><dd className="break-all font-mono text-xs">{selected.booking_id}</dd></div>
          </dl>
        ) : null}
      </AdminModal>
    </section>
  );
}
