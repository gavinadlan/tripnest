'use client';

import { useEffect, useState } from 'react';
import { adminTripApi } from '@/lib/adminApi';
import toast from 'react-hot-toast';
import { formatRupiah } from '@/lib/currency';
import { formatTanggal } from '@/lib/format';
import { btnDanger, btnGhost, btnPrimary, btnSmall, card, input, tableWrap, td, th } from '@/lib/ui';
import TripArt from '@/components/trips/TripArt';

type Trip = {
  id: string;
  title: string;
  destination: string;
  price: number;
  date: string;
  total_slots: number;
  available_slots: number;
};

// Form admin menetapkan kapasitas (total_slots). Sisa kursi dihitung
// server dari reservasi yang berjalan, jadi tidak pernah dikirim dari sini.
type TripForm = Omit<Trip, 'id' | 'available_slots'>;

const initialForm: TripForm = {
  title: '',
  destination: '',
  price: 0,
  date: '',
  total_slots: 0,
};

export default function AdminTripsPage() {
  const [trips, setTrips] = useState<Trip[]>([]);
  const [form, setForm] = useState<TripForm>(initialForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitted, setSubmitted] = useState(false);

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
    // Pengambilan awal ditulis langsung di dalam effect. Memanggil
    // loadTrips() di sini akan menjalankan setState secara sinkron pada
    // saat effect berjalan, yang memicu render berantai.
    let cancelled = false;

    void (async () => {
      try {
        const res = await adminTripApi.get('/trips');
        if (!cancelled) setTrips(res.data.data || []);
      } catch {
        if (!cancelled) toast.error('Gagal memuat data trip');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  const resetForm = () => {
    setForm(initialForm);
    setEditingId(null);
    setSubmitted(false);
  };

  const errors = {
    title: form.title.trim() ? '' : 'Nama trip wajib diisi',
    destination: form.destination.trim() ? '' : 'Lokasi wajib diisi',
    date: form.date ? '' : 'Pilih tanggal keberangkatan',
    price: form.price > 0 ? '' : 'Harga harus lebih dari 0',
    total_slots: form.total_slots > 0 ? '' : 'Total slot harus lebih dari 0',
  };

  const submitTrip = async () => {
    setSubmitted(true);
    if (!form.title || !form.destination || !form.date || form.price <= 0 || form.total_slots <= 0) {
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

  const field = (key: keyof typeof errors, label: string, control: React.ReactNode) => (
    <div>
      <label htmlFor={`f-${key}`} className="mb-1.5 block text-sm font-semibold text-ink">{label}</label>
      {control}
      {submitted && errors[key] ? <p id={`e-${key}`} className="mt-1 text-xs font-semibold text-bad">{errors[key]}</p> : null}
    </div>
  );
  const bad = (key: keyof typeof errors) => (submitted && errors[key] ? { 'aria-invalid': true, 'aria-describedby': `e-${key}` } : {});

  return (
    <section className="space-y-6">
      <div className={`${card} p-6`}>
        <h2 className="text-xl font-bold text-navy">{editingId ? 'Ubah trip' : 'Buat trip baru'}</h2>
        <p className="mb-5 mt-1 text-sm text-ink-soft">Lengkapi detail trip agar mudah dipahami pengguna.</p>
        <div className="grid gap-4 md:grid-cols-2">
          {field('title', 'Nama trip', <input id="f-title" className={input} placeholder="Contoh: Liburan Bali 3 Hari" value={form.title} onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))} {...bad('title')} />)}
          {field('destination', 'Lokasi', <input id="f-destination" className={input} placeholder="Contoh: Bali" value={form.destination} onChange={(e) => setForm((p) => ({ ...p, destination: e.target.value }))} {...bad('destination')} />)}
          {field('date', 'Tanggal keberangkatan', <input id="f-date" type="date" className={input} value={form.date} onChange={(e) => setForm((p) => ({ ...p, date: e.target.value }))} {...bad('date')} />)}
          {field('price', 'Harga per kursi (Rupiah)', <input id="f-price" type="number" min={1} className={input} placeholder="Contoh: 1500000" value={form.price || ''} onChange={(e) => setForm((p) => ({ ...p, price: Number(e.target.value) }))} {...bad('price')} />)}
          {field('total_slots', 'Total slot', <input id="f-total_slots" type="number" min={1} className={input} placeholder="Contoh: 20" value={form.total_slots || ''} onChange={(e) => setForm((p) => ({ ...p, total_slots: Number(e.target.value) }))} {...bad('total_slots')} />)}
          <div className="rounded-xl bg-krem p-4 text-sm text-ink-soft">
            <p className="font-semibold text-ink">Pratinjau</p>
            <p className="mt-1">Harga per kursi: {formatRupiah(form.price)}</p>
            <p className="mt-1">Total potensi pendapatan: {formatRupiah(form.price * form.total_slots)}</p>
          </div>
          <div className="flex gap-2 md:col-span-2">
            <button onClick={submitTrip} disabled={loading} className={btnPrimary}>
              {loading ? 'Menyimpan...' : editingId ? 'Simpan perubahan' : 'Buat trip'}
            </button>
            {editingId ? <button onClick={resetForm} className={btnGhost}>Batal ubah</button> : null}
          </div>
        </div>
      </div>

      <div className={tableWrap}>
        <table className="min-w-full text-sm">
          <thead className="bg-krem/70">
            <tr>
              <th className={th}>Trip</th>
              <th className={th}>Lokasi</th>
              <th className={th}>Tanggal</th>
              <th className={th}>Harga</th>
              <th className={th}>Sisa / total</th>
              <th className={th}>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {trips.map((trip) => (
              <tr key={trip.id} className="border-t border-garis">
                <td className={td}>
                  <div className="flex items-center gap-3">
                    <span className="h-10 w-14 shrink-0 overflow-hidden rounded-lg">
                      <TripArt id={trip.id} destination={trip.destination} title={trip.title} className="h-full w-full" />
                    </span>
                    <span className="font-semibold text-navy">{trip.title}</span>
                  </div>
                </td>
                <td className={td}>{trip.destination}</td>
                <td className={`${td} whitespace-nowrap`}>{formatTanggal(trip.date)}</td>
                <td className={`${td} whitespace-nowrap`}>{formatRupiah(trip.price)}</td>
                <td className={td}>{trip.available_slots} / {trip.total_slots}</td>
                <td className={td}>
                  <div className="flex gap-2">
                    <button
                      onClick={() => {
                        setEditingId(trip.id);
                        setSubmitted(false);
                        setForm({ title: trip.title, destination: trip.destination, date: trip.date, price: trip.price, total_slots: trip.total_slots });
                        window.scrollTo({ top: 0, behavior: 'smooth' });
                      }}
                      className={`${btnGhost} ${btnSmall}`}
                    >
                      Ubah
                    </button>
                    <button onClick={() => deleteTrip(trip.id)} className={`${btnDanger} ${btnSmall}`}>Hapus</button>
                  </div>
                </td>
              </tr>
            ))}
            {!loading && trips.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-10 text-center text-ink-soft">Belum ada trip.</td></tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}
