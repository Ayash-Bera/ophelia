package alchemyst

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}

func NewClient(baseURL, apiKey string, logger *logrus.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // Increased from 30s to 120s
		},
		logger: logger,
	}
}

func (c *Client) AddContext(req AddContextRequest) error {
	return c.makeRequest("POST", "/add", req, nil)
}

func (c *Client) SearchContext(req SearchRequest) (*SearchResponse, error) {
	var response SearchResponse
	err := c.makeRequest("POST", "/search", req, &response)
	return &response, err
}

func (c *Client) DeleteContext(req DeleteContextRequest) error {
	return c.makeRequest("POST", "/delete", req, nil)
}

func (c *Client) ViewContext() (*ViewContextResponse, error) {
	var response ViewContextResponse
	err := c.makeRequest("GET", "/view", nil, &response)
	return &response, err
}

func (c *Client) makeRequest(method, endpoint string, payload interface{}, result interface{}) error {
	url := c.baseURL + endpoint
	
	var body io.Reader
	var contentLength int
	
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		contentLength = len(jsonData)
		
		// Log payload size for debugging
		c.logger.WithFields(logrus.Fields{
			"method":         method,
			"url":            url,
			"payload_size":   contentLength,
		}).Debug("Request payload info")
		
		// Only log full payload for small requests to avoid spam
		if contentLength < 1000 {
			c.logger.WithFields(logrus.Fields{
				"method":       method,
				"url":          url,
				"payload_json": string(jsonData),
			}).Debug("Request payload")
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"method":   method,
		"url":      url,
		"has_body": payload != nil,
		"size":     contentLength,
	}).Debug("Making Alchemyst API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"status_code":   resp.StatusCode,
		"method":        method,
		"url":           url,
		"response_size": len(responseBody),
	}).Debug("Alchemyst API response received")

	// Only log response body for small responses or errors
	if len(responseBody) < 500 || resp.StatusCode >= 400 {
		c.logger.WithFields(logrus.Fields{
			"status_code":   resp.StatusCode,
			"method":        method,
			"url":           url,
			"response_body": string(responseBody),
		}).Debug("Response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	if result != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}