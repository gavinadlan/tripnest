/**
 * Klien admin kini hanya alias dari klien biasa.
 *
 * Otorisasi ditentukan role di dalam JWT dan diperiksa server, bukan oleh
 * base URL atau flag di browser. Modul ini dipertahankan supaya halaman
 * admin yang sudah ada tidak perlu diubah semuanya sekaligus.
 */
export {
  bookingApi as adminBookingApi,
  paymentApi as adminPaymentApi,
  inventoryApi as adminInventoryApi,
  searchApi as adminTripApi,
} from './api';
