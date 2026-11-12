import { Quote } from 'lucide-react';
import TripArt from '@/components/trips/TripArt';
import Logo from '@/components/Logo';

/** Dua kolom di layar lebar: kartu form di kiri, blok visual dan kutipan di kanan. */
export default function AuthLayout({
  title,
  subtitle,
  children,
  artId,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  artId: string;
}) {
  return (
    <div className="mx-auto grid max-w-5xl overflow-hidden rounded-3xl border border-garis bg-surface lg:grid-cols-2">
      <div className="p-6 sm:p-10">
        <Logo className="h-8" />
        <h1 className="mt-8 text-3xl font-extrabold text-navy sm:text-4xl">{title}</h1>
        <p className="mt-2 text-ink-soft">{subtitle}</p>
        <div className="mt-8">{children}</div>
      </div>
      <div className="on-dark relative hidden min-h-[28rem] lg:block">
        <TripArt id={artId} destination="Raja Ampat" title="Island" className="absolute inset-0 h-full w-full" />
        <div className="absolute inset-0 bg-gradient-to-t from-dark-bg via-dark-bg/30 to-transparent" />
        <figure className="absolute inset-x-0 bottom-0 p-8 text-white">
          <Quote className="h-6 w-6 text-amber" aria-hidden />
          <blockquote className="font-display mt-3 text-2xl font-bold leading-snug">
            Kursimu dikunci dulu, baru kamu membayar. Tidak ada rebutan di menit terakhir.
          </blockquote>
          <figcaption className="mt-3 text-sm text-white/80">Cara kerja pemesanan TripNest</figcaption>
        </figure>
      </div>
    </div>
  );
}
