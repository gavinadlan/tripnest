import { Loader2 } from 'lucide-react';

export default function Spinner({ className = "h-6 w-6 text-indigo-600" }: { className?: string }) {
    return (
        <div className="flex justify-center items-center">
            <Loader2 className={`animate-spin ${className}`} />
        </div>
    );
}
