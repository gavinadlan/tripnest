'use client';

import { X } from 'lucide-react';

declare global {
  interface Window {
    snap?: {
      pay: (
        token: string,
        options?: {
          onSuccess?: () => void;
          onPending?: () => void;
          onError?: () => void;
          onClose?: () => void;
        }
      ) => void;
    };
  }
}

export default function PaymentModal({
  open,
  snapToken,
  onClose,
  onSuccess,
  onPending,
  onError,
}: {
  open: boolean;
  snapToken: string;
  onClose: () => void;
  onSuccess: () => void;
  onPending: () => void;
  onError: () => void;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="w-full max-w-md rounded-2xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--color-text-primary)]">Selesaikan Pembayaran</h3>
          <button onClick={onClose} className="rounded-lg p-1 text-[var(--color-text-secondary)] hover:bg-slate-100">
            <X className="h-5 w-5" />
          </button>
        </div>
        <p className="text-sm text-[var(--color-text-secondary)]">
          Lanjutkan ke gateway pembayaran Midtrans yang aman.
        </p>
        <button
          onClick={() => {
            if (!window.snap) return;
            window.snap.pay(snapToken, {
              onSuccess,
              onPending,
              onError,
              onClose,
            });
          }}
          className="mt-6 w-full rounded-xl bg-[var(--color-primary)] px-4 py-2.5 text-sm font-semibold text-white"
        >
          Bayar Sekarang
        </button>
      </div>
    </div>
  );
}
