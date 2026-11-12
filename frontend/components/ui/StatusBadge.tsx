'use client';

type Status = 'PENDING_SEAT' | 'PENDING_PAYMENT' | 'CONFIRMED' | 'CANCELLED' | 'EXPIRED' | 'FAILED' | string;

const styles: Record<string, string> = {
  PENDING_SEAT: 'bg-biru-muda/30 text-navy border-biru/30',
  PENDING_PAYMENT: 'bg-warn-bg text-warn border-amber/60',
  PENDING: 'bg-warn-bg text-warn border-amber/60',
  CONFIRMED: 'bg-ok-bg text-ok border-ok/30',
  SUCCESS: 'bg-ok-bg text-ok border-ok/30',
  CANCELLED: 'bg-bad-bg text-bad border-bad/30',
  FAILED: 'bg-bad-bg text-bad border-bad/30',
  EXPIRED: 'bg-mute-bg text-mute border-mute/30',
};

export const STATUS_LABEL: Record<string, string> = {
  PENDING_SEAT: 'Menahan kursi',
  PENDING_PAYMENT: 'Menunggu pembayaran',
  CONFIRMED: 'Terkonfirmasi',
  CANCELLED: 'Dibatalkan',
  EXPIRED: 'Kedaluwarsa',
  FAILED: 'Gagal',
  SUCCESS: 'Berhasil',
  PENDING: 'Menunggu',
};

export default function StatusBadge({ status }: { status: Status }) {
  const label = STATUS_LABEL[status] || status.replaceAll('_', ' ');
  const style = styles[status] || 'bg-krem-tua text-ink border-garis';
  return (
    <span className={`inline-flex items-center whitespace-nowrap rounded-full border px-3 py-1 text-xs font-semibold ${style}`}>
      {label}
    </span>
  );
}
