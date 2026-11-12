const BULAN = ['Januari','Februari','Maret','April','Mei','Juni','Juli','Agustus','September','Oktober','November','Desember'];

/** "2027-01-18" -> "18 Januari 2027". Nilai yang bukan tanggal dikembalikan apa adanya. */
export function formatTanggal(value: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value || '');
  if (!m) return value || '-';
  return `${Number(m[3])} ${BULAN[Number(m[2]) - 1] ?? m[2]} ${m[1]}`;
}

/** Tanggal dan jam lokal dalam format Indonesia. */
export function formatWaktu(value: string | undefined): string {
  if (!value) return '-';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString('id-ID', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

/** Tampilan ID pendek untuk fallback bila judul trip tidak bisa dimuat. */
export function idPendek(id: string): string {
  return id ? id.slice(0, 8) : '-';
}
