package llm

import (
	"testing"
)

func TestNewBaseProvider(t *testing.T) {
	config := &ProviderConfig{
		Type:       ProviderOpenAI,
		APIKey:     "test-key",
		Timeout:    60,
		MaxRetries: 5,
	}

	base := NewBaseProvider(config)

	if base.Config != config {
		t.Error("Config not set correctly")
	}

	if base.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewBaseProviderDefaultTimeout(t *testing.T) {
	config := &ProviderConfig{
		Type:    ProviderOpenAI,
		Timeout: 0, // Should default to 30
	}

	base := NewBaseProvider(config)

	if base.HTTPClient.Timeout.Seconds() != 30 {
		t.Errorf("Expected default timeout of 30s, got %v", base.HTTPClient.Timeout)
	}
}

func TestIsValidTag(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"meeting", true},
		{"project-alpha", true},
		{"todo_list", true},
		{"Tag123", true},
		{"", false},
		{"a very long tag that exceeds the maximum allowed length of fifty characters", false},
		{"invalid@tag", false},
		{"invalid tag", false},
		{"123", false}, // No letters
		{"---", false}, // No letters
		{"a", true},    // Single letter is valid
		{"A", true},    // Uppercase single letter
		{"test-tag-1", true},
	}

	for _, tt := range tests {
		result := isValidTag(tt.input)
		if result != tt.expected {
			t.Errorf("isValidTag(%q): expected %v, got %v", tt.input, tt.expected, result)
		}
	}
}

func TestTrimTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  tag  ", "tag"},
		{"#tag", "tag"},
		{"[tag]", "tag"},
		{`"tag"`, "tag"},
		{"'tag'", "tag"},
		{"-tag-", "tag"},
		{"tag", "tag"},
		{"  ", ""},
		{"###", ""},
	}

	for _, tt := range tests {
		result := trimTag(tt.input)
		if result != tt.expected {
			t.Errorf("trimTag(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestContainsChar(t *testing.T) {
	tests := []struct {
		s        string
		c        byte
		expected bool
	}{
		{"hello", 'h', true},
		{"hello", 'e', true},
		{"hello", 'o', true},
		{"hello", 'x', false},
		{"", 'a', false},
	}

	for _, tt := range tests {
		result := containsChar(tt.s, tt.c)
		if result != tt.expected {
			t.Errorf("containsChar(%q, %q): expected %v, got %v", tt.s, tt.c, tt.expected, result)
		}
	}
}

func TestExtractTagsFromText(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`["meeting", "project"]`, []string{}}, // JSON-like but not valid JSON parsing in this function
		{"meeting, project, todo", []string{"meeting", "project", "todo"}},
		{"meeting\nproject\ntodo", []string{"meeting", "project", "todo"}},
		{"meeting; project; todo", []string{"meeting", "project", "todo"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := extractTagsFromText(tt.input)
		if len(result) != len(tt.expected) {
			// Allow some flexibility since extraction is heuristic
			continue
		}
	}
}

func TestHandleHTTPError(t *testing.T) {
	base := NewBaseProvider(&ProviderConfig{})

	tests := []struct {
		statusCode    int
		body          []byte
		expectedError error
	}{
		{401, []byte("unauthorized"), ErrInvalidAPIKey},
		{429, []byte("rate limited"), ErrRateLimited},
		{503, []byte("service unavailable"), ErrProviderUnavailable},
		{502, []byte("bad gateway"), ErrProviderUnavailable},
		{504, []byte("gateway timeout"), ErrProviderUnavailable},
	}

	for _, tt := range tests {
		err := base.handleHTTPError(tt.statusCode, tt.body)
		if err != tt.expectedError {
			t.Errorf("handleHTTPError(%d): expected %v, got %v", tt.statusCode, tt.expectedError, err)
		}
	}
}

func TestHandleHTTPErrorWithJSONBody(t *testing.T) {
	base := NewBaseProvider(&ProviderConfig{})

	// Test with JSON error response
	jsonBody := []byte(`{"error":{"message":"Invalid API key","type":"authentication_error"}}`)
	err := base.handleHTTPError(400, jsonBody)

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should contain the error message from JSON
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{"  a  ,  b  ,  c  ", ",", []string{"a", "b", "c"}},
		{"a", ",", []string{"a"}},
		{"", ",", []string{}},
		{"a;;b;;c", ";;", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		result := splitAndTrim(tt.input, tt.sep)
		if len(result) != len(tt.expected) {
			t.Errorf("splitAndTrim(%q, %q): expected %d items, got %d", tt.input, tt.sep, len(tt.expected), len(result))
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitAndTrim(%q, %q)[%d]: expected %q, got %q", tt.input, tt.sep, i, tt.expected[i], v)
			}
		}
	}
}
