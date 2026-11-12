/**
 * Bingkai dengan gradien yang berputar di tepi, untuk tombol ajakan utama.
 * Pola "Moving Border" ala Aceternity UI, ditulis ulang dengan CSS murni
 * dan palet Samudra. Animasinya mati otomatis lewat prefers-reduced-motion
 * (aturan global di globals.css).
 */
export default function MovingBorder({ children, className = '' }: { children: React.ReactNode; className?: string }) {
  return (
    <span className={`relative inline-flex overflow-hidden rounded-xl p-[2px] ${className}`}>
      <span
        aria-hidden
        className="absolute left-1/2 top-1/2 aspect-square w-[250%] -translate-x-1/2 -translate-y-1/2 animate-[spin_4s_linear_infinite] bg-[conic-gradient(from_0deg,transparent_0_70%,var(--color-amber)_85%,var(--color-biru-muda)_100%)]"
      />
      <span className="relative inline-flex rounded-[10px] bg-navy">{children}</span>
    </span>
  );
}
