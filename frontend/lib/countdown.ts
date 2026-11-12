'use client';

import { useEffect, useState } from 'react';

/** Sisa detik menuju `target` (ISO). null bila target kosong, 0 bila sudah lewat. */
export function useCountdown(target: string | undefined): number | null {
  const [now, setNow] = useState<number | null>(null);
  useEffect(() => {
    if (!target) return;
    const tick = () => setNow(Date.now());
    tick();
    const t = setInterval(tick, 1000);
    return () => clearInterval(t);
  }, [target]);
  if (!target || now === null) return null;
  return Math.max(0, Math.floor((new Date(target).getTime() - now) / 1000));
}

export function formatSisa(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}
