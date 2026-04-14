'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function LegacyBookingsPage() {
  const router = useRouter();
  useEffect(() => {
    router.replace('/my-bookings');
  }, [router]);

  return null;
}
