package alchemyst

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_AddContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/add", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", logrus.New())
	
	req := AddContextRequest{
		Documents: []Document{{
			Content: "Test content",
		}},
		Source:      "test",
		ContextType: "resource",
	}

	err := client.AddContext(req)
	require.NoError(t, err)
}

func TestClient_SearchContext(t *testing.T) {
	expectedResponse := SearchResponse{
		Results: []SearchResult{{
			ContextID:   "ctx-123",
			ContextData: "Test data",
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/search", r.URL.Path)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", logrus.New())
	
	req := SearchRequest{
		Query:                      "test query",
		SimilarityThreshold:        0.8,
		MinimumSimilarityThreshold: 0.5,
	}

	response, err := client.SearchContext(req)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse.Results[0].ContextID, response.Results[0].ContextID)
}

func TestClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", logrus.New())
	
	req := AddContextRequest{}
	err := client.AddContext(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}