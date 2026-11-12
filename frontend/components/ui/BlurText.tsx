'use client';

import { motion, useReducedMotion } from 'motion/react';

/**
 * Judul yang muncul kata demi kata dari kabur ke tajam. Pola "Blur Text"
 * ala React Bits, ditulis ulang dengan motion. Tanpa animasi bila
 * prefers-reduced-motion aktif.
 */
export default function BlurText({ text, className = '' }: { text: string; className?: string }) {
  const reduce = useReducedMotion();
  if (reduce) return <span className={className}>{text}</span>;
  return (
    <span className={className} aria-label={text}>
      {text.split(' ').map((w, i) => (
        <motion.span
          key={i}
          aria-hidden
          className="inline-block"
          initial={{ opacity: 0, filter: 'blur(10px)', y: 14 }}
          animate={{ opacity: 1, filter: 'blur(0px)', y: 0 }}
          transition={{ delay: i * 0.07, duration: 0.55, ease: 'easeOut' }}
        >
          {w}
          {' '}
        </motion.span>
      ))}
    </span>
  );
}
