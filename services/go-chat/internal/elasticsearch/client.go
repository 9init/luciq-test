package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/logging"
	"go-chat/internal/model"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *logging.Logger
}

type SearchResult struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []struct {
			ID     string                 `json:"_id"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func NewClient(cfg *config.Config, logger *logging.Logger) *Client {
	return &Client{
		baseURL: cfg.ElasticsearchURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Search(appToken string, chatNumber int, query string, page int, pageSize int) ([]*model.Message, int, error) {
	routing := fmt.Sprintf("%s:%d", appToken, chatNumber)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	from := (page - 1) * pageSize

	searchBody := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"application_token": appToken,
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"chat_number": chatNumber,
						},
					},
					map[string]interface{}{
						"multi_match": map[string]interface{}{
							"query":     query,
							"fields":    []string{"content.partial^2", "content.fuzzy"},
							"operator":  "and",
							"fuzziness": "AUTO",
						},
					},
				},
			},
		},
		"sort": []interface{}{
			map[string]interface{}{
				"message_number": "asc",
			},
		},
		"from": from,
		"size": pageSize,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal search body: %w", err)
	}

	url := fmt.Sprintf("%s/messages/_search?routing=%s", c.baseURL, routing)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, 0, fmt.Errorf("search failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result SearchResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse search result: %w", err)
	}

	messages := make([]*model.Message, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var msg model.Message
		sourceBytes, _ := json.Marshal(hit.Source)
		if err := json.Unmarshal(sourceBytes, &msg); err != nil {
			c.logger.Error("[ES] Failed to parse message: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, result.Hits.Total.Value, nil
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
