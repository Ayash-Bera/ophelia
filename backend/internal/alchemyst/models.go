package alchemyst

import "time"

// Request models
type AddContextRequest struct {
	UserID         string      `json:"user_id,omitempty"`
	OrganizationID string      `json:"organization_id,omitempty"`
	Documents      []Document  `json:"documents,omitempty"`
	Source         string      `json:"source,omitempty"`
	ContextType    string      `json:"context_type,omitempty"`
	Scope          string      `json:"scope,omitempty"`
	Metadata       interface{} `json:"metadata,omitempty"`
	Chained        bool        `json:"chained,omitempty"`
}

type Document struct {
	Content      string `json:"content"`
	FileName     string `json:"fileName,omitempty"`
	FileType     string `json:"fileType,omitempty"`
	FileSize     int64  `json:"fileSize,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
}

type SearchRequest struct {
	UserID                     string      `json:"user_id,omitempty"`
	Query                      string      `json:"query"`
	SimilarityThreshold        float64     `json:"similarity_threshold"`
	MinimumSimilarityThreshold float64     `json:"minimum_similarity_threshold"`
	Scope                      string      `json:"scope,omitempty"`
	Metadata                   interface{} `json:"metadata,omitempty"`
}

type DeleteContextRequest struct {
	Source         string `json:"source,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	ByDoc          bool   `json:"by_doc,omitempty"`
	ByID           bool   `json:"by_id,omitempty"`
}

// Response models
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	ContextID   string `json:"contextId"`
	ContextData string `json:"contextData"`
}

type ViewContextResponse struct {
	Context []ContextItem `json:"context"`
}

type ContextItem struct {
	ID                      string      `json:"_id"`
	UserID                  string      `json:"user_id"`
	OrganizationID          *string     `json:"organization_id"`
	ParentContextNodes      []string    `json:"parent_context_nodes"`
	ChildrenContextNodes    []string    `json:"children_context_nodes"`
	ContextType             string      `json:"context_type"`
	Tags                    []string    `json:"tags"`
	Source                  string      `json:"source"`
	Content                 interface{} `json:"content"`
	Blob                    interface{} `json:"blob"`
	Indexed                 bool        `json:"indexed"`
	Indexable               bool        `json:"indexable"`
	GoverningPolicies       interface{} `json:"governing_policies"`
	OverridePolicy          bool        `json:"override_policy"`
	TelemetryData           interface{} `json:"telemetry_data"`
	CreatedAt               time.Time   `json:"createdAt"`
	UpdatedAt               time.Time   `json:"updatedAt"`
	Scopes                  []string    `json:"scopes"`
	Metadata                Metadata    `json:"metadata"`
	CorrespondingVectorID   string      `json:"corresponding_vector_id"`
	Text                    string      `json:"text"`
	Score                   float64     `json:"score"`
}

type Metadata struct {
	Size       int64    `json:"size"`
	FileName   string   `json:"file_name"`
	DocType    string   `json:"doc_type"`
	Modalities []string `json:"modalities"`
}