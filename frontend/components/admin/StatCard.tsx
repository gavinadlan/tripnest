export default function StatCard({
  label,
  value,
  tone = 'default',
}: {
  label: string;
  value: string | number;
  tone?: 'default' | 'success' | 'error' | 'warning';
}) {
  const toneMap: Record<string, string> = {
    default: 'text-[var(--color-primary)]',
    success: 'text-[var(--color-success)]',
    error: 'text-[var(--color-error)]',
    warning: 'text-[var(--color-warning)]',
  };

  return (
    <article className="rounded-2xl border border-[var(--color-border)] bg-white p-5 shadow-sm">
      <p className="text-sm text-[var(--color-text-secondary)]">{label}</p>
      <p className={`mt-2 text-3xl font-bold ${toneMap[tone]}`}>{value}</p>
    </article>
  );
}
