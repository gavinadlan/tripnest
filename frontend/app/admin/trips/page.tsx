'use client';

import { useEffect, useState } from 'react';
import { adminTripApi } from '@/lib/adminApi';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';

type Trip = {
  id: string;
  title: string;
  destination: string;
  price: number;
  date: string;
  available_slots: number;
};

type TripForm = Omit<Trip, 'id'>;

const initialForm: TripForm = {
  title: '',
  destination: '',
  price: 0,
  date: '',
  available_slots: 0,
};

export default function AdminTripsPage() {
  const [trips, setTrips] = useState<Trip[]>([]);
  const [form, setForm] = useState<TripForm>(initialForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const loadTrips = async () => {
    setLoading(true);
    try {
      const res = await adminTripApi.get('/trips');
      setTrips(res.data.data || []);
    } catch {
      toast.error('Gagal memuat data trip');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadTrips();
  }, []);

  const resetForm = () => {
    setForm(initialForm);
    setEditingId(null);
  };

  const submitTrip = async () => {
    if (!form.title || !form.destination || !form.date || form.price <= 0 || form.available_slots <= 0) {
      toast.error('Lengkapi data. Harga dan slot harus lebih dari 0');
      return;
    }

    setLoading(true);
    try {
      if (editingId) {
        await adminTripApi.put(`/trips/${editingId}`, form);
        toast.success('Trip berhasil diperbarui');
      } else {
        await adminTripApi.post('/trips', form);
        toast.success('Trip berhasil dibuat');
      }
      resetForm();
      await loadTrips();
    } catch {
      toast.error(editingId ? 'Gagal memperbarui trip' : 'Gagal membuat trip');
    } finally {
      setLoading(false);
    }
  };

  const deleteTrip = async (id: string) => {
    setLoading(true);
    try {
      await adminTripApi.delete(`/trips/${id}`);
      toast.success('Trip berhasil dihapus');
      await loadTrips();
    } catch {
      toast.error('Gagal menghapus trip');
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-5">
      <div className="rounded-2xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
        <h2 className="mb-1 text-base font-semibold text-[var(--color-text-primary)]">
          {editingId ? 'Ubah Trip' : 'Buat Trip Baru'}
        </h2>
        <p className="mb-4 text-sm text-[var(--color-text-secondary)]">Lengkapi detail trip agar mudah dipahami pengguna.</p>
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Nama Trip</label>
            <input className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm" placeholder="Contoh: Liburan Bali 3 Hari" value={form.title} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Lokasi</label>
            <input className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm" placeholder="Contoh: Bali" value={form.destination} onChange={(e) => setForm((p) => ({ ...p, destination: e.target.value }))} />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Tanggal Keberangkatan</label>
            <input type="date" className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm" value={form.date} onChange={(e) => setForm((p) => ({ ...p, date: e.target.value }))} />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Harga (Rupiah)</label>
            <input type="number" min={1} className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm" placeholder="Contoh: 1500000" value={form.price} onChange={(e) => setForm((p) => ({ ...p, price: Number(e.target.value) }))} />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text-primary)]">Total Slot</label>
            <input type="number" min={1} className="w-full rounded-lg border border-[var(--color-border)] px-3 py-2 text-sm" placeholder="Contoh: 20" value={form.available_slots} onChange={(e) => setForm((p) => ({ ...p, available_slots: Number(e.target.value) }))} />
          </div>
          <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-muted)] p-3 text-sm text-[var(--color-text-secondary)]">
            <p className="font-medium text-[var(--color-text-primary)]">Preview</p>
            <p className="mt-1">Harga per kursi: {formatRupiah(form.price)}</p>
            <p className="mt-1">Total potensi pendapatan: {formatRupiah(form.price * form.available_slots)}</p>
          </div>
          <div className="flex gap-2 md:col-span-2">
            <button onClick={submitTrip} disabled={loading} className="rounded-lg bg-[var(--color-primary)] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50">
              {loading ? 'Menyimpan...' : editingId ? 'Simpan Perubahan' : 'Buat Trip'}
            </button>
            {editingId ? (
              <button onClick={resetForm} className="rounded-lg border border-[var(--color-border)] px-4 py-2 text-sm">
                Batal Ubah
              </button>
            ) : null}
          </div>
        </div>
      </div>

      <div className="overflow-hidden rounded-2xl border border-[var(--color-border)] bg-white shadow-sm">
        <table className="min-w-full text-sm">
          <thead className="bg-slate-50 text-left text-[var(--color-text-secondary)]">
            <tr>
              <th className="px-4 py-3 font-medium">Nama Trip</th>
              <th className="px-4 py-3 font-medium">Lokasi</th>
              <th className="px-4 py-3 font-medium">Tanggal</th>
              <th className="px-4 py-3 font-medium">Harga</th>
              <th className="px-4 py-3 font-medium">Slot</th>
              <th className="px-4 py-3 font-medium">Aksi</th>
            </tr>
          </thead>
          <tbody>
            {trips.map((trip) => (
              <tr key={trip.id} className="border-t border-[var(--color-border)]">
                <td className="px-4 py-3">{trip.title}</td>
                <td className="px-4 py-3">{trip.destination}</td>
                <td className="px-4 py-3">{trip.date}</td>
                <td className="px-4 py-3">{formatRupiah(trip.price)}</td>
                <td className="px-4 py-3">{trip.available_slots}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-2">
                    <button
                      onClick={() => {
                        setEditingId(trip.id);
                        setForm({
                          title: trip.title,
                          destination: trip.destination,
                          date: trip.date,
                          price: trip.price,
                          available_slots: trip.available_slots,
                        });
                      }}
                      className="rounded-md border border-[var(--color-border)] px-2 py-1 text-xs"
                    >
                      Ubah
                    </button>
                    <button onClick={() => deleteTrip(trip.id)} className="rounded-md bg-[var(--color-error)] px-2 py-1 text-xs text-white">
                      Hapus
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!loading && trips.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-6 text-center text-[var(--color-text-secondary)]">
                  Tidak ada trip
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}
