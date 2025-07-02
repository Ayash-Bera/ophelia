// frontend/src/components/layout/Header.tsx
'use client';

import { useState, useEffect } from 'react';
import { Moon, Sun, Terminal, Github, ExternalLink } from 'lucide-react';

export default function Header() {
    const [darkMode, setDarkMode] = useState(false);

    useEffect(() => {
        // Check for saved theme preference or default to system preference
        const savedTheme = localStorage.getItem('theme');
        const systemPrefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

        const shouldUseDark = savedTheme === 'dark' || (!savedTheme && systemPrefersDark);
        setDarkMode(shouldUseDark);

        // Apply theme
        if (shouldUseDark) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
    }, []);

    const toggleDarkMode = () => {
        const newDarkMode = !darkMode;
        setDarkMode(newDarkMode);

        if (newDarkMode) {
            document.documentElement.classList.add('dark');
            localStorage.setItem('theme', 'dark');
        } else {
            document.documentElement.classList.remove('dark');
            localStorage.setItem('theme', 'light');
        }
    };

    return (
        <header className="sticky top-0 z-50 w-full border-b border-gray-200 dark:border-gray-800 bg-white/80 dark:bg-gray-950/80 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:supports-[backdrop-filter]:bg-gray-950/60">
            <div className="container mx-auto px-4">
                <div className="flex h-16 items-center justify-between">
                    {/* Logo and title */}
                    <div className="flex items-center gap-3">
                        <div className="flex items-center gap-2">
                            <Terminal className="h-6 w-6 text-blue-600" />
                            <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">
                                arch-search
                            </h1>
                        </div>
                        <div className="hidden sm:block h-6 w-px bg-gray-300 dark:bg-gray-700" />
                        <p className="hidden sm:block text-sm text-gray-600 dark:text-gray-400">
                            Find solutions to Arch Linux errors
                        </p>
                    </div>

                    {/* Navigation */}
                    <nav className="flex items-center gap-6">
                        <a
                            href="/about"
                            className="text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
                        >
                            How it works
                        </a>

                        <a
                            href="https://wiki.archlinux.org"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-1 text-sm font-medium text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
                        >
                            Arch Wiki
                            <ExternalLink className="h-3 w-3" />
                        </a>

                        <a
                            href="https://github.com"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
                            title="View source code"
                        >
                            <Github className="h-5 w-5" />
                        </a>

                        {/* Theme toggle */}
                        <button
                            onClick={toggleDarkMode}
                            className="p-2 rounded-md bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                            title={`Switch to ${darkMode ? 'light' : 'dark'} mode`}
                        >
                            {darkMode ? (
                                <Sun className="h-4 w-4 text-gray-600 dark:text-gray-300" />
                            ) : (
                                <Moon className="h-4 w-4 text-gray-600 dark:text-gray-300" />
                            )}
                        </button>
                    </nav>
                </div>
            </div>
        </header>
    );
}