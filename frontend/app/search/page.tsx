'use client';

import { useState, useEffect } from 'react';
import { searchApi, bookingApi } from '@/lib/api';
import { useRouter } from 'next/navigation';

interface Listing {
    id: string;
    title: string;
    destination: string;
    price: number;
    date: string;
    available_slots: number;
}

export default function SearchPage() {
    const router = useRouter();
    const [listings, setListings] = useState<Listing[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const [searchParams, setSearchParams] = useState({
        destination: '',
        min_price: '',
        max_price: '',
        date: ''
    });

    const fetchListings = async () => {
        setLoading(true);
        setError('');
        try {
            const query = new URLSearchParams();
            if (searchParams.destination) query.append('destination', searchParams.destination);
            if (searchParams.min_price) query.append('min_price', searchParams.min_price);
            if (searchParams.max_price) query.append('max_price', searchParams.max_price);
            if (searchParams.date) query.append('date', searchParams.date);

            const response = await searchApi.get(`/search?${query.toString()}`);
            setListings(response.data.data || []);
        } catch (err: any) {
            setError('Failed to fetch listings');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchListings();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault();
        fetchListings();
    };

    const handleBook = async (listing: Listing) => {
        const userId = localStorage.getItem('user_id');
        const token = localStorage.getItem('token');

        if (!token || !userId) {
            router.push('/login');
            return;
        }

        try {
            await bookingApi.post('/bookings', {
                user_id: userId,
                resource_id: listing.id,
                total_amount: listing.price
            });
            alert('Booking created successfully!');
            router.push('/bookings');
        } catch (error: any) {
            alert(error.response?.data?.error || 'Failed to create booking');
        }
    };

    return (
        <div className="space-y-6">
            <div className="bg-white p-6 rounded-lg shadow-sm">
                <form onSubmit={handleSearch} className="grid grid-cols-1 gap-4 sm:grid-cols-5">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Destination</label>
                        <input
                            type="text"
                            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm px-3 py-2 border"
                            value={searchParams.destination}
                            onChange={(e) => setSearchParams({ ...searchParams, destination: e.target.value })}
                            placeholder="e.g. Paris"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Min Price</label>
                        <input
                            type="number"
                            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm px-3 py-2 border"
                            value={searchParams.min_price}
                            onChange={(e) => setSearchParams({ ...searchParams, min_price: e.target.value })}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Max Price</label>
                        <input
                            type="number"
                            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm px-3 py-2 border"
                            value={searchParams.max_price}
                            onChange={(e) => setSearchParams({ ...searchParams, max_price: e.target.value })}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Date</label>
                        <input
                            type="date"
                            className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm px-3 py-2 border"
                            value={searchParams.date}
                            onChange={(e) => setSearchParams({ ...searchParams, date: e.target.value })}
                        />
                    </div>
                    <div className="flex items-end">
                        <button
                            type="submit"
                            className="w-full bg-blue-600 border border-transparent rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                        >
                            Search
                        </button>
                    </div>
                </form>
            </div>

            {error && <div className="text-red-500 text-center">{error}</div>}

            {loading ? (
                <div className="text-center text-gray-500">Loading listings...</div>
            ) : (
                <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {listings.map((listing) => (
                        <div key={listing.id} className="bg-white overflow-hidden shadow-sm rounded-lg border border-gray-200">
                            <div className="p-6">
                                <h3 className="text-xl font-semibold text-gray-900 mb-2">{listing.title}</h3>
                                <div className="space-y-2 text-sm text-gray-500">
                                    <p><span className="font-medium text-gray-700">Destination:</span> {listing.destination}</p>
                                    <p><span className="font-medium text-gray-700">Price:</span> ${listing.price.toFixed(2)}</p>
                                    <p><span className="font-medium text-gray-700">Date:</span> {listing.date}</p>
                                    <p><span className="font-medium text-gray-700">Available Slots:</span> {listing.available_slots}</p>
                                </div>
                                <div className="mt-6">
                                    <button
                                        onClick={() => handleBook(listing)}
                                        className="w-full flex justify-center items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700"
                                    >
                                        Book Now
                                    </button>
                                </div>
                            </div>
                        </div>
                    ))}
                    {listings.length === 0 && !error && (
                        <div className="col-span-full text-center text-gray-500 py-12">
                            No listings found matching your criteria.
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
