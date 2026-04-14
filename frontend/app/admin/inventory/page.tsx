'use client';

import { useEffect, useState } from 'react';
import { adminInventoryApi } from '@/lib/adminApi';
import toast from 'react-hot-toast';

type Inventory = {
  resource_id: string;
  total_slots: number;
  available_slots: number;
  reserved_slots: number;
};

export default function AdminInventoryPage() {
  const [items, setItems] = useState<Inventory[]>([]);
  const [resourceId, setResourceId] = useState('');
  const [totalSlots, setTotalSlots] = useState(0);
  const [editing, setEditing] = useState<Record<string, number>>({});

  useEffect(() => {
    void (async () => {
      try {
        const res = await adminInventoryApi.get('/inventory');
        setItems(res.data.data || res.data || []);
      } catch {
        toast.error('Gagal memuat inventaris');
      }
    })();
  }, []);

  const createInventory = async () => {
    try {
      await adminInventoryApi.post('/inventory', {
        resource_id: resourceId,
        total_slots: totalSlots,
      });
      toast.success('Inventaris berhasil dibuat');
      setResourceId('');
      setTotalSlots(0);
      const res = await adminInventoryApi.get('/inventory');
      setItems(res.data.data || res.data || []);
    } catch {
      toast.error('Gagal membuat inventaris');
    }
  };

  const updateInventory = async (id: string) => {
    try {
      await adminInventoryApi.patch(`/inventory/${id}`, { total_slots: editing[id] });
      toast.success('Inventaris berhasil diperbarui');
      const res = await adminInventoryApi.get('/inventory');
      setItems(res.data.data || res.data || []);
    } catch {
      toast.error('Gagal memperbarui inventaris');
    }
  };

  return (
    <section className="space-y-5">
      <div className="rounded-2xl border border-[var(--color-border)] bg-white p-4 shadow-sm">
        <h2 className="mb-3 text-base font-semibold text-[var(--color-text-primary)]">Buat Inventaris</h2>
        <div className="flex flex-wrap gap-3">
          <input
            placeholder="resource_id trip"
            value={resourceId}
            onChange={(e) => setResourceId(e.target.value)}
            className="rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
          />
          <input
            type="number"
            placeholder="total slot"
            value={totalSlots}
            onChange={(e) => setTotalSlots(Number(e.target.value))}
            className="w-40 rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm"
          />
          <button onClick={createInventory} className="rounded-lg bg-[var(--color-primary)] px-4 py-2 text-sm font-semibold text-white">
            Buat
          </button>
        </div>
      </div>

      <div className="overflow-hidden rounded-2xl border border-[var(--color-border)] bg-white shadow-sm">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-[var(--color-text-secondary)]">
            <tr>
              <th className="px-4 py-3 font-medium">ID Resource</th>
              <th className="px-4 py-3 font-medium">Total Slot</th>
              <th className="px-4 py-3 font-medium">Slot Tersedia</th>
              <th className="px-4 py-3 font-medium">Slot Dipesan</th>
              <th className="px-4 py-3 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const key = item.resource_id;
              return (
                <tr key={key} className="border-t border-[var(--color-border)]">
                  <td className="px-4 py-3">{item.resource_id}</td>
                  <td className="px-4 py-3">
                    <input
                      type="number"
                      value={editing[key] ?? item.total_slots}
                      onChange={(e) => setEditing((prev) => ({ ...prev, [key]: Number(e.target.value) }))}
                      className="w-24 rounded-md border border-[var(--color-border)] px-2 py-1 text-xs"
                    />
                  </td>
                  <td className="px-4 py-3">{item.available_slots}</td>
                  <td className="px-4 py-3">{item.reserved_slots}</td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => updateInventory(item.resource_id)}
                      className="rounded-md border border-[var(--color-border)] px-2 py-1 text-xs"
                    >
                      Simpan
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
