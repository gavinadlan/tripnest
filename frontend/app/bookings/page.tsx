'use client';

import { useState, useEffect } from 'react';
import { bookingApi } from '@/lib/api';
import { useRouter } from 'next/navigation';

interface Booking {
    id: string;
    resource_id: string;
    total_amount: number;
    status: string;
    created_at: string;
}

export default function BookingsPage() {
    const router = useRouter();
    const [bookings, setBookings] = useState<Booking[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const fetchBookings = async () => {
        setLoading(true);
        setError('');

        try {
            const storedBookingIds = JSON.parse(localStorage.getItem('my_bookings') || '[]');

            if (storedBookingIds.length === 0) {
                setBookings([]);
                setLoading(false);
                return;
            }

            // Fetch each booking status individually since we don't have a bulk endpoint
            const bookingPromises = storedBookingIds.map((id: string) =>
                bookingApi.get(`/bookings/${id}`).catch(() => null)
            );

            const results = await Promise.all(bookingPromises);
            const validBookings = results
                .filter((res) => res && res.data)
                .map((res) => res?.data);

            // Sort by created_at descending
            validBookings.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

            setBookings(validBookings);
        } catch (err: any) {
            setError('Failed to fetch bookings');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        const token = localStorage.getItem('token');
        if (!token) {
            router.push('/login');
            return;
        }

        fetchBookings();
    }, [router]);

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold text-gray-900">My Bookings</h1>
                <button
                    onClick={fetchBookings}
                    className="px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 text-sm font-medium transition-colors"
                >
                    Refresh Status
                </button>
            </div>

            {error && <div className="text-red-500 bg-red-50 p-4 rounded-md text-center">{error}</div>}

            {loading ? (
                <div className="text-center text-gray-500 py-12">Loading bookings...</div>
            ) : (
                <div className="bg-white shadow-sm rounded-lg border border-gray-200 overflow-hidden">
                    {bookings.length === 0 ? (
                        <div className="p-12 text-center text-gray-500">
                            You haven't made any bookings yet.
                        </div>
                    ) : (
                        <ul className="divide-y divide-gray-200">
                            {bookings.map((booking) => (
                                <li key={booking.id} className="p-6 hover:bg-gray-50 transition-colors">
                                    <div className="flex items-center justify-between">
                                        <div className="space-y-1">
                                            <p className="text-sm font-medium text-gray-900">Booking ID: <span className="font-mono text-gray-500 font-normal">{booking.id}</span></p>
                                            <p className="text-sm text-gray-500">Resource: {booking.resource_id}</p>
                                            <p className="text-sm text-gray-500">Date: {new Date(booking.created_at).toLocaleString()}</p>
                                            <p className="text-sm font-medium text-gray-900">Total: ${booking.total_amount.toFixed(2)}</p>
                                        </div>
                                        <div>
                                            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium
                        ${booking.status === 'CONFIRMED' ? 'bg-green-100 text-green-800' :
                                                    booking.status === 'PENDING' ? 'bg-yellow-100 text-yellow-800' :
                                                        'bg-red-100 text-red-800'}`}
                                            >
                                                {booking.status}
                                            </span>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
            )}
        </div>
    );
}
