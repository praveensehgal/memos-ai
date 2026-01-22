package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BaseProvider provides common functionality for all providers.
// Embed this in concrete provider implementations.
type BaseProvider struct {
	// Config holds the provider configuration.
	Config *ProviderConfig

	// HTTPClient is the HTTP client for API requests.
	HTTPClient *http.Client
}

// NewBaseProvider creates a new base provider with the given config.
func NewBaseProvider(config *ProviderConfig) *BaseProvider {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &BaseProvider{
		Config: config,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// DoRequest performs an HTTP request with common handling.
func (b *BaseProvider) DoRequest(ctx context.Context, method, url string, body interface{}, headers map[string]string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request with retries
	var lastErr error
	maxRetries := b.Config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := b.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Handle HTTP errors
		if resp.StatusCode >= 400 {
			lastErr = b.handleHTTPError(resp.StatusCode, respBody)

			// Don't retry on client errors (4xx) except rate limiting
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				return nil, lastErr
			}

			continue
		}

		return respBody, nil
	}

	return nil, lastErr
}

// handleHTTPError converts HTTP errors to appropriate LLM errors.
func (b *BaseProvider) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case 401:
		return ErrInvalidAPIKey
	case 429:
		return ErrRateLimited
	case 503, 502, 504:
		return ErrProviderUnavailable
	default:
		// Try to extract error message from response body
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
			Message string `json:"message"` // Some APIs use this directly
		}

		if err := json.Unmarshal(body, &errResp); err == nil {
			msg := errResp.Error.Message
			if msg == "" {
				msg = errResp.Message
			}
			if msg != "" {
				return fmt.Errorf("API error (status %d): %s", statusCode, msg)
			}
		}

		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
}

// DefaultSuggestTags provides a default implementation using chat completion.
// Providers can override this with native implementations if available.
func (b *BaseProvider) DefaultSuggestTags(ctx context.Context, provider Provider, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	maxTags := req.MaxTags
	if maxTags == 0 {
		maxTags = 5
	}

	systemPrompt := `You are a helpful assistant that suggests relevant tags for notes and memos.
Analyze the content and suggest concise, relevant tags that capture the main topics.
Return ONLY a JSON array of tag strings, nothing else. Example: ["project", "meeting", "todo"]
Tags should be lowercase, single words or hyphenated phrases (e.g., "machine-learning").`

	existingTagsHint := ""
	if len(req.ExistingTags) > 0 {
		existingTagsHint = fmt.Sprintf("\nPrefer using these existing tags when relevant: %v", req.ExistingTags)
	}

	userPrompt := fmt.Sprintf(`Suggest up to %d tags for this content:%s

Content:
%s`, maxTags, existingTagsHint, req.Content)

	completionReq := &CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: systemPrompt},
			{Role: RoleUser, Content: userPrompt},
		},
		Temperature: 0.3, // Lower temperature for more consistent results
		MaxTokens:   100,
	}

	resp, err := provider.Complete(ctx, completionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag suggestions: %w", err)
	}

	// Parse the JSON response
	var tags []string
	if err := json.Unmarshal([]byte(resp.Content), &tags); err != nil {
		// Try to extract tags from non-JSON response
		tags = extractTagsFromText(resp.Content)
	}

	// Limit to maxTags
	if len(tags) > maxTags {
		tags = tags[:maxTags]
	}

	return &SuggestTagsResponse{
		Tags: tags,
	}, nil
}

// DefaultSummarize provides a default implementation using chat completion.
func (b *BaseProvider) DefaultSummarize(ctx context.Context, provider Provider, req *SummarizeRequest) (*SummarizeResponse, error) {
	maxLength := req.MaxLength
	if maxLength == 0 {
		maxLength = 200
	}

	style := req.Style
	if style == "" {
		style = "brief"
	}

	systemPrompt := fmt.Sprintf(`You are a helpful assistant that summarizes content.
Create a %s summary that captures the main points.
Keep the summary under %d characters.
Be concise and informative.`, style, maxLength)

	userPrompt := fmt.Sprintf("Summarize this content:\n\n%s", req.Content)

	completionReq := &CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: systemPrompt},
			{Role: RoleUser, Content: userPrompt},
		},
		Temperature: 0.5,
		MaxTokens:   300,
	}

	resp, err := provider.Complete(ctx, completionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	return &SummarizeResponse{
		Summary: resp.Content,
	}, nil
}

// extractTagsFromText attempts to extract tags from a non-JSON response.
func extractTagsFromText(text string) []string {
	// Simple extraction: split by common delimiters
	var tags []string
	// Remove common prefixes/suffixes
	text = bytes.NewBufferString(text).String()

	// Try to find tags in various formats
	// This is a basic implementation; providers should return proper JSON
	delimiters := []string{",", "\n", ";", " "}
	for _, delim := range delimiters {
		if len(tags) == 0 {
			for _, part := range splitAndTrim(text, delim) {
				if isValidTag(part) {
					tags = append(tags, part)
				}
			}
		}
	}

	return tags
}

// splitAndTrim splits a string and trims each part.
func splitAndTrim(s string, sep string) []string {
	var result []string
	for i, j := 0, 0; i < len(s); i = j + len(sep) {
		j = i
		for j < len(s) {
			found := true
			for k := 0; k < len(sep) && j+k < len(s); k++ {
				if s[j+k] != sep[k] {
					found = false
					break
				}
			}
			if found {
				break
			}
			j++
		}
		if j == len(s) {
			j = len(s)
		}
		part := s[i:j]
		// Trim whitespace and common characters
		trimmed := trimTag(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// trimTag cleans up a potential tag string.
func trimTag(s string) string {
	// Remove leading/trailing whitespace and common characters
	chars := " \t\n\r\"'[]{}#-"
	start := 0
	end := len(s)

	for start < end && containsChar(chars, s[start]) {
		start++
	}
	for end > start && containsChar(chars, s[end-1]) {
		end--
	}

	return s[start:end]
}

// containsChar checks if a string contains a character.
func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// isValidTag checks if a string is a valid tag.
func isValidTag(s string) bool {
	if len(s) == 0 || len(s) > 50 {
		return false
	}

	// Must contain at least one letter
	hasLetter := false
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
		// Allow letters, numbers, hyphens, and underscores
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}

	return hasLetter
}
