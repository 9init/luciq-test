package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-worker/internal/config"
	"go-worker/internal/logging"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *logging.Logger
}

func NewClient(cfg *config.Config, logger *logging.Logger) *Client {
	c := &Client{
		baseURL: cfg.ElasticsearchURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithPrefix("Elasticsearch"),
	}

	c.waitForConnection()

	if err := c.EnsureIndex(); err != nil {
		c.logger.Error("Failed to create index (will retry on first index operation): %v", err)
	}

	return c
}

func (c *Client) waitForConnection() {
	maxRetries := 30
	retryDelay := 2 * time.Second

	c.logger.Info("Waiting for Elasticsearch to be available...")

	for i := 0; i < maxRetries; i++ {
		if err := c.HealthCheck(); err == nil {
			c.logger.Info("Successfully connected to Elasticsearch")
			return
		}

		if i < maxRetries-1 {
			c.logger.Info("Elasticsearch not ready, retrying in %v... (attempt %d/%d)", retryDelay, i+1, maxRetries)
			time.Sleep(retryDelay)
		}
	}

	c.logger.Error("Failed to connect to Elasticsearch after %d attempts. Indexing worker will continue but indexing will fail until ES is available", maxRetries)
}

func (c *Client) BulkIndex(indexName, bulkBody string) error {
	url := fmt.Sprintf("%s/%s/_bulk", c.baseURL, indexName)
	req, err := http.NewRequest("POST", url, strings.NewReader(bulkBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req.Header.Set("Content-Type", "application/x-ndjson")
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("bulk index failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		c.logger.Error("[ES] Failed to parse response: %v", err)
		return nil
	}

	if errors, ok := result["errors"].(bool); ok && errors {
		c.logger.Error("[ES] Bulk index had some errors: %s", string(bodyBytes))
	}

	return nil
}

func (c *Client) EnsureIndex() error {
	indexName := "messages"

	url := fmt.Sprintf("%s/%s", c.baseURL, indexName)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check index: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		c.logger.Info("[ES] Index '%s' already exists", indexName)
		return nil
	}

	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   3,
			"number_of_replicas": 1,
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"partial_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
							"edge_ngram_filter",
						},
					},
					"standard_lowercase": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
						},
					},
				},
				"filter": map[string]interface{}{
					"edge_ngram_filter": map[string]interface{}{
						"type":     "edge_ngram",
						"min_gram": 2,
						"max_gram": 20,
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"application_token": map[string]string{"type": "keyword"},
				"application_name":  map[string]string{"type": "text"},
				"chat_number":       map[string]string{"type": "integer"},
				"message_number":    map[string]string{"type": "integer"},
				"content": map[string]interface{}{
					"type": "text",
					"fields": map[string]interface{}{
						"partial": map[string]interface{}{
							"type":            "text",
							"analyzer":        "partial_analyzer",
							"search_analyzer": "standard",
						},
						"fuzzy": map[string]interface{}{
							"type":     "text",
							"analyzer": "standard_lowercase",
						},
					},
				},
				"created_at": map[string]string{"type": "date"},
			},
		},
	}

	body, err := json.Marshal(mapping)
	if err != nil {
		return err
	}

	req, err = http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to create index (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	c.logger.Info("[ES] Created index '%s' successfully", indexName)

	return nil
}

func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/_cluster/health", c.baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("elasticsearch unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("elasticsearch unhealthy (status %d)", resp.StatusCode)
	}

	return nil
}
