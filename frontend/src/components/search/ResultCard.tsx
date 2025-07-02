// frontend/src/components/search/ResultCard.tsx
'use client';

import { useState } from 'react';
import { ExternalLink, ThumbsUp, ThumbsDown, Copy, Check } from 'lucide-react';
import { SearchResult } from '@/lib/types';

interface ResultCardProps {
    result: SearchResult;
    onFeedback?: (resultId: string, type: 'helpful' | 'not_helpful') => void;
}

export default function ResultCard({ result, onFeedback }: ResultCardProps) {
    const [feedbackGiven, setFeedbackGiven] = useState<string | null>(null);
    const [copied, setCopied] = useState(false);

    const handleFeedback = (type: 'helpful' | 'not_helpful') => {
        setFeedbackGiven(type);
        onFeedback?.(result.context_id, type);
    };

    const copyToClipboard = async () => {
        try {
            await navigator.clipboard.writeText(result.content);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    };

    const getRelevanceBadge = (relevance: string, score: number) => {
        const colors = {
            high: 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400',
            medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400',
            low: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400'
        };

        return (
            <div className="flex items-center gap-2">
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${colors[relevance as keyof typeof colors]}`}>
                    {relevance} relevance
                </span>
                <span className="text-xs text-gray-500 font-mono">
                    {Math.round(score * 100)}%
                </span>
            </div>
        );
    };

    return (
        <div className="group relative rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-600 transition-all duration-200 hover:shadow-lg">
            <div className="p-6">
                {/* Header */}
                <div className="flex items-start justify-between mb-4">
                    <div className="flex-1">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                            {result.title.replace(/_/g, ' ')}
                        </h3>
                        {getRelevanceBadge(result.relevance, result.score)}
                    </div>

                    <div className="flex items-center gap-2 ml-4">
                        <button
                            onClick={copyToClipboard}
                            className="p-2 rounded-md bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                            title="Copy content"
                        >
                            {copied ? (
                                <Check className="h-4 w-4 text-green-600" />
                            ) : (
                                <Copy className="h-4 w-4 text-gray-600 dark:text-gray-400" />
                            )}
                        </button>

                        <a
                            href={result.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-2 rounded-md bg-blue-100 dark:bg-blue-900/20 hover:bg-blue-200 dark:hover:bg-blue-900/40 transition-colors"
                            title="Open in Arch Wiki"
                        >
                            <ExternalLink className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        </a>
                    </div>
                </div>

                {/* Content */}
                <div className="mb-4">
                    <div className="text-sm text-gray-700 dark:text-gray-300 line-clamp-4 leading-relaxed">
                        {result.content.length > 300
                            ? `${result.content.substring(0, 300)}...`
                            : result.content
                        }
                    </div>
                </div>

                {/* Command extraction (if content contains commands) */}
                {result.content.match(/`[^`]+`|sudo \w+|\$ \w+/g) && (
                    <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-md">
                        <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
                            Commands found:
                        </h4>
                        <div className="space-y-1">
                            {result.content.match(/`[^`]+`|sudo \w+[^.\n]*|\$ \w+[^.\n]*/g)?.slice(0, 3).map((cmd, idx) => (
                                <code key={idx} className="block text-xs font-mono text-gray-800 dark:text-gray-200 bg-white dark:bg-gray-900 px-2 py-1 rounded">
                                    {cmd.replace(/`/g, '')}
                                </code>
                            ))}
                        </div>
                    </div>
                )}

                {/* Footer */}
                <div className="flex items-center justify-between pt-4 border-t border-gray-100 dark:border-gray-800">
                    <div className="text-xs text-gray-500">
                        Source: <span className="font-medium">Arch Wiki</span>
                    </div>

                    {/* Feedback buttons */}
                    <div className="flex items-center gap-2">
                        <span className="text-xs text-gray-500 mr-2">Helpful?</span>

                        <button
                            onClick={() => handleFeedback('helpful')}
                            disabled={feedbackGiven !== null}
                            className={`p-1.5 rounded transition-colors ${feedbackGiven === 'helpful'
                                    ? 'bg-green-100 text-green-600 dark:bg-green-900/20 dark:text-green-400'
                                    : feedbackGiven === 'not_helpful'
                                        ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                                        : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-green-100 dark:hover:bg-green-900/20'
                                }`}
                        >
                            <ThumbsUp className="h-3.5 w-3.5" />
                        </button>

                        <button
                            onClick={() => handleFeedback('not_helpful')}
                            disabled={feedbackGiven !== null}
                            className={`p-1.5 rounded transition-colors ${feedbackGiven === 'not_helpful'
                                    ? 'bg-red-100 text-red-600 dark:bg-red-900/20 dark:text-red-400'
                                    : feedbackGiven === 'helpful'
                                        ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                                        : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-red-100 dark:hover:bg-red-900/20'
                                }`}
                        >
                            <ThumbsDown className="h-3.5 w-3.5" />
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}