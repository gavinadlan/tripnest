import type { Metadata } from 'next';
import './globals.css';
import Link from 'next/link';
import Navbar from '@/components/Navbar';

export const metadata: Metadata = {
  title: 'TripNest',
  description: 'Distributed Travel Booking System',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="antialiased bg-gray-50 text-gray-900">
        <Navbar />
        <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
          {children}
        </main>
      </body>
    </html>
  );
}
