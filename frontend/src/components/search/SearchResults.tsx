// frontend/src/components/search/SearchResults.tsx
'use client';

import { Clock, AlertCircle, CheckCircle2 } from 'lucide-react';
import { SearchResult } from '@/lib/types';
import ResultCard from './ResultCard';

interface SearchResultsProps {
    results: SearchResult[];
    total: number;
    responseTime: number;
    query: string;
    onFeedback?: (resultId: string, type: 'helpful' | 'not_helpful') => void;
}

export default function SearchResults({
    results,
    total,
    responseTime,
    query,
    onFeedback
}: SearchResultsProps) {
    if (results.length === 0) {
        return (
            <div className="w-full max-w-4xl mx-auto mt-8">
                <div className="text-center py-12">
                    <AlertCircle className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                        No solutions found
                    </h3>
                    <p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto">
                        We couldn't find any matching solutions for your query. Try rephrasing your error message or searching for specific components.
                    </p>

                    <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                        <h4 className="text-sm font-medium text-blue-900 dark:text-blue-300 mb-2">
                            Search Tips:
                        </h4>
                        <ul className="text-sm text-blue-800 dark:text-blue-400 space-y-1">
                            <li>• Include specific error messages</li>
                            <li>• Mention the component (pacman, systemd, etc.)</li>
                            <li>• Try broader terms if too specific</li>
                        </ul>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="w-full max-w-4xl mx-auto mt-8">
            {/* Results header */}
            <div className="flex items-center justify-between mb-6 pb-4 border-b border-gray-200 dark:border-gray-700">
                <div className="flex items-center gap-4">
                    <CheckCircle2 className="h-5 w-5 text-green-600" />
                    <div>
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                            Found {total} solution{total !== 1 ? 's' : ''}
                        </h2>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                            for <span className="font-mono bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
                                {query.length > 60 ? `${query.substring(0, 60)}...` : query}
                            </span>
                        </p>
                    </div>
                </div>

                <div className="flex items-center gap-2 text-sm text-gray-500">
                    <Clock className="h-4 w-4" />
                    <span>{responseTime}ms</span>
                </div>
            </div>

            {/* Results list */}
            <div className="space-y-6">
                {results.map((result, index) => (
                    <div key={result.context_id} className="relative">
                        {/* Result number */}
                        <div className="absolute -left-8 top-6 w-6 h-6 bg-blue-100 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center text-sm font-medium">
                            {index + 1}
                        </div>

                        <ResultCard
                            result={result}
                            onFeedback={onFeedback}
                        />
                    </div>
                ))}
            </div>

            {/* Results footer */}
            <div className="mt-8 pt-6 border-t border-gray-200 dark:border-gray-700">
                <div className="text-center text-sm text-gray-600 dark:text-gray-400">
                    <p>
                        Results sourced from the{' '}
                        <a
                            href="https://wiki.archlinux.org"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 dark:text-blue-400 hover:underline font-medium"
                        >
                            Arch Linux Wiki
                        </a>
                    </p>
                    <p className="mt-2">
                        Can't find what you're looking for?{' '}
                        <a
                            href="https://bbs.archlinux.org"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 dark:text-blue-400 hover:underline"
                        >
                            Try the Arch forums
                        </a>{' '}
                        or{' '}
                        <a
                            href="https://www.reddit.com/r/archlinux"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 dark:text-blue-400 hover:underline"
                        >
                            r/archlinux
                        </a>
                    </p>
                </div>
            </div>
        </div>
    );
}