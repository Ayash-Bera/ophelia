// frontend/src/lib/api-client.ts
import { SearchRequest, SearchResponse, FeedbackRequest, APIResponse, SearchSuggestion } from './types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class APIClient {
    private baseURL: string;

    constructor(baseURL: string = API_BASE_URL) {
        this.baseURL = baseURL;
    }

    private async request<T>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<T> {
        const url = `${this.baseURL}${endpoint}`;

        const config: RequestInit = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
            ...options,
        };

        try {
            const response = await fetch(url, config);

            if (!response.ok) {
                throw new Error(`API Error: ${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            return data;
        } catch (error) {
            console.error('API Request failed:', error);
            throw error;
        }
    }

    async search(query: string): Promise<SearchResponse> {
        const requestBody: SearchRequest = { query };

        const response = await this.request<APIResponse<{
            results: SearchResult[];
            total: number;
            response_time: number;
        }>>('/api/v1/search', {
            method: 'POST',
            body: JSON.stringify(requestBody),
        });

        // Handle wrapped response format
        return {
            success: response.success,
            message: response.message,
            data: {
                results: response.data?.results || [],
                total: response.data?.total || 0,
                response_time: response.data?.response_time || 0,
            }
        };
    }

    async submitFeedback(feedback: FeedbackRequest): Promise<APIResponse> {
        return this.request<APIResponse>('/api/v1/feedback', {
            method: 'POST',
            body: JSON.stringify(feedback),
        });
    }

    async getSuggestions(query: string, limit: number = 5): Promise<SearchSuggestion[]> {
        const params = new URLSearchParams({
            q: query,
            limit: limit.toString(),
        });

        const response = await this.request<APIResponse<SearchSuggestion[]>>(
            `/api/v1/suggestions?${params}`
        );

        return response.data || [];
    }

    async healthCheck(): Promise<{ status: string }> {
        return this.request<{ status: string }>('/health');
    }
}

export const apiClient = new APIClient();
export default apiClient;