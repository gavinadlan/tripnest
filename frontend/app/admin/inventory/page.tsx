'use client';

import { useEffect, useState } from 'react';
import { adminInventoryApi } from '@/lib/adminApi';
import toast from 'react-hot-toast';
import { idPendek } from '@/lib/format';
import { tripLabel, useTrips } from '@/lib/trips';
import { btnGhost, btnSmall, input, tableWrap, td, th } from '@/lib/ui';

type Inventory = {
  resource_id: string;
  total_slots: number;
  available_slots: number;
  reserved_slots: number;
};

export default function AdminInventoryPage() {
  const [items, setItems] = useState<Inventory[]>([]);
  const [editing, setEditing] = useState<Record<string, number>>({});
  const trips = useTrips(items.map((i) => i.resource_id));

  const load = async () => {
    const res = await adminInventoryApi.get('/inventory');
    setItems(res.data.data || res.data || []);
  };

  useEffect(() => {
    void (async () => {
      try {
        await load();
      } catch {
        toast.error('Gagal memuat inventaris');
      }
    })();
  }, []);

  const updateInventory = async (id: string) => {
    try {
      await adminInventoryApi.patch(`/inventory/${id}`, { total_slots: editing[id] });
      toast.success('Inventaris berhasil diperbarui');
      await load();
    } catch {
      toast.error('Gagal memperbarui inventaris');
    }
  };

  return (
    <section className="space-y-4">
      <p className="max-w-2xl text-sm text-ink-soft">
        Kapasitas dibuat bersama trip di halaman Trip. Di sini kamu hanya bisa menyesuaikan total kursi trip yang sudah ada.
      </p>
      <div className={tableWrap}>
        <table className="min-w-full text-sm">
          <thead className="bg-krem/70">
            <tr>
              <th className={th}>Trip</th>
              <th className={th}>Total kursi</th>
              <th className={th}>Terisi</th>
              <th className={th}>Tersedia</th>
              <th className={th}>Dipesan</th>
              <th className={th}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => {
              const key = item.resource_id;
              const used = item.total_slots ? ((item.total_slots - item.available_slots) / item.total_slots) * 100 : 0;
              return (
                <tr key={key} className="border-t border-garis">
                  <td className={`${td} font-semibold text-navy`} title={`ID ${idPendek(key)}`}>{tripLabel(trips[key], key)}</td>
                  <td className={td}>
                    <label className="sr-only" htmlFor={`slot-${key}`}>Total kursi</label>
                    <input
                      id={`slot-${key}`}
                      type="number"
                      min={1}
                      value={editing[key] ?? item.total_slots}
                      onChange={(e) => setEditing((prev) => ({ ...prev, [key]: Number(e.target.value) }))}
                      className={`${input} !w-24 !px-3 !py-1.5`}
                    />
                  </td>
                  <td className={`${td} min-w-32`}>
                    <div className="h-2 rounded-full bg-krem-tua" role="img" aria-label={`${Math.round(used)}% terisi`}>
                      <div className={`h-2 rounded-full ${used >= 90 ? 'bg-bad' : used >= 70 ? 'bg-amber' : 'bg-biru'}`} style={{ width: `${used}%` }} />
                    </div>
                  </td>
                  <td className={td}>{item.available_slots}</td>
                  <td className={td}>{item.reserved_slots}</td>
                  <td className={td}>
                    <button onClick={() => updateInventory(item.resource_id)} className={`${btnGhost} ${btnSmall}`}>Simpan</button>
                  </td>
                </tr>
              );
            })}
            {items.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-10 text-center text-ink-soft">Belum ada data inventaris.</td></tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}
