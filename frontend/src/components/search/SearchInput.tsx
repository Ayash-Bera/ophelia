// frontend/src/components/search/SearchInput.tsx
'use client';

import { useState, useEffect } from 'react';
import { Search, Loader2, Terminal } from 'lucide-react';

interface SearchInputProps {
    onSearch: (query: string) => void;
    loading: boolean;
    placeholder?: string;
}

export default function SearchInput({ onSearch, loading, placeholder }: SearchInputProps) {
    const [query, setQuery] = useState('');
    const [focused, setFocused] = useState(false);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (query.trim() && !loading) {
            onSearch(query.trim());
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            handleSubmit(e);
        }
    };

    // Global keyboard shortcut
    useEffect(() => {
        const handleGlobalKeyDown = (e: KeyboardEvent) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                document.getElementById('search-input')?.focus();
            }
        };

        document.addEventListener('keydown', handleGlobalKeyDown);
        return () => document.removeEventListener('keydown', handleGlobalKeyDown);
    }, []);

    const exampleQueries = [
        "pacman: error: failed to commit transaction (conflicting files)",
        "systemd service failed to start",
        "NetworkManager: wifi not working",
        "GRUB rescue prompt",
        "kernel panic on boot"
    ];

    return (
        <div className="w-full max-w-4xl mx-auto">
            <form onSubmit={handleSubmit} className="space-y-4">
                <div className="relative">
                    <div className={`
            relative overflow-hidden rounded-lg border-2 transition-all duration-200
            ${focused
                            ? 'border-blue-500 shadow-lg shadow-blue-500/20'
                            : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                        }
            bg-white dark:bg-gray-950
          `}>
                        <div className="flex items-center gap-3 p-4 border-b border-gray-100 dark:border-gray-800">
                            <Terminal className="h-4 w-4 text-gray-500" />
                            <span className="text-sm font-mono text-gray-600 dark:text-gray-400">
                                arch-search: ~/errors $
                            </span>
                        </div>

                        <textarea
                            id="search-input"
                            value={query}
                            onChange={(e) => setQuery(e.target.value)}
                            onFocus={() => setFocused(true)}
                            onBlur={() => setFocused(false)}
                            onKeyDown={handleKeyDown}
                            placeholder={placeholder || "Paste your error message or describe your problem here..."}
                            className="w-full h-32 p-4 resize-none bg-transparent font-mono text-sm border-none outline-none placeholder:text-gray-500 dark:placeholder:text-gray-400"
                            disabled={loading}
                        />

                        <div className="flex items-center justify-between p-4 border-t border-gray-100 dark:border-gray-800 bg-gray-50 dark:bg-gray-900/50">
                            <div className="flex items-center gap-2 text-xs text-gray-500">
                                <kbd className="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-xs font-mono">
                                    Ctrl+K
                                </kbd>
                                <span>to focus</span>
                                <kbd className="px-2 py-1 bg-gray-200 dark:bg-gray-700 rounded text-xs font-mono">
                                    Ctrl+Enter
                                </kbd>
                                <span>to search</span>
                            </div>

                            <button
                                type="submit"
                                disabled={loading || !query.trim()}
                                className={`
                  flex items-center gap-2 px-6 py-2 rounded-md font-medium transition-all
                  ${loading || !query.trim()
                                        ? 'bg-gray-200 dark:bg-gray-700 text-gray-400 cursor-not-allowed'
                                        : 'bg-blue-600 hover:bg-blue-700 text-white shadow-md hover:shadow-lg'
                                    }
                `}
                            >
                                {loading ? (
                                    <>
                                        <Loader2 className="h-4 w-4 animate-spin" />
                                        Searching...
                                    </>
                                ) : (
                                    <>
                                        <Search className="h-4 w-4" />
                                        Find Solution
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </div>

                {/* Example queries */}
                {!query && (
                    <div className="space-y-3">
                        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                            Common error patterns:
                        </h3>
                        <div className="grid gap-2">
                            {exampleQueries.map((example, index) => (
                                <button
                                    key={index}
                                    type="button"
                                    onClick={() => setQuery(example)}
                                    className="text-left p-3 rounded-md bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors border border-gray-200 dark:border-gray-700"
                                >
                                    <code className="text-sm text-gray-800 dark:text-gray-200 font-mono">
                                        {example}
                                    </code>
                                </button>
                            ))}
                        </div>
                    </div>
                )}
            </form>
        </div>
    );
}