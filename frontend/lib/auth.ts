/**
 * Penyimpanan sesi berbasis cookie.
 *
 * Sebelumnya token disimpan di localStorage. Itu punya dua masalah:
 * middleware Next.js tidak bisa membacanya (jadi tidak ada route guard
 * sama sekali), dan nilainya terbaca oleh skrip pihak ketiga mana pun.
 *
 * Cookie di sini masih bisa dibaca JavaScript karena axios perlu
 * menyisipkannya ke header Authorization. Langkah berikutnya adalah
 * cookie httpOnly plus route handler Next sebagai proxy -- itu masuk
 * daftar pekerjaan lanjutan, bukan sprint ini.
 */

export const TOKEN_COOKIE = 'tripnest_token';

export type SessionUser = {
  id: string;
  email: string;
  role: string;
};

type JwtPayload = {
  user_id?: string;
  role?: string;
  exp?: number;
};

const isBrowser = () => typeof document !== 'undefined';

export function setToken(token: string, maxAgeSeconds = 60 * 60 * 24): void {
  if (!isBrowser()) return;
  const secure = window.location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; Path=/; Max-Age=${maxAgeSeconds}; SameSite=Lax${secure}`;
}

export function getToken(): string | null {
  if (!isBrowser()) return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${TOKEN_COOKIE}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

export function clearToken(): void {
  if (!isBrowser()) return;
  document.cookie = `${TOKEN_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}

/**
 * Membaca payload JWT tanpa memverifikasi tanda tangannya.
 *
 * Ini HANYA untuk menentukan tampilan (mis. menyembunyikan menu admin).
 * Setiap keputusan yang benar-benar penting tetap diambil server, yang
 * memverifikasi tanda tangan token. Jangan pernah memercayai hasil
 * fungsi ini untuk otorisasi.
 */
export function decodeToken(token: string | null): JwtPayload | null {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3) return null;

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
    return JSON.parse(atob(padded)) as JwtPayload;
  } catch {
    return null;
  }
}

export function isExpired(payload: JwtPayload | null): boolean {
  if (!payload?.exp) return false;
  return payload.exp * 1000 <= Date.now();
}

export function isLoggedIn(): boolean {
  const payload = decodeToken(getToken());
  return payload !== null && !isExpired(payload);
}

export function currentRole(): string | null {
  const payload = decodeToken(getToken());
  if (!payload || isExpired(payload)) return null;
  return payload.role ?? 'user';
}

export function currentUserId(): string | null {
  const payload = decodeToken(getToken());
  if (!payload || isExpired(payload)) return null;
  return payload.user_id ?? null;
}

export function isAdmin(): boolean {
  return currentRole() === 'admin';
}
