import axios from 'axios';
import { BOOKING_SERVICE_URL, PAYMENT_SERVICE_URL, SEARCH_SERVICE_URL } from './api';

export const ADMIN_BOOKING_API_URL = process.env.NEXT_PUBLIC_ADMIN_BOOKING_API_URL || BOOKING_SERVICE_URL;
export const ADMIN_PAYMENT_API_URL = process.env.NEXT_PUBLIC_ADMIN_PAYMENT_API_URL || PAYMENT_SERVICE_URL;
export const ADMIN_INVENTORY_API_URL = process.env.NEXT_PUBLIC_ADMIN_INVENTORY_API_URL || 'http://localhost:8084';
export const ADMIN_TRIP_API_URL = process.env.NEXT_PUBLIC_ADMIN_TRIP_API_URL || SEARCH_SERVICE_URL;

const attachAuth = (client: ReturnType<typeof axios.create>) => {
  client.interceptors.request.use((config) => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });
};

export const adminBookingApi = axios.create({ baseURL: ADMIN_BOOKING_API_URL });
export const adminPaymentApi = axios.create({ baseURL: ADMIN_PAYMENT_API_URL });
export const adminInventoryApi = axios.create({ baseURL: ADMIN_INVENTORY_API_URL });
export const adminTripApi = axios.create({ baseURL: ADMIN_TRIP_API_URL });

[adminBookingApi, adminPaymentApi, adminInventoryApi, adminTripApi].forEach((api) => {
  attachAuth(api);
});
