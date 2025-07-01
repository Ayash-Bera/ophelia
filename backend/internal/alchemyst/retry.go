package alchemyst

import (
	"context"
	"fmt"
	"math"
	"strings"
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
		MaxRetries: 4,
		BaseDelay:  2 * time.Second,
		MaxDelay:   15 * time.Second,
	}
}

func (c *Client) AddContextWithRetry(ctx context.Context, req AddContextRequest) error {
	config := DefaultRetryConfig()

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := c.AddContext(req)
		if err == nil {
			return nil
		}

		// Handle filename conflicts
		if strings.Contains(err.Error(), "File name already exists") ||
			strings.Contains(err.Error(), "BAD_REQUEST") {

			c.logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"error":   err.Error(),
			}).Warn("File name conflict, modifying filename")

			// Modify filename for retry
			if len(req.Documents) > 0 {
				originalName := req.Documents[0].FileName
				timestamp := time.Now().Format("150405")
				newName := fmt.Sprintf("%s-retry%d-%s.txt",
					strings.TrimSuffix(originalName, ".txt"),
					attempt+1,
					timestamp)

				req.Documents[0].FileName = newName

				c.logger.WithFields(logrus.Fields{
					"old_name": originalName,
					"new_name": newName,
				}).Debug("Updated filename for retry")
			}
		}

		if attempt == config.MaxRetries {
			return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, err)
		}

		delay := time.Duration(float64(config.BaseDelay) * math.Pow(1.5, float64(attempt)))
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

		delay := time.Duration(float64(config.BaseDelay) * math.Pow(1.5, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		c.logger.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"delay":   delay,
			"error":   err.Error(),
		}).Warn("Retrying operation")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil
}
