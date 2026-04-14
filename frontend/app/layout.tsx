import type { Metadata } from 'next';
import './globals.css';
import AppChrome from '@/components/AppChrome';
import { Toaster } from 'react-hot-toast';

export const metadata: Metadata = {
  title: 'TripNest | Modern Travel Booking',
  description: 'Distributed Travel Booking System with Event-Driven Architecture',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="h-full">
      <body className="flex min-h-screen flex-col antialiased">
        <AppChrome>{children}</AppChrome>
        <Toaster
          position="bottom-right"
          toastOptions={{
            style: {
              background: '#ffffff',
              color: '#0F172A',
              border: '1px solid #E2E8F0',
              borderRadius: '8px',
              fontWeight: 500,
            }
          }}
        />
      </body>
    </html>
  );
}
