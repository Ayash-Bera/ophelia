// frontend/src/lib/types.ts
export interface SearchRequest {
    query: string;
}

export interface SearchResult {
    context_id: string;
    title: string;
    content: string;
    url: string;
    score: number;
    relevance: 'high' | 'medium' | 'low';
}

export interface SearchResponse {
    success: boolean;
    message?: string;
    data: {
        results: SearchResult[];
        total: number;
        response_time: number;
    };
}

export interface FeedbackRequest {
    query_id: number;
    feedback_type: 'helpful' | 'not_helpful' | 'partially_helpful';
    feedback_text?: string;
}

export interface APIResponse<T = any> {
    success: boolean;
    message?: string;
    data?: T;
    error?: string;
}

export interface SearchSuggestion {
    query_text: string;
    search_count: number;
}