import { useId } from 'react';

/**
 * Ilustrasi destinasi yang dibuat sendiri: langit bergradien, matahari atau
 * bulan, dan siluet berlapis sesuai kata kunci. Deterministik: id yang sama
 * selalu menghasilkan gambar yang sama. Tanpa foto, tanpa aset eksternal.
 */

type Scene = 'gunung' | 'laut' | 'pantai' | 'budaya' | 'bawaan';

const W = 400;
const H = 240;

function hash(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

function rng(seed: number) {
  let a = seed || 1;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function sceneFor(destination: string, title: string): Scene {
  const t = `${title} ${destination}`.toLowerCase();
  if (/yogya|heritage|candi|budaya|keraton/.test(t)) return 'budaya';
  if (/bromo|rinjani|toba|trek|highland|gunung|danau|malang/.test(t)) return 'gunung';
  if (/raja ampat|komodo|labuan|derawan|island|diving|sailing|pulau|kalimantan/.test(t)) return 'laut';
  if (/bali|lombok|sunset|pantai|beach/.test(t)) return 'pantai';
  return 'bawaan';
}

const SKIES = [
  { name: 'pagi', top: '#8ec5ff', bottom: '#ffe3a8', sun: '#fff6df', far: '#6fa4d6', mid: '#2f6fb0', near: '#0b2a4a' },
  { name: 'siang', top: '#1b5fa8', bottom: '#8ec5ff', sun: '#ffd37a', far: '#4a86c4', mid: '#1b5fa8', near: '#0b2a4a' },
  { name: 'senja', top: '#0b2a4a', bottom: '#ffb02e', sun: '#ffe9b0', far: '#7b6a9e', mid: '#2b3f6e', near: '#07182b' },
  { name: 'fajar', top: '#2a4d86', bottom: '#f4795b', sun: '#ffd9a0', far: '#8a6a98', mid: '#35477a', near: '#07182b' },
];

function peaks(r: () => number, base: number, minH: number, maxH: number, n: number): string {
  const step = W / n;
  let d = `M0 ${H} L0 ${base}`;
  for (let i = 0; i <= n; i++) {
    const x = i * step + (i % 2 ? step * 0.5 : 0) * (r() * 0.4);
    const y = base - (minH + r() * (maxH - minH)) * (i % 2 ? 1 : 0.45);
    d += ` L${x.toFixed(1)} ${y.toFixed(1)}`;
  }
  return d + ` L${W} ${base} L${W} ${H} Z`;
}

function wave(y: number, amp: number, phase: number): string {
  let d = `M0 ${H} L0 ${y}`;
  for (let x = 0; x <= W; x += 50) {
    const yy = y + Math.sin(x / 38 + phase) * amp;
    d += ` Q${x + 25} ${(yy - amp).toFixed(1)} ${x + 50} ${yy.toFixed(1)}`;
  }
  return d + ` L${W} ${H} Z`;
}

function island(cx: number, base: number, w: number, h: number): string {
  return `M${cx - w} ${base} Q${cx - w / 2} ${base - h} ${cx} ${base - h * 0.8} Q${cx + w / 2} ${base - h} ${cx + w} ${base} Z`;
}

function temple(cx: number, base: number, scale: number): string {
  const tiers = [
    [90, 16],
    [70, 16],
    [50, 14],
    [32, 12],
  ];
  let d = '';
  let y = base;
  for (const [w, h] of tiers) {
    const hw = (w * scale) / 2;
    d += `M${cx - hw} ${y} L${cx + hw} ${y} L${cx + hw} ${y - h * scale} L${cx - hw} ${y - h * scale} Z `;
    y -= h * scale;
  }
  const hw = 9 * scale;
  d += `M${cx - hw} ${y} Q${cx} ${y - 26 * scale} ${cx + hw} ${y} Z M${cx - 1.5} ${y - 22 * scale} h3 v-9 h-3 Z`;
  return d;
}

export default function TripArt({
  id,
  destination,
  title,
  className = '',
}: {
  id: string;
  destination: string;
  title: string;
  className?: string;
}) {
  const uid = useId().replace(/:/g, '');
  const seed = hash(id || `${destination}${title}`);
  const r = rng(seed);
  const sky = SKIES[seed % SKIES.length];
  const scene = sceneFor(destination, title);
  const night = sky.name === 'senja';

  const horizon = scene === 'gunung' ? 175 : scene === 'budaya' ? 185 : 150;
  const sunX = 70 + r() * 260;
  const sunR = scene === 'pantai' ? 46 : 26 + r() * 8;
  const sunY = scene === 'pantai' ? horizon - 30 : 46 + r() * 40;

  const stars = night
    ? Array.from({ length: 14 }, () => ({ x: r() * W, y: r() * 80, s: 0.8 + r() * 1.2 }))
    : [];

  return (
    <svg
      viewBox={`0 0 ${W} ${H}`}
      preserveAspectRatio="xMidYMid slice"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      <defs>
        <linearGradient id={`${uid}s`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={sky.top} />
          <stop offset="1" stopColor={sky.bottom} />
        </linearGradient>
        <linearGradient id={`${uid}w`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={sky.mid} />
          <stop offset="1" stopColor={sky.near} />
        </linearGradient>
      </defs>
      <rect width={W} height={H} fill={`url(#${uid}s)`} />
      {stars.map((s, i) => (
        <circle key={i} cx={s.x} cy={s.y} r={s.s} fill="#fff" opacity={0.7} />
      ))}
      <circle cx={sunX} cy={sunY} r={sunR * 1.9} fill={sky.sun} opacity={0.22} />
      <circle cx={sunX} cy={sunY} r={sunR} fill={sky.sun} />

      {scene === 'gunung' && (
        <>
          <path d={peaks(r, 150, 40, 90, 6)} fill={sky.far} opacity={0.85} />
          <path d={peaks(r, 185, 40, 95, 5)} fill={sky.mid} />
          <path d={wave(212, 3, seed % 6)} fill={sky.near} />
        </>
      )}

      {scene === 'laut' && (
        <>
          <rect y={horizon} width={W} height={H - horizon} fill={`url(#${uid}w)`} />
          <path d={island(60 + r() * 60, horizon + 2, 46, 26)} fill={sky.near} opacity={0.9} />
          <path d={island(230 + r() * 40, horizon + 4, 34, 18)} fill={sky.mid} />
          <path d={island(340 + r() * 30, horizon + 3, 40, 22)} fill={sky.near} opacity={0.9} />
          <path d={wave(horizon + 40, 4, 1)} fill={sky.mid} opacity={0.7} />
          <path d={wave(horizon + 66, 5, 3)} fill={sky.near} />
        </>
      )}

      {scene === 'pantai' && (
        <>
          <rect y={horizon} width={W} height={H - horizon} fill={`url(#${uid}w)`} />
          <path d={wave(horizon + 22, 3, 2)} fill={sky.mid} opacity={0.8} />
          <path d={wave(horizon + 48, 4, 0)} fill={sky.far} opacity={0.5} />
          <path d={`M0 ${H} L0 ${horizon + 72} Q200 ${horizon + 52} ${W} ${horizon + 76} L${W} ${H} Z`} fill={night ? '#2b3f6e' : '#f0d9a6'} />
          <path d={`M${60 + r() * 40} ${H} q4 -60 16 -78 q-18 8 -34 4 q18 -6 20 -16 q8 10 26 8 q-14 6 -18 14 q10 22 4 68 Z`} fill={sky.near} />
        </>
      )}

      {scene === 'budaya' && (
        <>
          <path d={peaks(r, 160, 18, 40, 7)} fill={sky.far} opacity={0.7} />
          <rect y={horizon + 6} width={W} height={H - horizon} fill={sky.mid} />
          <path d={temple(150 + r() * 100, horizon + 8, 1.25)} fill={sky.near} />
          <path d={temple(60 + r() * 40, horizon + 12, 0.6)} fill={sky.near} opacity={0.8} />
          <path d={`M0 ${H} L0 ${horizon + 40} L${W} ${horizon + 40} L${W} ${H} Z`} fill={sky.near} />
        </>
      )}

      {scene === 'bawaan' && (
        <>
          <path d={peaks(r, 170, 10, 30, 4)} fill={sky.mid} />
          <path d={`M0 ${H} L0 200 Q100 170 200 200 T${W} 195 L${W} ${H} Z`} fill={sky.near} />
        </>
      )}
    </svg>
  );
}
