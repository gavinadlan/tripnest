'use client';

import { useEffect, useMemo, useState } from 'react';
import { adminPaymentApi } from '@/lib/adminApi';
import StatusBadge from '@/components/ui/StatusBadge';
import AdminModal from '@/components/admin/AdminModal';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

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
  const [statusFilter, setStatusFilter] = useState('ALL');
  const [selected, setSelected] = useState<Payment | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const res = await adminPaymentApi.get('/payments');
        setPayments(res.data.data || res.data || []);
      } catch {
        toast.error('Gagal memuat pembayaran');
      }
    };
    load();
  }, []);

  const filtered = useMemo(
    () => payments.filter((p) => statusFilter === 'ALL' || p.status === statusFilter),
    [payments, statusFilter]
  );

  return (
    <section className="space-y-4">
      <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="rounded-lg border border-[var(--color-border)] bg-white px-3 py-2 text-sm">
        {statuses.map((s) => <option key={s} value={s}>{s}</option>)}
      </select>

      <div className="overflow-hidden rounded-2xl border border-[var(--color-border)] bg-white shadow-sm">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-[var(--color-text-secondary)]">
            <tr>
              <th className="px-4 py-3 font-medium">ID Pembayaran</th>
              <th className="px-4 py-3 font-medium">ID Booking</th>
              <th className="px-4 py-3 font-medium">Total</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Metode Bayar</th>
              <th className="px-4 py-3 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((p) => (
              <tr key={p.id} className="border-t border-[var(--color-border)]">
                <td className="px-4 py-3 font-mono text-xs">{p.id}</td>
                <td className="px-4 py-3 font-mono text-xs">{p.booking_id}</td>
                <td className="px-4 py-3">{formatRupiah(p.amount)}</td>
                <td className="px-4 py-3"><StatusBadge status={p.status} /></td>
                <td className="px-4 py-3">{p.payment_type || '-'}</td>
                <td className="px-4 py-3">
                  <button onClick={() => setSelected(p)} className="rounded-md border border-[var(--color-border)] px-2 py-1 text-xs">
                    Detail
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <AdminModal open={!!selected} title="Detail pembayaran" onClose={() => setSelected(null)}>
        {selected ? (
          <div className="grid gap-2 text-sm text-[var(--color-text-secondary)]">
            <p><span className="font-semibold text-[var(--color-text-primary)]">ID Pembayaran:</span> {selected.id}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">ID Booking:</span> {selected.booking_id}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Total:</span> {formatRupiah(selected.amount)}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Status:</span> {selected.status}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Metode:</span> {selected.payment_type || '-'}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">Status Transaksi:</span> {selected.transaction_status || '-'}</p>
            <p><span className="font-semibold text-[var(--color-text-primary)]">ID Transaksi:</span> {selected.transaction_id || '-'}</p>
          </div>
        ) : null}
      </AdminModal>
    </section>
  );
}
