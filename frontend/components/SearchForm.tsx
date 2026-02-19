import { Search, MapPin, DollarSign, Calendar } from 'lucide-react';

interface SearchParams {
    destination: string;
    min_price: string;
    max_price: string;
    date: string;
}

interface SearchFormProps {
    searchParams: SearchParams;
    setSearchParams: (params: SearchParams) => void;
    onSearch: (e: React.FormEvent) => void;
}

export default function SearchForm({ searchParams, setSearchParams, onSearch }: SearchFormProps) {
    return (
        <div className="bg-white p-6 md:p-8 rounded-2xl shadow-sm border border-gray-100 mb-10 transition-shadow hover:shadow-md">
            <form onSubmit={onSearch} className="grid grid-cols-1 gap-5 md:grid-cols-12 items-end">
                <div className="md:col-span-4 relative group">
                    <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Destination</label>
                    <div className="relative">
                        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                            <MapPin className="h-5 w-5 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                        </div>
                        <input
                            type="text"
                            className="pl-10 block w-full outline-none border-b-2 border-gray-200 focus:border-indigo-600 focus:ring-0 py-2 transition-colors sm:text-sm bg-transparent font-medium placeholder-gray-400"
                            value={searchParams.destination}
                            onChange={(e) => setSearchParams({ ...searchParams, destination: e.target.value })}
                            placeholder="Where are you going?"
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
                    <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">Check-in</label>
                    <div className="relative">
                        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                            <Calendar className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
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
                        className="w-full bg-indigo-600 rounded-xl shadow-md shadow-indigo-200 py-3 px-4 inline-flex justify-center items-center gap-2 text-sm font-bold text-white hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-all active:scale-[0.98]"
                    >
                        <Search className="h-4 w-4" />
                        Search
                    </button>
                </div>
            </form>
        </div>
    );
}
