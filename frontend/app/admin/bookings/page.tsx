'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminBookingApi } from '@/lib/adminApi';
import StatusBadge, { STATUS_LABEL } from '@/components/ui/StatusBadge';
import AdminModal from '@/components/admin/AdminModal';
import TripArt from '@/components/trips/TripArt';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';
import { formatWaktu, idPendek } from '@/lib/format';
import { tripLabel, useTrips } from '@/lib/trips';
import { btnDanger, btnGhost, btnSmall, select, tableWrap, td, th, input } from '@/lib/ui';

type Booking = {
  id: string;
  user_id: string;
  resource_id: string;
  quantity?: number;
  status: string;
  total_amount: number;
  created_at: string;
  expires_at?: string;
};

const statuses = ['ALL', 'PENDING_SEAT', 'PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED'];

export default function AdminBookingsPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [dateFilter, setDateFilter] = useState('');
  const [selected, setSelected] = useState<Booking | null>(null);
  const trips = useTrips(bookings.map((b) => b.resource_id));

  useEffect(() => {
    void (async () => {
      try {
        const res = await adminBookingApi.get('/bookings');
        setBookings(res.data.data || res.data || []);
      } catch {
        toast.error('Gagal memuat booking');
      }
    })();
  }, []);

  const filtered = useMemo(
    () =>
      bookings.filter((b) => {
        const statusOk = statusFilter === 'ALL' || b.status === statusFilter;
        const dateOk = !dateFilter || b.created_at.slice(0, 10) === dateFilter;
        return statusOk && dateOk;
      }),
    [bookings, statusFilter, dateFilter]
  );

  const cancelBooking = async (id: string) => {
    try {
      await adminBookingApi.post(`/bookings/${id}/cancel`);
      toast.success('Booking berhasil dibatalkan');
      const res = await adminBookingApi.get('/bookings');
      setBookings(res.data.data || res.data || []);
    } catch {
      toast.error('Gagal membatalkan booking');
    }
  };

  const sel = selected ? trips[selected.resource_id] : undefined;

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap gap-3">
        <label className="sr-only" htmlFor="f-status">Filter status</label>
        <select id="f-status" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className={select}>
          {statuses.map((s) => <option key={s} value={s}>{s === 'ALL' ? 'Semua status' : STATUS_LABEL[s]}</option>)}
        </select>
        <label className="sr-only" htmlFor="f-date">Filter tanggal dibuat</label>
        <input id="f-date" type="date" value={dateFilter} onChange={(e) => setDateFilter(e.target.value)} className={`${input} !w-auto`} />
      </div>

      <div className={tableWrap}>
        <table className="min-w-full text-sm">
          <thead className="bg-krem/70">
            <tr>
              <th className={th}>Trip</th>
              <th className={th}>Booking</th>
              <th className={th}>Total</th>
              <th className={th}>Status</th>
              <th className={th}>Dibuat</th>
              <th className={th}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((b) => (
              <tr key={b.id} className="border-t border-garis">
                <td className={`${td} font-semibold text-navy`}>{tripLabel(trips[b.resource_id], b.resource_id)}</td>
                <td className={`${td} font-mono text-xs text-ink-soft`} title={b.id}>{idPendek(b.id)}</td>
                <td className={`${td} whitespace-nowrap`}>{formatRupiah(b.total_amount)}</td>
                <td className={td}><StatusBadge status={b.status} /></td>
                <td className={`${td} whitespace-nowrap text-ink-soft`}>{formatWaktu(b.created_at)}</td>
                <td className={td}>
                  <div className="flex gap-2">
                    <button onClick={() => setSelected(b)} className={`${btnGhost} ${btnSmall}`}>Detail</button>
                    <button onClick={() => cancelBooking(b.id)} disabled={b.status !== 'PENDING_PAYMENT'} className={`${btnDanger} ${btnSmall}`}>Batalkan</button>
                  </div>
                </td>
              </tr>
            ))}
            {filtered.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-10 text-center text-ink-soft">Tidak ada booking yang cocok.</td></tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <AdminModal open={!!selected} title="Detail booking" onClose={() => setSelected(null)}>
        {selected ? (
          <div>
            <div className="relative mb-4 h-24 overflow-hidden rounded-xl">
              <TripArt id={selected.resource_id} destination={sel?.destination ?? ''} title={sel?.title ?? ''} className="h-full w-full" />
            </div>
            <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
              <div><dt className="text-ink-soft">Trip</dt><dd className="font-semibold text-ink">{tripLabel(sel, selected.resource_id)}</dd></div>
              <div><dt className="text-ink-soft">Status</dt><dd><StatusBadge status={selected.status} /></dd></div>
              <div><dt className="text-ink-soft">Total</dt><dd className="font-semibold text-ink">{formatRupiah(selected.total_amount)}</dd></div>
              <div><dt className="text-ink-soft">Kedaluwarsa</dt><dd className="font-semibold text-ink">{formatWaktu(selected.expires_at)}</dd></div>
              <div className="sm:col-span-2"><dt className="text-ink-soft">ID booking</dt><dd className="break-all font-mono text-xs">{selected.id}</dd></div>
              <div className="sm:col-span-2"><dt className="text-ink-soft">ID pengguna</dt><dd className="break-all font-mono text-xs">{selected.user_id}</dd></div>
            </dl>
          </div>
        ) : null}
      </AdminModal>
    </section>
  );
}
