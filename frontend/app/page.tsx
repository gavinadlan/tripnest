'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowDown, Search, SearchX } from 'lucide-react';
import { searchApi } from '@/lib/api';
import { Trip } from '@/lib/types';
import TripCard from '@/components/trips/TripCard';
import TripArt from '@/components/trips/TripArt';
import HoverGrid from '@/components/ui/HoverGrid';
import BlurText from '@/components/ui/BlurText';
import MovingBorder from '@/components/ui/MovingBorder';
import { btnAmber, btnGhost, input } from '@/lib/ui';
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

    // Penyaringan dan paginasi dikerjakan server lewat /search.
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

  const resetFilters = () => {
    setDestinationInput('');
    setMinPriceInput('');
    setMaxPriceInput('');
    setPage(1);
    setFilters({ destination: '', minPrice: '', maxPrice: '' });
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const filtering = Boolean(filters.destination || filters.minPrice || filters.maxPrice);

  const onBook = (trip: Trip) => {
    setBookingTripId(trip.id);
    // Hanya id yang dioper. Judul, tanggal, dan harga diambil ulang dari
    // server di halaman booking -- harga tidak boleh datang dari URL.
    router.push(`/booking?tripId=${encodeURIComponent(trip.id)}`);
  };

  return (
    <section className="space-y-10">
      <div className="on-dark relative overflow-hidden rounded-3xl bg-dark-bg text-white">
        <div aria-hidden className="absolute inset-y-0 right-0 hidden w-3/5 md:block">
          <TripArt id="hero-tripnest-senja" destination="Bali" title="Sunset" className="h-full w-full opacity-90" />
          <div className="absolute inset-0 bg-gradient-to-r from-dark-bg via-dark-bg/70 to-transparent" />
        </div>
        <div className="relative px-6 py-12 sm:px-10 sm:py-16">
          <p className="text-sm font-bold uppercase tracking-[0.2em] text-amber">Temukan perjalanan impianmu</p>
          <h1 className="mt-3 max-w-2xl text-4xl font-extrabold leading-[1.05] sm:text-6xl">
            <BlurText text="Pesan trip, pastikan kursimu aman." />
          </h1>
          <p className="mt-4 max-w-xl text-lg text-white/85">
            Jelajahi trip yang tersedia, pesan dalam hitungan detik, lalu bayar lewat Midtrans. Kursi baru dikunci setelah dipastikan tersedia.
          </p>

          <form
            className="mt-8 grid max-w-3xl gap-3 sm:grid-cols-[1.6fr_1fr_1fr_auto]"
            onSubmit={(e) => {
              e.preventDefault();
              applyFilters();
            }}
            role="search"
            aria-label="Cari trip"
          >
            <input aria-label="Destinasi" value={destinationInput} onChange={(e) => setDestinationInput(e.target.value)} placeholder="Cari destinasi" className={input} />
            <input aria-label="Harga minimum" type="number" min={0} value={minPriceInput} onChange={(e) => setMinPriceInput(e.target.value)} placeholder="Harga minimum" className={input} />
            <input aria-label="Harga maksimum" type="number" min={0} value={maxPriceInput} onChange={(e) => setMaxPriceInput(e.target.value)} placeholder="Harga maksimum" className={input} />
            <button type="submit" className={btnAmber}>
              <Search className="h-4 w-4" aria-hidden /> Cari
            </button>
          </form>

          <div className="mt-6">
            <MovingBorder>
              <a href="#daftar-trip" className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-semibold text-white">
                Lihat semua trip <ArrowDown className="h-4 w-4" aria-hidden />
              </a>
            </MovingBorder>
          </div>
        </div>
      </div>

      <div id="daftar-trip" className="scroll-mt-24">
        {loading ? (
          <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3" role="status" aria-label="Memuat daftar trip">
            {Array.from({ length: 6 }, (_, i) => (
              <div key={i} className="h-[22rem] animate-pulse rounded-2xl border border-garis bg-krem-tua/60" />
            ))}
          </div>
        ) : trips.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-garis bg-surface px-6 py-14 text-center">
            <SearchX className="mx-auto h-10 w-10 text-ink-soft" aria-hidden />
            <h2 className="mt-4 text-2xl font-bold text-navy">Belum ada trip yang cocok</h2>
            <p className="mx-auto mt-2 max-w-md text-ink-soft">
              {filtering ? 'Coba ganti destinasi atau longgarkan rentang harga.' : 'Daftar trip masih kosong. Cek lagi sebentar lagi.'}
            </p>
            {filtering ? (
              <button onClick={resetFilters} className={`${btnGhost} mt-5`}>Hapus filter</button>
            ) : null}
          </div>
        ) : (
          <>
            <HoverGrid
              className="-m-2 grid md:grid-cols-2 xl:grid-cols-3"
              items={trips.map((trip) => ({
                key: trip.id,
                node: <TripCard trip={trip} onBook={onBook} loading={bookingTripId === trip.id} />,
              }))}
            />

            <div className="mt-8 flex flex-wrap items-center justify-between gap-3">
              <p className="text-sm text-ink-soft">
                Halaman {page} dari {totalPages} &middot; {total} trip
              </p>
              <div className="flex gap-2">
                <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className={btnGhost}>Sebelumnya</button>
                <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className={btnGhost}>Berikutnya</button>
              </div>
            </div>
          </>
        )}
      </div>
    </section>
  );
}
