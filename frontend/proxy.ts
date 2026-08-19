import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const TOKEN_COOKIE = 'tripnest_token';

/** Halaman yang butuh login. */
const PROTECTED = ['/my-bookings', '/booking', '/payment', '/booking-status'];

/** Halaman yang butuh role admin. */
const ADMIN_ONLY = ['/admin'];

/**
 * Membaca payload JWT tanpa verifikasi tanda tangan.
 *
 * Middleware hanya mengatur navigasi; token palsu paling banter membuat
 * halaman admin ter-render, lalu setiap panggilan API di dalamnya ditolak
 * server yang memverifikasi tanda tangannya. Verifikasi sungguhan di sini
 * butuh secret di sisi Next, dan itu bukan tempatnya.
 */
function readPayload(token: string): { role?: string; exp?: number } | null {
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
    return JSON.parse(atob(padded));
  } catch {
    return null;
  }
}

export default function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const needsAuth = PROTECTED.some((p) => pathname.startsWith(p));
  const needsAdmin = ADMIN_ONLY.some((p) => pathname.startsWith(p));

  // Halaman login admin harus tetap bisa dibuka tanpa token.
  if (pathname === '/admin/login' || (!needsAuth && !needsAdmin)) {
    return NextResponse.next();
  }

  const token = request.cookies.get(TOKEN_COOKIE)?.value;
  const payload = token ? readPayload(token) : null;
  const expired = payload?.exp ? payload.exp * 1000 <= Date.now() : false;

  if (!payload || expired) {
    const target = needsAdmin ? '/admin/login' : '/login';
    const url = new URL(target, request.url);
    url.searchParams.set('next', pathname);
    return NextResponse.redirect(url);
  }

  if (needsAdmin && payload.role !== 'admin') {
    return NextResponse.redirect(new URL('/', request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/my-bookings/:path*', '/booking/:path*', '/payment/:path*', '/booking-status/:path*', '/admin/:path*'],
};
