"use client";

import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { PlaneTakeoff, LogOut, Menu, X } from "lucide-react";

export default function Navbar() {
    const router = useRouter();
    const pathname = usePathname();

    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [mounted, setMounted] = useState(false);
    const [isLoggedIn, setIsLoggedIn] = useState(false);

    useEffect(() => {
        setMounted(true);

        const syncAuth = () => {
            const token = localStorage.getItem("token");
            setIsLoggedIn(!!token);
        };

        syncAuth();
        window.addEventListener("storage", syncAuth);

        return () => window.removeEventListener("storage", syncAuth);
    }, []);

    const handleLogout = () => {
        localStorage.removeItem("token");
        localStorage.removeItem("user_id");
        setIsLoggedIn(false);
        setMobileMenuOpen(false);
        router.push("/login");
    };

    const navLinks = [
        { name: "Trip", href: "/" },
        { name: "Pesanan Saya", href: "/my-bookings" },
    ];

    // 🔥 CRITICAL FIX: jangan render sebelum mounted
    if (!mounted) {
        return (
            <nav className="sticky top-0 z-50 border-b border-[var(--color-border)] bg-white/95 backdrop-blur-md">
                <div className="mx-auto max-w-7xl px-4 h-16 flex items-center">
                    <span className="text-lg font-bold">TripNest</span>
                </div>
            </nav>
        );
    }

    return (
        <nav className="sticky top-0 z-50 border-b border-[var(--color-border)] bg-white/95 backdrop-blur-md">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="flex h-16 justify-between">
                    <div className="flex items-center">
                        <Link href="/" className="flex items-center gap-2">
                            <div className="rounded-lg bg-[var(--color-primary)] p-2">
                                <PlaneTakeoff className="h-5 w-5 text-white" />
                            </div>
                            <span className="text-xl font-bold">
                                TripNest
                            </span>
                        </Link>

                        <div className="hidden sm:ml-10 sm:flex sm:space-x-8">
                            {navLinks.map((link) => (
                                <Link
                                    key={link.name}
                                    href={link.href}
                                    className={`inline-flex items-center border-b-2 px-1 pt-1 text-sm font-medium ${
                                        pathname === link.href
                                            ? "border-indigo-500 text-indigo-600"
                                            : "border-transparent text-gray-500 hover:text-gray-700"
                                    }`}
                                >
                                    {link.name}
                                </Link>
                            ))}
                        </div>
                    </div>

                    {/* 🔥 FIX HYDRATION */}
                    <div className="hidden sm:flex sm:items-center">
                        {isLoggedIn ? (
                            <button
                                onClick={handleLogout}
                                className="ml-4 flex items-center gap-2 rounded-lg border px-4 py-2 text-sm"
                            >
                                <LogOut className="h-4 w-4" />
                                Keluar
                            </button>
                        ) : (
                            <div className="flex items-center space-x-4">
                                <Link href="/login">Masuk</Link>
                                <Link href="/register">Daftar</Link>
                            </div>
                        )}
                    </div>

                    <div className="flex items-center sm:hidden">
                        <button
                            onClick={() => setMobileMenuOpen((v) => !v)}
                        >
                            {mobileMenuOpen ? <X /> : <Menu />}
                        </button>
                    </div>
                </div>
            </div>

            {mobileMenuOpen && (
                <div className="border-t bg-white sm:hidden">
                    {navLinks.map((link) => (
                        <Link
                            key={link.name}
                            href={link.href}
                            onClick={() => setMobileMenuOpen(false)}
                            className="block px-4 py-2"
                        >
                            {link.name}
                        </Link>
                    ))}

                    <div className="border-t px-4 py-3">
                        {isLoggedIn ? (
                            <button onClick={handleLogout}>
                                Keluar
                            </button>
                        ) : (
                            <>
                                <Link href="/login">Masuk</Link>
                                <Link href="/register">Daftar</Link>
                            </>
                        )}
                    </div>
                </div>
            )}
        </nav>
    );
}