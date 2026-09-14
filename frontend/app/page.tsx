'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { searchApi } from '@/lib/api';
import { Trip } from '@/lib/types';
import TripCard from '@/components/trips/TripCard';
import toast from 'react-hot-toast';

const PAGE_SIZE = 9;

export default function HomePage() {
  const router = useRouter();
  const [trips, setTrips] = useState<Trip[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [bookingTripId, setBookingTripId] = useState<string | null>(null);

  // Nilai di kotak isian dipisah dari filter yang benar-benar dikirim,
  // supaya setiap ketikan tidak langsung memicu request.
  const [destinationInput, setDestinationInput] = useState('');
  const [minPriceInput, setMinPriceInput] = useState('');
  const [maxPriceInput, setMaxPriceInput] = useState('');
  const [filters, setFilters] = useState({ destination: '', minPrice: '', maxPrice: '' });
  const [page, setPage] = useState(1);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    // Penyaringan dan paginasi dikerjakan server. Sebelumnya seluruh trip
    // ditarik sekaligus lalu disaring di browser, jadi parameter /search
    // yang sudah ada tidak pernah terpakai.
    void (async () => {
      try {
        const res = await searchApi.get('/search', {
          params: {
            destination: filters.destination || undefined,
            min_price: filters.minPrice || undefined,
            max_price: filters.maxPrice || undefined,
            page,
            limit: PAGE_SIZE,
          },
        });
        if (cancelled) return;
        setTrips(res.data.data || []);
        setTotal(res.data.total || 0);
      } catch {
        if (!cancelled) toast.error('Gagal memuat daftar trip');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [filters, page]);

  const applyFilters = () => {
    setPage(1);
    setFilters({ destination: destinationInput.trim(), minPrice: minPriceInput, maxPrice: maxPriceInput });
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const onBook = (trip: Trip) => {
    setBookingTripId(trip.id);
    // Hanya id yang dioper. Judul, tanggal, dan harga diambil ulang dari
    // server di halaman booking -- harga tidak boleh datang dari URL.
    router.push(`/booking?tripId=${encodeURIComponent(trip.id)}`);
  };

  return (
    <section className="space-y-8">
      <div className="rounded-3xl bg-gradient-to-r from-[var(--color-muted)] to-white p-8 shadow-sm">
        <p className="text-sm font-medium text-[var(--color-primary)]">Temukan Perjalanan Impianmu</p>
        <h1 className="mt-2 text-4xl font-bold text-[var(--color-text-primary)]">Booking travel modern, cepat, dan aman.</h1>
        <p className="mt-3 max-w-2xl text-[var(--color-text-secondary)]">
          Jelajahi trip yang tersedia, booking dalam hitungan detik, lalu selesaikan pembayaran aman lewat Midtrans.
        </p>
        <form
          className="mt-6 flex flex-wrap items-end gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            applyFilters();
          }}
        >
          <input
            value={destinationInput}
            onChange={(e) => setDestinationInput(e.target.value)}
            placeholder="Cari destinasi"
            className="w-full max-w-xs rounded-xl border border-[var(--color-border)] bg-white px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none ring-[var(--color-primary)]/20 focus:ring-4"
          />
          <input
            type="number"
            min={0}
            value={minPriceInput}
            onChange={(e) => setMinPriceInput(e.target.value)}
            placeholder="Harga minimum"
            className="w-40 rounded-xl border border-[var(--color-border)] bg-white px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none ring-[var(--color-primary)]/20 focus:ring-4"
          />
          <input
            type="number"
            min={0}
            value={maxPriceInput}
            onChange={(e) => setMaxPriceInput(e.target.value)}
            placeholder="Harga maksimum"
            className="w-40 rounded-xl border border-[var(--color-border)] bg-white px-4 py-2.5 text-sm text-[var(--color-text-primary)] outline-none ring-[var(--color-primary)]/20 focus:ring-4"
          />
          <button type="submit" className="rounded-xl bg-[var(--color-primary)] px-5 py-2.5 text-sm font-semibold text-white">
            Cari
          </button>
        </form>
      </div>

      {loading ? (
        <p className="text-[var(--color-text-secondary)]">Memuat daftar trip...</p>
      ) : trips.length === 0 ? (
        <p className="text-[var(--color-text-secondary)]">Tidak ada trip yang cocok dengan pencarianmu.</p>
      ) : (
        <>
          <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
            {trips.map((trip) => (
              <TripCard key={trip.id} trip={trip} onBook={onBook} loading={bookingTripId === trip.id} />
            ))}
          </div>

          <div className="flex items-center justify-between">
            <p className="text-sm text-[var(--color-text-secondary)]">
              Halaman {page} dari {totalPages} &middot; {total} trip
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-sm disabled:opacity-40"
              >
                Sebelumnya
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="rounded-lg border border-[var(--color-border)] px-3 py-1.5 text-sm disabled:opacity-40"
              >
                Berikutnya
              </button>
            </div>
          </div>
        </>
      )}
    </section>
  );
}
