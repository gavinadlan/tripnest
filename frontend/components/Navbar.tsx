'use client';

import Link from "next/link";
import { useRouter, usePathname } from "next/navigation";
import { useCallback, useState, useSyncExternalStore } from "react";
import { PlaneTakeoff, LogOut, Menu, X } from "lucide-react";
import { clearToken, isAdmin, isLoggedIn } from "@/lib/auth";

/**
 * Status sesi dibaca lewat useSyncExternalStore, bukan useEffect + setState.
 *
 * Cookie adalah state di luar React. useSyncExternalStore menangani
 * perbedaan render server dan client tanpa flag `mounted`, dan tidak
 * memicu render berantai seperti setState di dalam effect.
 */
function subscribe(onChange: () => void) {
    window.addEventListener("storage", onChange);
    window.addEventListener("focus", onChange);
    return () => {
        window.removeEventListener("storage", onChange);
        window.removeEventListener("focus", onChange);
    };
}

const serverFalse = () => false;

export default function Navbar() {
    const router = useRouter();
    const pathname = usePathname();

    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

    const loggedIn = useSyncExternalStore(subscribe, isLoggedIn, serverFalse);
    const admin = useSyncExternalStore(subscribe, isAdmin, serverFalse);

    const handleLogout = useCallback(() => {
        clearToken();
        setMobileMenuOpen(false);
        router.push("/login");
    }, [router]);

    const navLinks = [
        { name: "Trip", href: "/" },
        { name: "Pesanan Saya", href: "/my-bookings" },
        // Tautan admin hanya untuk tampilan; server tetap memverifikasi
        // role pada setiap panggilan API.
        ...(admin ? [{ name: "Admin", href: "/admin" }] : []),
    ];

    return (
        <nav className="sticky top-0 z-50 border-b bg-white">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
                <div className="flex h-16 justify-between">
                    
                    {/* LOGO */}
                    <div className="flex items-center">
                        <Link href="/" className="flex items-center gap-2">
                            <div className="rounded-lg bg-indigo-500 p-2">
                                <PlaneTakeoff className="h-5 w-5 text-white" />
                            </div>
                            <span className="text-xl font-bold">
                                TripNest
                            </span>
                        </Link>

                        {/* MENU */}
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

                    {/* 🔥 AUTH FIX */}
                    <div className="hidden sm:flex sm:items-center">
                        {loggedIn ? (
                            <button
                                onClick={handleLogout}
                                className="ml-4 flex items-center gap-2 rounded-lg border px-4 py-2 text-sm hover:bg-red-50 hover:text-red-600"
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

                    {/* MOBILE */}
                    <div className="flex items-center sm:hidden">
                        <button onClick={() => setMobileMenuOpen((v) => !v)}>
                            {mobileMenuOpen ? <X /> : <Menu />}
                        </button>
                    </div>
                </div>
            </div>

            {/* MOBILE MENU */}
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
                        {loggedIn ? (
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