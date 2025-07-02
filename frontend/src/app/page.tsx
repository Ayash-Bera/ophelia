// frontend/src/app/page.tsx
'use client';

import { useState } from 'react';
import { AlertCircle, Terminal, Zap, BookOpen } from 'lucide-react';
import SearchInput from '@/components/search/SearchInput';
import SearchResults from '@/components/search/SearchResults';
import { SearchResult } from '@/lib/types';
import { apiClient } from '@/lib/api-client';

export default function HomePage() {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchStats, setSearchStats] = useState<{
    total: number;
    responseTime: number;
  } | null>(null);

  const handleSearch = async (query: string) => {
    if (!query.trim()) return;

    setLoading(true);
    setError(null);
    setSearchQuery(query);

    try {
      const response = await apiClient.search(query);

      if (response.success && response.data) {
        setResults(response.data.results || []);
        setSearchStats({
          total: response.data.total || 0,
          responseTime: response.data.response_time || 0
        });
      } else {
        throw new Error(response.error || 'Search failed');
      }
    } catch (err) {
      console.error('Search error:', err);
      setError(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
      setSearchStats(null);
    } finally {
      setLoading(false);
    }
  };

  const handleFeedback = async (resultId: string, type: 'helpful' | 'not_helpful') => {
    try {
      // For feedback, we'd need the query_id from search results
      // This is a simplified implementation
      await apiClient.submitFeedback({
        query_id: 0, // Would need actual query ID from backend
        feedback_type: type,
        feedback_text: `Feedback for result ${resultId}`
      });
    } catch (err) {
      console.error('Failed to submit feedback:', err);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-950 dark:to-gray-900">
      <main className="container mx-auto px-4 py-8">
        {/* Hero section */}
        {!searchStats && (
          <div className="text-center mb-12 pt-8">
            <div className="flex justify-center mb-6">
              <div className="relative">
                <Terminal className="h-16 w-16 text-blue-600" />
                <div className="absolute -top-1 -right-1 h-4 w-4 bg-green-500 rounded-full border-2 border-white dark:border-gray-950" />
              </div>
            </div>

            <h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100 mb-4">
              Arch Linux Error Search
            </h1>
            <p className="text-xl text-gray-600 dark:text-gray-400 mb-8 max-w-2xl mx-auto">
              Paste your error message and find solutions from the Arch Wiki instantly
            </p>

            {/* Feature highlights */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-3xl mx-auto mb-12">
              <div className="flex items-center gap-3 p-4 bg-white dark:bg-gray-800 rounded-lg shadow-sm">
                <Zap className="h-6 w-6 text-yellow-500" />
                <div className="text-left">
                  <h3 className="font-medium text-gray-900 dark:text-gray-100">
                    AI-Powered
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Smart error pattern matching
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-3 p-4 bg-white dark:bg-gray-800 rounded-lg shadow-sm">
                <BookOpen className="h-6 w-6 text-blue-500" />
                <div className="text-left">
                  <h3 className="font-medium text-gray-900 dark:text-gray-100">
                    Wiki-Sourced
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Official Arch documentation
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-3 p-4 bg-white dark:bg-gray-800 rounded-lg shadow-sm">
                <Terminal className="h-6 w-6 text-green-500" />
                <div className="text-left">
                  <h3 className="font-medium text-gray-900 dark:text-gray-100">
                    Developer-First
                  </h3>
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    Built for efficiency
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Search interface */}
        <SearchInput
          onSearch={handleSearch}
          loading={loading}
          placeholder="Paste your error message here... (e.g., 'pacman: error: failed to commit transaction')"
        />

        {/* Error state */}
        {error && (
          <div className="w-full max-w-4xl mx-auto mt-8">
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
              <div className="flex items-start gap-3">
                <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5" />
                <div>
                  <h3 className="font-medium text-red-900 dark:text-red-300 mb-1">
                    Search Failed
                  </h3>
                  <p className="text-red-700 dark:text-red-400 text-sm">
                    {error}
                  </p>
                  <button
                    onClick={() => setError(null)}
                    className="text-red-600 dark:text-red-400 text-sm font-medium hover:underline mt-2"
                  >
                    Try again
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Search results */}
        {searchStats && (
          <SearchResults
            results={results}
            total={searchStats.total}
            responseTime={searchStats.responseTime}
            query={searchQuery}
            onFeedback={handleFeedback}
          />
        )}

        {/* Loading skeleton */}
        {loading && (
          <div className="w-full max-w-4xl mx-auto mt-8">
            <div className="animate-pulse space-y-6">
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3"></div>
              {[...Array(3)].map((_, i) => (
                <div key={i} className="border border-gray-200 dark:border-gray-700 rounded-lg p-6">
                  <div className="space-y-3">
                    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
                    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2"></div>
                    <div className="space-y-2">
                      <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded"></div>
                      <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-5/6"></div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}