import type { Metadata } from 'next';
import '@fontsource-variable/bricolage-grotesque';
import '@fontsource-variable/plus-jakarta-sans';
import './globals.css';
import AppChrome from '@/components/AppChrome';
import { Toaster } from 'react-hot-toast';

export const metadata: Metadata = {
  title: { default: 'TripNest | Booking Perjalanan', template: '%s | TripNest' },
  description: 'TripNest: booking perjalanan berbasis arsitektur event-driven. Cari trip, pesan kursi, dan bayar dengan aman.',
  icons: { icon: '/favicon.svg', apple: '/apple-touch-icon.png' },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id" className="h-full">
      <body className="flex min-h-screen flex-col antialiased">
        <AppChrome>{children}</AppChrome>
        <Toaster
          position="bottom-right"
          toastOptions={{
            style: {
              background: '#fffdf8',
              color: '#0a1b2e',
              border: '1px solid #e3d9c4',
              borderRadius: '12px',
              fontWeight: 600,
            },
          }}
        />
      </body>
    </html>
  );
}
