export default function Footer() {
    return (
        <footer className="bg-white border-t border-gray-100 mt-auto">
            <div className="max-w-7xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
                <div className="md:flex md:items-center md:justify-between text-center md:text-left">
                    <div className="flex justify-center md:justify-start space-x-6 md:order-2">
                        <span className="text-sm font-medium text-gray-400 hover:text-gray-500 cursor-pointer transition-colors">
                            Privacy Policy
                        </span>
                        <span className="text-sm font-medium text-gray-400 hover:text-gray-500 cursor-pointer transition-colors">
                            Terms of Service
                        </span>
                    </div>
                    <p className="mt-8 text-sm text-gray-400 font-medium md:mt-0 md:order-1">
                        &copy; {new Date().getFullYear()} TripNest, Inc. All rights reserved.
                    </p>
                </div>
            </div>
        </footer>
    );
}
