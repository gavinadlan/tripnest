'use client';

import { useState, useEffect, useCallback } from 'react';
import { searchApi, bookingApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import Spinner from '@/components/Spinner';
import ListingCard from '@/components/ListingCard';
import { CalendarDays, MapPin, DollarSign, Search } from 'lucide-react';

interface Listing {
    id: string;
    title: string;
    destination: string;
    price: number;
    date: string;
    available_slots: number;
}

interface SearchParams {
    destination: string;
    min_price: string;
    max_price: string;
    date: string;
}

export default function SearchPage() {
    const router = useRouter();
    const [listings, setListings] = useState<Listing[]>([]);
    const [loading, setLoading] = useState(true);
    const [bookingId, setBookingId] = useState<string | null>(null);

    const [searchParams, setSearchParams] = useState<SearchParams>({
        destination: '',
        min_price: '',
        max_price: '',
        date: ''
    });

    const fetchListings = useCallback(async () => {
        setLoading(true);
        try {
            const query = new URLSearchParams();
            if (searchParams.destination) query.append('destination', searchParams.destination);
            if (searchParams.min_price) query.append('min_price', searchParams.min_price);
            if (searchParams.max_price) query.append('max_price', searchParams.max_price);
            if (searchParams.date) query.append('date', searchParams.date);

            const response = await searchApi.get(`/search?${query.toString()}`);
            setListings(response.data.data || []);
        } catch (err: any) {
            toast.error('Failed to fetch destinations.');
        } finally {
            setLoading(false);
        }
    }, [searchParams]);

    useEffect(() => {
        fetchListings();
    }, []);

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        fetchListings();
    };

    const handleBook = async (listing: Listing) => {
        const userId = localStorage.getItem('user_id');
        const token = localStorage.getItem('token');

        if (!token || !userId) {
            toast('Please sign in to book a trip.', { icon: '👋' });
            router.push('/login');
            return;
        }

        setBookingId(listing.id);
        const loadingToast = toast.loading('Securing your spot...');

        try {
            const response = await bookingApi.post('/bookings', {
                user_id: userId,
                resource_id: listing.id,
                total_amount: listing.price
            });

            const existingBookings = JSON.parse(localStorage.getItem('my_bookings') || '[]');
            existingBookings.push(response.data.id);
            localStorage.setItem('my_bookings', JSON.stringify(existingBookings));

            toast.success('Booking confirmed successfully!', { id: loadingToast });
            router.push('/bookings');
        } catch (error: any) {
            toast.error(error.response?.data?.error || 'Failed to create booking', { id: loadingToast });
        } finally {
            setBookingId(null);
        }
    };

    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500 ease-out">
            {/* Header section */}
            <div className="text-center max-w-2xl mx-auto py-8">
                <h1 className="text-4xl font-extrabold text-gray-900 tracking-tight mb-4">
                    Find your next adventure.
                </h1>
                <p className="text-lg text-gray-500 font-medium">
                    Discover hand-picked spots from around the world. Secure your booking within seconds.
                </p>
            </div>

            {/* Modern Search Form */}
            <div className="bg-white p-6 md:p-8 rounded-2xl shadow-sm border border-gray-100 mb-10 transition-shadow hover:shadow-md">
                <form onSubmit={handleSearch} className="grid grid-cols-1 gap-5 md:grid-cols-12 items-end">
                    <div className="md:col-span-4 relative group">
                        <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Location</label>
                        <div className="relative">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <MapPin className="h-5 w-5 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                            </div>
                            <input
                                type="text"
                                className="pl-10 block w-full outline-none border-b-2 border-gray-200 focus:border-indigo-600 focus:ring-0 py-2 transition-colors sm:text-sm bg-transparent font-medium placeholder-gray-400"
                                value={searchParams.destination}
                                onChange={(e) => setSearchParams({ ...searchParams, destination: e.target.value })}
                                placeholder="Where to?"
                            />
                        </div>
                    </div>

                    <div className="md:col-span-2 relative group">
                        <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Min Price</label>
                        <div className="relative">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <DollarSign className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                            </div>
                            <input
                                type="number"
                                className="pl-8 block w-full outline-none border-b-2 border-gray-200 focus:border-indigo-600 focus:ring-0 py-2 transition-colors sm:text-sm bg-transparent font-medium"
                                value={searchParams.min_price}
                                onChange={(e) => setSearchParams({ ...searchParams, min_price: e.target.value })}
                                placeholder="0"
                            />
                        </div>
                    </div>

                    <div className="md:col-span-2 relative group">
                        <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Max Price</label>
                        <div className="relative">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <DollarSign className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                            </div>
                            <input
                                type="number"
                                className="pl-8 block w-full outline-none border-b-2 border-gray-200 focus:border-indigo-600 focus:ring-0 py-2 transition-colors sm:text-sm bg-transparent font-medium"
                                value={searchParams.max_price}
                                onChange={(e) => setSearchParams({ ...searchParams, max_price: e.target.value })}
                                placeholder="Any"
                            />
                        </div>
                    </div>

                    <div className="md:col-span-2 relative group">
                        <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Date</label>
                        <div className="relative">
                            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                <CalendarDays className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                            </div>
                            <input
                                type="date"
                                className="pl-9 block w-full outline-none border-b-2 border-gray-200 focus:border-indigo-600 focus:ring-0 py-2 transition-colors sm:text-sm bg-transparent font-medium text-gray-700"
                                value={searchParams.date}
                                onChange={(e) => setSearchParams({ ...searchParams, date: e.target.value })}
                            />
                        </div>
                    </div>

                    <div className="md:col-span-2 flex items-end">
                        <button
                            type="submit"
                            className="w-full bg-indigo-600 rounded-xl shadow-md border border-transparent shadow-indigo-200 py-2.5 px-4 inline-flex justify-center items-center gap-2 text-sm font-bold text-white hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-all active:scale-[0.98]"
                        >
                            <Search className="h-4 w-4" />
                            Search
                        </button>
                    </div>
                </form>
            </div>

            {/* Results grid */}
            <div className="min-h-[400px]">
                {loading ? (
                    <div className="flex flex-col items-center pt-24 space-y-4">
                        <Spinner className="w-10 h-10 text-indigo-600" />
                        <p className="text-gray-500 font-medium">Finding the best destinations...</p>
                    </div>
                ) : (
                    <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3">
                        {listings.map((listing) => (
                            <ListingCard
                                key={listing.id}
                                listing={listing}
                                onBook={handleBook}
                                isLoading={bookingId === listing.id}
                            />
                        ))}
                        {listings.length === 0 && (
                            <div className="col-span-full pt-16 pb-24 text-center">
                                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-100 mb-4">
                                    <Search className="h-8 w-8 text-gray-400" />
                                </div>
                                <h3 className="text-xl font-bold text-gray-900 mb-2">No destinations found</h3>
                                <p className="text-gray-500 max-w-sm mx-auto">
                                    We couldn't find any trips matching your current search criteria. Try adjusting your dates or price range.
                                </p>
                                <button
                                    onClick={() => setSearchParams({ destination: '', min_price: '', max_price: '', date: '' })}
                                    className="mt-6 text-sm font-semibold text-indigo-600 hover:text-indigo-500"
                                >
                                    Clear all filters
                                </button>
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
}
