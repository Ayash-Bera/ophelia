package alchemyst

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
	}
}

func (c *Client) AddContextWithRetry(ctx context.Context, req AddContextRequest) error {
	return c.retryOperation(ctx, func() error {
		return c.AddContext(req)
	})
}

func (c *Client) SearchContextWithRetry(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	var result *SearchResponse
	err := c.retryOperation(ctx, func() error {
		var err error
		result, err = c.SearchContext(req)
		return err
	})
	return result, err
}

func (c *Client) retryOperation(ctx context.Context, operation func() error) error {
	config := DefaultRetryConfig()
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil
		}

		if attempt == config.MaxRetries {
			return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, err)
		}

		delay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		c.logger.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"delay":   delay,
			"error":   err.Error(),
		}).Warn("Retrying Alchemyst operation")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil
}