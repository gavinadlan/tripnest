'use client';

import { useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';

/**
 * Grid yang menggeser satu sorotan lembut ke kartu yang sedang di-hover
 * atau difokus. Pola "Card Hover Effect" ala Aceternity UI, ditulis ulang
 * dengan motion (layoutId) dan warna amber Samudra.
 */
export default function HoverGrid({ items, className = '' }: { items: { key: string; node: React.ReactNode }[]; className?: string }) {
  const [active, setActive] = useState<string | null>(null);
  return (
    <div className={className} onMouseLeave={() => setActive(null)}>
      {items.map(({ key, node }) => (
        <div
          key={key}
          className="relative p-2"
          onMouseEnter={() => setActive(key)}
          onFocus={() => setActive(key)}
          onBlur={() => setActive(null)}
        >
          <AnimatePresence>
            {active === key && (
              <motion.span
                aria-hidden
                layoutId="hover-sorot"
                className="absolute inset-0 rounded-3xl bg-amber/30"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1, transition: { duration: 0.15 } }}
                exit={{ opacity: 0, transition: { duration: 0.15, delay: 0.1 } }}
              />
            )}
          </AnimatePresence>
          <div className="relative h-full">{node}</div>
        </div>
      ))}
    </div>
  );
}
