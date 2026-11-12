'use client';

import { useEffect, useRef, useState } from 'react';

/**
 * Angka yang menghitung naik dari 0. Pola "Count Up" ala React Bits,
 * ditulis ulang tanpa dependensi. Bila pengguna meminta gerak dikurangi,
 * angka langsung tampil.
 */
export default function CountUp({ value, durationMs = 900, format = (n: number) => String(n) }: { value: number; durationMs?: number; format?: (n: number) => string }) {
  const [shown, setShown] = useState(0);
  const from = useRef(0);

  useEffect(() => {
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const start = from.current;
    if (reduce || start === value) {
      from.current = value;
      const t = requestAnimationFrame(() => setShown(value));
      return () => cancelAnimationFrame(t);
    }
    let raf = 0;
    const t0 = performance.now();
    const tick = (now: number) => {
      const p = Math.min(1, (now - t0) / durationMs);
      const eased = 1 - Math.pow(1 - p, 3);
      setShown(Math.round(start + (value - start) * eased));
      if (p < 1) raf = requestAnimationFrame(tick);
      else from.current = value;
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [value, durationMs]);

  return <span>{format(shown)}</span>;
}
