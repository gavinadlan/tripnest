/** Kelas Tailwind bersama supaya tombol, kolom isian, dan kartu konsisten. */
const base =
  'inline-flex items-center justify-center gap-2 rounded-xl px-5 py-2.5 text-sm font-semibold transition focus-visible:outline-offset-2 disabled:cursor-not-allowed';

export const btnPrimary = `${base} bg-biru text-white hover:bg-biru-tua disabled:bg-krem-tua disabled:text-ink-soft`;
export const btnAmber = `${base} bg-amber text-navy hover:brightness-95 disabled:bg-krem-tua disabled:text-ink-soft`;
export const btnGhost = `${base} border border-garis bg-surface text-ink hover:bg-krem-tua disabled:text-ink-soft`;
export const btnDanger = `${base} bg-bad text-white hover:brightness-110 disabled:bg-krem-tua disabled:text-ink-soft`;
export const btnSmall = 'rounded-lg px-3 py-1.5 text-xs';

export const input =
  'w-full rounded-xl border border-garis bg-surface px-4 py-2.5 text-sm text-ink placeholder:text-ink-soft/70 focus:border-biru focus:outline-none focus:ring-4 focus:ring-biru/20';

export const card = 'rounded-2xl border border-garis bg-surface';

export const tableWrap = 'overflow-x-auto rounded-2xl border border-garis bg-surface';
export const th = 'whitespace-nowrap px-4 py-3 text-left text-xs font-bold uppercase tracking-wide text-ink-soft';
export const td = 'px-4 py-3 align-middle';
export const select = 'rounded-xl border border-garis bg-surface px-3 py-2 text-sm text-ink focus:border-biru focus:outline-none focus:ring-4 focus:ring-biru/20';
