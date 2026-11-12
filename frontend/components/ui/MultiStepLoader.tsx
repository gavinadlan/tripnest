'use client';

import { Check, Loader2 } from 'lucide-react';

/**
 * Daftar langkah dengan penanda selesai / sedang berjalan / menunggu.
 * Pola "Multi Step Loader" ala Aceternity UI, ditulis ulang tanpa overlay
 * penuh layar supaya tetap terbaca screen reader (role=status).
 */
export default function MultiStepLoader({ steps, active }: { steps: string[]; active: number }) {
  return (
    <ol role="status" aria-live="polite" className="space-y-3">
      {steps.map((label, i) => {
        const done = i < active;
        const current = i === active;
        return (
          <li key={label} className="flex items-center gap-3 text-sm">
            <span
              className={`flex h-6 w-6 items-center justify-center rounded-full border ${
                done ? 'border-ok bg-ok text-white' : current ? 'border-biru bg-surface text-biru' : 'border-garis bg-krem-tua text-ink-soft'
              }`}
            >
              {done ? <Check className="h-3.5 w-3.5" aria-hidden /> : current ? <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden /> : <span className="text-[10px]">{i + 1}</span>}
            </span>
            <span className={current ? 'font-semibold text-ink' : done ? 'text-ink' : 'text-ink-soft'}>{label}</span>
          </li>
        );
      })}
    </ol>
  );
}
