import { MapPin, CalendarDays, Users, Tag } from 'lucide-react';

interface Listing {
    id: string;
    title: string;
    destination: string;
    price: number;
    date: string;
    available_slots: number;
}

interface ListingCardProps {
    listing: Listing;
    onBook: (listing: Listing) => void;
    isLoading?: boolean;
}

export default function ListingCard({ listing, onBook, isLoading = false }: ListingCardProps) {
    // Simple fake image based on destination string length to keep UI looking somewhat populated
    const imgNum = (listing.destination.length % 5) + 1;
    const imagePlaceholder = `https://images.unsplash.com/photo-1507525428034-b723cf961d3e?ixlib=rb-4.0.3&auto=format&fit=crop&w=400&q=80`;

    return (
        <div className="bg-white overflow-hidden shadow-sm hover:shadow-xl rounded-2xl border border-gray-100 transition-all duration-300 flex flex-col group">
            <div className="relative h-48 overflow-hidden bg-gray-200">
                <img
                    src={imagePlaceholder}
                    alt={listing.title}
                    className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500 ease-in-out"
                    style={{ filter: `hue-rotate(${imgNum * 45}deg)` }} // Fake variety 
                />
                <div className="absolute top-4 right-4 bg-white/95 backdrop-blur-sm px-3 py-1.5 rounded-full shadow-sm">
                    <span className="text-sm font-bold text-gray-900 flex items-center gap-1">
                        <Tag className="h-3 w-3 text-indigo-500" />
                        ${listing.price.toFixed(0)}
                    </span>
                </div>
            </div>

            <div className="p-6 flex-1 flex flex-col">
                <h3 className="text-xl font-bold text-gray-900 mb-4 line-clamp-1 group-hover:text-indigo-600 transition-colors">
                    {listing.title}
                </h3>

                <div className="space-y-3 mt-auto">
                    <div className="flex items-center text-sm text-gray-600">
                        <MapPin className="h-4 w-4 mr-3 text-gray-400" />
                        <span className="font-medium text-gray-800">{listing.destination}</span>
                    </div>

                    <div className="flex items-center text-sm text-gray-600">
                        <CalendarDays className="h-4 w-4 mr-3 text-gray-400" />
                        <span className="text-gray-700">{listing.date}</span>
                    </div>

                    <div className="flex items-center text-sm">
                        <Users className="h-4 w-4 mr-3 text-gray-400" />
                        <span className={`font-medium ${listing.available_slots > 5 ? 'text-green-600' : 'text-orange-500'}`}>
                            {listing.available_slots} slots available
                        </span>
                    </div>
                </div>

                <div className="mt-6 pt-6 border-t border-gray-100">
                    <button
                        onClick={() => onBook(listing)}
                        disabled={isLoading || listing.available_slots <= 0}
                        className={`w-full flex justify-center items-center px-4 py-3 border border-transparent text-sm font-bold rounded-xl text-white transition-all active:scale-[0.98] ${isLoading || listing.available_slots <= 0
                                ? 'bg-gray-300 cursor-not-allowed'
                                : 'bg-gray-900 hover:bg-gray-800 shadow-sm hover:shadow-md'
                            }`}
                    >
                        {isLoading ? 'Booking...' : listing.available_slots > 0 ? 'Book Trip Now' : 'Sold Out'}
                    </button>
                </div>
            </div>
        </div>
    );
}
