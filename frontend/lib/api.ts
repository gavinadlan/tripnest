import axios from 'axios';

export const USER_SERVICE_URL = process.env.NEXT_PUBLIC_USER_SERVICE_URL || 'http://localhost:8080';
export const BOOKING_SERVICE_URL = process.env.NEXT_PUBLIC_BOOKING_SERVICE_URL || 'http://localhost:8081';
export const SEARCH_SERVICE_URL = process.env.NEXT_PUBLIC_SEARCH_SERVICE_URL || 'http://localhost:8083';

export const userApi = axios.create({
  baseURL: USER_SERVICE_URL,
});

export const bookingApi = axios.create({
  baseURL: BOOKING_SERVICE_URL,
});

export const searchApi = axios.create({
  baseURL: SEARCH_SERVICE_URL,
});

// Adding an interceptor to inject JWT into bookingApi
bookingApi.interceptors.request.use((config) => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
