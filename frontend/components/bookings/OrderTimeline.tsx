'use client';

import { motion, useReducedMotion } from 'motion/react';
import { Armchair, Ban, CheckCheck, Hourglass, Wallet } from 'lucide-react';

/**
 * "Perjalanan pesanan": garis waktu vertikal dengan beam yang terisi
 * sampai langkah aktif. Pola "Timeline" ala Aceternity UI, ditulis ulang
 * dengan motion dan dibangun dari status booking nyata.
 */
const NORMAL = [
  { status: 'PENDING_SEAT', title: 'Menahan kursi', desc: 'Katalog memastikan kursimu masih tersedia dan menahannya.', icon: Armchair },
  { status: 'PENDING_PAYMENT', title: 'Menunggu pembayaran', desc: 'Kursi sudah tertahan. Selesaikan pembayaran sebelum batas waktu.', icon: Wallet },
  { status: 'CONFIRMED', title: 'Terkonfirmasi', desc: 'Pembayaran diterima. Kursimu aman, sampai jumpa di hari keberangkatan.', icon: CheckCheck },
];

const END: Record<string, { title: string; desc: string; icon: typeof Ban }> = {
  CANCELLED: { title: 'Dibatalkan', desc: 'Pesanan dihentikan dan kursi dikembalikan ke stok.', icon: Ban },
  EXPIRED: { title: 'Kedaluwarsa', desc: 'Batas waktu pembayaran habis, jadi kursi dilepas otomatis.', icon: Hourglass },
};

export default function OrderTimeline({ status }: { status: string }) {
  const reduce = useReducedMotion();
  const failed = status in END;
  const steps = failed
    ? [...NORMAL.slice(0, 2).map((s) => ({ ...s, state: 'lewat' as const })), { ...END[status], status, state: 'akhir' as const }]
    : NORMAL.map((s, i) => {
        const cur = Math.max(0, NORMAL.findIndex((n) => n.status === status));
        return { ...s, state: i < cur ? ('selesai' as const) : i === cur ? ('aktif' as const) : ('menunggu' as const) };
      });

  const activeIndex = steps.findIndex((s) => s.state === 'aktif' || s.state === 'akhir');
  const done = status === 'CONFIRMED';
  const fill = done || failed ? 100 : (Math.max(activeIndex, 0) / (steps.length - 1)) * 100;

  return (
    <ol className="relative space-y-7 pl-14" aria-label="Perjalanan pesanan">
      <span aria-hidden className="absolute bottom-4 left-[19px] top-4 w-0.5 bg-garis" />
      <motion.span
        aria-hidden
        className={`absolute left-[19px] top-4 w-0.5 origin-top ${failed ? 'bg-ink-soft/50' : done ? 'bg-ok' : 'bg-biru'}`}
        style={{ bottom: 16 }}
        initial={reduce ? false : { scaleY: 0 }}
        animate={{ scaleY: fill / 100 }}
        transition={{ duration: 0.8, ease: 'easeOut' }}
      />
      {steps.map((s) => {
        const Icon = s.icon;
        const tone =
          s.state === 'akhir'
            ? status === 'CANCELLED'
              ? 'border-bad bg-bad-bg text-bad'
              : 'border-mute bg-mute-bg text-mute'
            : s.state === 'selesai' || (s.state === 'aktif' && done)
              ? 'border-ok bg-ok text-white'
              : s.state === 'aktif'
                ? 'border-biru bg-biru text-white'
                : s.state === 'lewat'
                  ? 'border-garis bg-krem-tua text-ink-soft'
                  : 'border-garis bg-surface text-ink-soft';
        const active = s.state === 'aktif' || s.state === 'akhir';
        return (
          <li key={s.status} className="relative" aria-current={active ? 'step' : undefined}>
            <span className={`absolute -left-14 top-0 flex h-10 w-10 items-center justify-center rounded-full border-2 ${tone}`}>
              <Icon className="h-4 w-4" aria-hidden />
              {s.state === 'aktif' && !done && !reduce ? (
                <span aria-hidden className="absolute inset-0 animate-ping rounded-full border-2 border-biru opacity-40" />
              ) : null}
            </span>
            <h3 className={`text-base font-semibold ${s.state === 'menunggu' ? 'text-ink-soft' : 'text-ink'}`}>{s.title}</h3>
            <p className="mt-0.5 text-sm text-ink-soft">{s.desc}</p>
          </li>
        );
      })}
    </ol>
  );
}
