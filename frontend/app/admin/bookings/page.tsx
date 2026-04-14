'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminBookingApi } from '@/lib/adminApi';
import StatusBadge from '@/components/ui/StatusBadge';
import AdminModal from '@/components/admin/AdminModal';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

type Booking = {
  id: string;
  user_id: string;
  resource_id: string;
  status: string;
  total_amount: number;
  created_at: string;
  expires_at?: string;
};

const statuses = ['ALL', 'PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED'];

export default function AdminBookingsPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [dateFilter, setDateFilter] = useState('');
  const [selected, setSelected] = useState<Booking | null>(null);

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

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap gap-3">
        <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-white px-3 py-2 text-sm">
          {statuses.map((s) => <option key={s} value={s}>{s}</option>)}
        </select>
        <input type="date" value={dateFilter} onChange={(e) => setDateFilter(e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-white px-3 py-2 text-sm" />
      </div>

      <div className="overflow-hidden rounded-2xl border border-[var(--color-border)] bg-white shadow-sm">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-[var(--color-text-secondary)]">
            <tr>
              <th className="px-4 py-3 font-medium">ID Booking</th>
              <th className="px-4 py-3 font-medium">Trip</th>
              <th className="px-4 py-3 font-medium">Total</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Tanggal</th>
              <th className="px-4 py-3 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((b) => (
              <tr key={b.id} className="border-t border-[var(--color-border)]">
                <td className="px-4 py-3 font-mono text-xs">{b.id}</td>
                <td className="px-4 py-3">{b.resource_id}</td>
                <td className="px-4 py-3">{formatRupiah(b.total_amount)}</td>
                <td className="px-4 py-3"><StatusBadge status={b.status} /></td>
                <td className="px-4 py-3">{new Date(b.created_at).toLocaleString('id-ID')}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-2">
                    <button onClick={() => setSelected(b)} className="rounded-md border border-[var(--color-border)] px-2 py-1 text-xs">Detail</button>
                    <button
                      onClick={() => cancelBooking(b.id)}
                      disabled={b.status !== 'PENDING_PAYMENT'}
                      className="rounded-md bg-[var(--color-error)] px-2 py-1 text-xs text-white disabled:cursor-not-allowed disabled:bg-slate-300"
                    >
                      Batalkan
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <AdminModal open={!!selected} title="Detail booking" onClose={() => setSelected(null)}>
        {selected ? (
          <div className="grid gap-2 text-sm text-[var(--color-text-secondary)]">
            <p><span className="font-semibold text-[var(--color-text-primary)]">ID:</span> {selected.id}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">ID User:</span> {selected.user_id}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Trip:</span> {selected.resource_id}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Status:</span> {selected.status}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Total:</span> {formatRupiah(selected.total_amount)}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Kedaluwarsa:</span> {selected.expires_at || '-'}</p>
          </div>
        ) : null}
      </AdminModal>
    </section>
  );
}
