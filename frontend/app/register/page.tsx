'use client';

import { useState } from 'react';
import { userApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import toast from 'react-hot-toast';
import { PlaneTakeoff, Mail, Lock, User, CheckCircle2 } from 'lucide-react';

export default function RegisterPage() {
    const router = useRouter();
    const [formData, setFormData] = useState({
        email: '',
        password: '',
        first_name: '',
        last_name: ''
    });
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);

        const loadingToast = toast.loading('Creating your account...');

        try {
            await userApi.post('/register', formData);
            toast.success('Account created successfully!', { id: loadingToast });
            router.push('/login');
        } catch (err: any) {
            toast.error(err.response?.data?.error || 'Failed to register', { id: loadingToast });
            setLoading(false);
        }
    };

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setFormData({ ...formData, [e.target.name]: e.target.value });
    };

    return (
        <div className="min-h-[80vh] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8 animate-in fade-in slide-in-from-bottom-4 duration-500 ease-out">
            <div className="max-w-md w-full space-y-8 bg-white p-8 sm:p-10 rounded-3xl shadow-xl shadow-indigo-100/50 border border-gray-100">
                <div className="text-center">
                    <div className="mx-auto w-16 h-16 bg-gradient-to-br from-indigo-500 to-indigo-700 rounded-2xl flex items-center justify-center transform -rotate-6 hover:rotate-0 transition-transform shadow-lg shadow-indigo-200">
                        <PlaneTakeoff className="h-8 w-8 text-white" />
                    </div>
                    <h2 className="mt-6 text-3xl font-extrabold text-gray-900 tracking-tight">
                        Create an account
                    </h2>
                    <p className="mt-2 text-sm text-gray-500 font-medium">
                        Join thousands of travelers exploring the world with TripNest.
                    </p>
                </div>

                <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
                    <div className="space-y-4 rounded-md shadow-sm">
                        <div className="grid grid-cols-2 gap-4">
                            <div className="relative group">
                                <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2 ml-1">First Name</label>
                                <div className="relative">
                                    <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                        <User className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                                    </div>
                                    <input
                                        name="first_name"
                                        type="text"
                                        required
                                        className="pl-11 block w-full px-4 py-3 bg-gray-50 border border-transparent rounded-xl text-gray-900 focus:bg-white focus:border-indigo-600 focus:ring-4 focus:ring-indigo-100 transition-all font-medium placeholder-gray-400"
                                        placeholder="John"
                                        value={formData.first_name}
                                        onChange={handleChange}
                                    />
                                </div>
                            </div>

                            <div className="relative group">
                                <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2 ml-1">Last Name</label>
                                <div className="relative">
                                    <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                        <User className="h-4 w-4 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                                    </div>
                                    <input
                                        name="last_name"
                                        type="text"
                                        required
                                        className="pl-11 block w-full px-4 py-3 bg-gray-50 border border-transparent rounded-xl text-gray-900 focus:bg-white focus:border-indigo-600 focus:ring-4 focus:ring-indigo-100 transition-all font-medium placeholder-gray-400"
                                        placeholder="Doe"
                                        value={formData.last_name}
                                        onChange={handleChange}
                                    />
                                </div>
                            </div>
                        </div>

                        <div className="relative group">
                            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2 ml-1">Email address</label>
                            <div className="relative">
                                <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                    <Mail className="h-5 w-5 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                                </div>
                                <input
                                    name="email"
                                    type="email"
                                    autoComplete="email"
                                    required
                                    className="pl-11 block w-full px-4 py-3 bg-gray-50 border border-transparent rounded-xl text-gray-900 focus:bg-white focus:border-indigo-600 focus:ring-4 focus:ring-indigo-100 transition-all font-medium placeholder-gray-400"
                                    placeholder="name@company.com"
                                    value={formData.email}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>

                        <div className="relative group">
                            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2 ml-1">Password</label>
                            <div className="relative">
                                <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                                    <Lock className="h-5 w-5 text-gray-400 group-focus-within:text-indigo-500 transition-colors" />
                                </div>
                                <input
                                    name="password"
                                    type="password"
                                    autoComplete="new-password"
                                    required
                                    className="pl-11 block w-full px-4 py-3 bg-gray-50 border border-transparent rounded-xl text-gray-900 focus:bg-white focus:border-indigo-600 focus:ring-4 focus:ring-indigo-100 transition-all font-medium placeholder-gray-400"
                                    placeholder="••••••••"
                                    value={formData.password}
                                    onChange={handleChange}
                                />
                            </div>
                        </div>
                    </div>

                    <div>
                        <button
                            type="submit"
                            disabled={loading}
                            className={`group relative w-full flex justify-center py-3.5 px-4 border border-transparent text-sm font-bold rounded-xl text-white transition-all active:scale-[0.98] ${loading ? 'bg-indigo-400 cursor-not-allowed' : 'bg-indigo-600 hover:bg-indigo-700 shadow-md shadow-indigo-200 hover:shadow-lg'
                                }`}
                        >
                            <span className="absolute left-0 inset-y-0 flex items-center pl-3">
                                <CheckCircle2 className="h-5 w-5 text-indigo-500 group-hover:text-indigo-400 transition-colors" aria-hidden="true" />
                            </span>
                            {loading ? 'Creating Account...' : 'Sign up'}
                        </button>
                    </div>
                </form>

                <div className="mt-6 text-center text-sm">
                    <span className="text-gray-500">Already have an account? </span>
                    <Link href="/login" className="font-bold text-indigo-600 hover:text-indigo-500 hover:underline transition-all">
                        Sign in
                    </Link>
                </div>
            </div>
        </div>
    );
}
