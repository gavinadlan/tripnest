export default function StatCard({
  label,
  value,
  hint,
  tone = 'default',
  className = '',
}: {
  label: string;
  value: React.ReactNode;
  hint?: string;
  tone?: 'default' | 'success' | 'error' | 'warning';
  className?: string;
}) {
  const toneMap = {
    default: 'text-navy',
    success: 'text-ok',
    error: 'text-bad',
    warning: 'text-warn',
  };
  return (
    <article className={`rounded-2xl border border-garis bg-surface p-5 ${className}`}>
      <p className="text-sm font-semibold text-ink-soft">{label}</p>
      <p className={`font-display mt-2 text-5xl font-bold ${toneMap[tone]}`}>{value}</p>
      {hint ? <p className="mt-2 text-xs text-ink-soft">{hint}</p> : null}
    </article>
  );
}
