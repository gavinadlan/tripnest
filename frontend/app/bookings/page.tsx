'use client';

import { useState, useEffect } from 'react';
import { bookingApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { Loader2, RefreshCcw, Ticket, AlertCircle, CheckCircle2, Clock } from 'lucide-react';
import toast from 'react-hot-toast';
import Spinner from '@/components/Spinner';

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

    const fetchBookings = async (showToast = false) => {
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
            if (showToast) toast.success('Booking status refreshed');
        } catch (err: any) {
            setError('Failed to fetch bookings');
            if (showToast) toast.error('Failed to refresh status');
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

    const StatusBadge = ({ status }: { status: string }) => {
        switch (status) {
            case 'CONFIRMED':
                return (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-bold bg-green-100 text-green-800 border border-green-200 shadow-sm">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        Confirmed
                    </span>
                );
            case 'PENDING':
                return (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-bold bg-amber-100 text-amber-800 border border-amber-200 shadow-sm">
                        <Clock className="w-3.5 h-3.5 animate-pulse" />
                        Processing
                    </span>
                );
            case 'CANCELLED':
            default:
                return (
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-bold bg-red-100 text-red-800 border border-red-200 shadow-sm">
                        <AlertCircle className="w-3.5 h-3.5" />
                        Cancelled
                    </span>
                );
        }
    };

    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500 ease-out max-w-5xl mx-auto">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 py-4 border-b border-gray-100">
                <div>
                    <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight">Your Trips</h1>
                    <p className="text-gray-500 mt-1 font-medium">Manage and view the status of all your bookings</p>
                </div>

                <button
                    onClick={() => fetchBookings(true)}
                    disabled={loading}
                    className="inline-flex items-center gap-2 px-5 py-2.5 bg-white border border-gray-200 text-gray-700 rounded-xl hover:bg-gray-50 hover:text-indigo-600 focus:ring-4 focus:ring-gray-100 text-sm font-semibold transition-all disabled:opacity-50 shadow-sm active:scale-95"
                >
                    {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCcw className="w-4 h-4" />}
                    Refresh
                </button>
            </div>

            {error ? (
                <div className="flex flex-col items-center justify-center p-12 bg-red-50 rounded-2xl border border-red-100">
                    <AlertCircle className="w-12 h-12 text-red-400 mb-4" />
                    <h3 className="text-lg font-bold text-red-800">Connection Error</h3>
                    <p className="text-red-600 mt-2 text-center max-w-sm">We're having trouble reaching our booking system right now.</p>
                    <button
                        onClick={() => fetchBookings()}
                        className="mt-6 px-4 py-2 bg-red-100 text-red-700 font-medium rounded-lg hover:bg-red-200 transition-colors"
                    >
                        Try Again
                    </button>
                </div>
            ) : loading && bookings.length === 0 ? (
                <div className="flex flex-col justify-center items-center py-24 space-y-4 bg-white rounded-3xl border border-gray-100 shadow-sm">
                    <Spinner className="w-10 h-10 text-indigo-600" />
                    <p className="text-gray-500 font-medium">Retrieving your itinerary...</p>
                </div>
            ) : bookings.length === 0 ? (
                <div className="flex flex-col justify-center items-center py-24 bg-white rounded-3xl border border-gray-100 shadow-sm text-center px-4">
                    <div className="w-20 h-20 bg-indigo-50 rounded-full flex items-center justify-center mb-6">
                        <Ticket className="w-10 h-10 text-indigo-500" />
                    </div>
                    <h3 className="text-2xl font-bold text-gray-900 mb-2">No bookings yet</h3>
                    <p className="text-gray-500 max-w-sm mb-8 text-lg">
                        You haven't planned any trips. Find your next great destination today!
                    </p>
                    <button
                        onClick={() => router.push('/search')}
                        className="px-8 py-3.5 bg-indigo-600 text-white rounded-xl font-bold hover:bg-indigo-700 hover:shadow-lg hover:shadow-indigo-200 transition-all active:scale-95"
                    >
                        Explore Destinations
                    </button>
                </div>
            ) : (
                <div className="grid grid-cols-1 gap-4">
                    {bookings.map((booking) => (
                        <div
                            key={booking.id}
                            className="group bg-white rounded-2xl border border-gray-100 p-6 sm:p-8 hover:shadow-xl hover:border-indigo-100 transition-all duration-300 relative overflow-hidden flex flex-col sm:flex-row sm:items-center justify-between gap-6"
                        >
                            {/* Subtle top border color based on status */}
                            <div className={`absolute top-0 left-0 right-0 h-1 ${booking.status === 'CONFIRMED' ? 'bg-green-400' :
                                    booking.status === 'PENDING' ? 'bg-amber-400' : 'bg-red-400'
                                }`} />

                            <div className="space-y-4 pt-2">
                                <div>
                                    <div className="flex items-center gap-3 mb-1">
                                        <span className="text-xs font-bold text-gray-400 uppercase tracking-wider">Booking Reference</span>
                                        <span className="font-mono text-xs bg-gray-100 px-2 py-1 rounded text-gray-600">{booking.id.split('-')[0]}***</span>
                                    </div>
                                    <h3 className="text-xl font-bold text-gray-900 group-hover:text-indigo-600 transition-colors">
                                        {booking.resource_id}
                                    </h3>
                                </div>

                                <div className="flex flex-wrap items-center gap-4 text-sm font-medium text-gray-500">
                                    <div className="flex items-center gap-1.5">
                                        <CalendarDays className="w-4 h-4 text-gray-400" />
                                        {new Date(booking.created_at).toLocaleDateString('en-US', {
                                            year: 'numeric', month: 'short', day: 'numeric'
                                        })}
                                    </div>
                                    <div className="flex items-center gap-1.5">
                                        <Clock className="w-4 h-4 text-gray-400" />
                                        {new Date(booking.created_at).toLocaleTimeString('en-US', {
                                            hour: '2-digit', minute: '2-digit'
                                        })}
                                    </div>
                                </div>
                            </div>

                            <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-center gap-4 border-t sm:border-t-0 sm:border-l border-gray-100 pt-4 sm:pt-0 sm:pl-8 mt-2 sm:mt-0">
                                <div className="text-2xl font-black text-gray-900">
                                    ${booking.total_amount.toFixed(0)}
                                </div>
                                <StatusBadge status={booking.status} />
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
