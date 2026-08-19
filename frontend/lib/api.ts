import axios, { AxiosInstance } from 'axios';
import { clearToken, getToken } from './auth';

export const USER_SERVICE_URL = process.env.NEXT_PUBLIC_USER_SERVICE_URL || 'http://localhost:8080';
export const BOOKING_SERVICE_URL = process.env.NEXT_PUBLIC_BOOKING_SERVICE_URL || 'http://localhost:8081';
export const PAYMENT_SERVICE_URL = process.env.NEXT_PUBLIC_PAYMENT_SERVICE_URL || 'http://localhost:8082';
export const SEARCH_SERVICE_URL = process.env.NEXT_PUBLIC_SEARCH_SERVICE_URL || 'http://localhost:8083';
export const INVENTORY_SERVICE_URL = process.env.NEXT_PUBLIC_INVENTORY_SERVICE_URL || 'http://localhost:8084';

/**
 * Memasang token pada setiap request dan menangani 401 di satu tempat.
 *
 * Semua klien memakai helper ini, termasuk klien admin. Sebelumnya logika
 * ini disalin di dua file dan halaman pembayaran memakai fetch mentah
 * tanpa header Authorization sama sekali.
 */
function withAuth(client: AxiosInstance): AxiosInstance {
  client.interceptors.request.use((config) => {
    const token = getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });

  client.interceptors.response.use(
    (response) => response,
    (error) => {
      const status = error?.response?.status;
      // Token kedaluwarsa atau dicabut: bersihkan sesi supaya UI tidak
      // terus menampilkan keadaan "sudah login" yang keliru.
      if (status === 401 && typeof window !== 'undefined') {
        clearToken();
        if (!window.location.pathname.startsWith('/login')) {
          window.location.href = `/login?next=${encodeURIComponent(window.location.pathname)}`;
        }
      }
      return Promise.reject(error);
    }
  );

  return client;
}

export const userApi = withAuth(axios.create({ baseURL: USER_SERVICE_URL }));
export const bookingApi = withAuth(axios.create({ baseURL: BOOKING_SERVICE_URL }));
export const paymentApi = withAuth(axios.create({ baseURL: PAYMENT_SERVICE_URL }));
export const searchApi = withAuth(axios.create({ baseURL: SEARCH_SERVICE_URL }));
export const inventoryApi = withAuth(axios.create({ baseURL: INVENTORY_SERVICE_URL }));
