'use client';

import { useEffect, useState } from 'react';
import { searchApi } from './api';
import { Trip } from './types';

/**
 * Cache sisi klien untuk detail trip, dikunci oleh id.
 *
 * Booking hanya menyimpan resource_id, jadi judul trip diambil dari
 * search-service (GET /trips/{id}, publik). Promise disimpan, bukan
 * hasilnya, supaya banyak komponen yang meminta id yang sama hanya
 * memicu satu request. null berarti trip tidak ditemukan atau gagal dimuat.
 */
const cache = new Map<string, Promise<Trip | null>>();

export function fetchTrip(id: string): Promise<Trip | null> {
  let p = cache.get(id);
  if (!p) {
    p = searchApi
      .get(`/trips/${encodeURIComponent(id)}`)
      .then((res) => res.data as Trip)
      .catch(() => null);
    cache.set(id, p);
  }
  return p;
}

/** Memuat sekumpulan trip. Nilai undefined = masih memuat, null = tidak tersedia. */
export function useTrips(ids: string[]): Record<string, Trip | null | undefined> {
  const key = Array.from(new Set(ids.filter(Boolean))).sort().join(',');
  const [map, setMap] = useState<Record<string, Trip | null | undefined>>({});

  useEffect(() => {
    if (!key) return;
    let cancelled = false;
    for (const id of key.split(',')) {
      void fetchTrip(id).then((trip) => {
        if (!cancelled) setMap((prev) => (prev[id] === trip ? prev : { ...prev, [id]: trip }));
      });
    }
    return () => {
      cancelled = true;
    };
  }, [key]);

  return map;
}

export function useTrip(id: string): Trip | null | undefined {
  return useTrips(id ? [id] : [])[id];
}

/** Nama yang ditampilkan untuk sebuah trip: judul, "Memuat..." selagi menunggu, atau ID pendek bila tidak ada. */
export function tripLabel(trip: Trip | null | undefined, id: string): string {
  if (trip) return trip.title;
  if (trip === undefined) return 'Memuat trip...';
  return `Trip ${id.slice(0, 8)}`;
}
