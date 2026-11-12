import { Loader2 } from 'lucide-react';

export default function Spinner({ className = 'h-6 w-6 text-biru' }: { className?: string }) {
  return (
    <div role="status" aria-label="Memuat" className="flex items-center justify-center">
      <Loader2 className={`animate-spin ${className}`} aria-hidden />
    </div>
  );
}
