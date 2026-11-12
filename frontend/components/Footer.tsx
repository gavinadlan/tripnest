import Logo from '@/components/Logo';

export default function Footer() {
  return (
    <footer className="mt-auto border-t border-garis bg-krem-tua/50">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-3 px-4 py-6 text-sm text-ink-soft sm:flex-row sm:px-6 lg:px-8">
        <Logo className="h-7" />
        <p>&copy; 2026 TripNest &middot; Proyek portofolio</p>
      </div>
    </footer>
  );
}
