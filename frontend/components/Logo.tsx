/* eslint-disable @next/next/no-img-element */
import Link from 'next/link';

export default function Logo({ dark = false, className = 'h-9', href = '/' }: { dark?: boolean; className?: string; href?: string | null }) {
  const img = (
    <img src={dark ? '/tripnest-logo-dark.svg' : '/tripnest-logo.svg'} alt="TripNest" className={`${className} w-auto`} />
  );
  return href ? (
    <Link href={href} aria-label="TripNest, ke beranda" className="inline-flex">
      {img}
    </Link>
  ) : (
    img
  );
}
