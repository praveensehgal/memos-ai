package llm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockLLMService implements Service interface for testing.
type mockLLMService struct {
	suggestTagsFunc func(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error)
	callCount       int32
	mu              sync.Mutex
}

func (m *mockLLMService) RegisterProvider(provider Provider) error {
	return nil
}

func (m *mockLLMService) GetProvider() Provider {
	return nil
}

func (m *mockLLMService) GetProviderByType(providerType ProviderType) (Provider, error) {
	return nil, nil
}

func (m *mockLLMService) SetActiveProvider(providerType ProviderType) error {
	return nil
}

func (m *mockLLMService) ListProviders() []ProviderStatus {
	return nil
}

func (m *mockLLMService) IsConfigured(ctx context.Context) bool {
	return true
}

func (m *mockLLMService) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return nil, nil
}

func (m *mockLLMService) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, nil
}

func (m *mockLLMService) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.suggestTagsFunc != nil {
		return m.suggestTagsFunc(ctx, req)
	}
	return &SuggestTagsResponse{
		Tags: []string{"tag1", "tag2", "tag3"},
	}, nil
}

func (m *mockLLMService) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	return nil, nil
}

func (m *mockLLMService) GetCallCount() int32 {
	return atomic.LoadInt32(&m.callCount)
}

func (m *mockLLMService) ResetCallCount() {
	atomic.StoreInt32(&m.callCount, 0)
}

func TestNewTagService(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, nil)
	defer ts.Stop()

	if ts == nil {
		t.Fatal("NewTagService returned nil")
	}

	if ts.config.MaxTagsPerRequest != 5 {
		t.Errorf("Expected default MaxTagsPerRequest 5, got %d", ts.config.MaxTagsPerRequest)
	}

	if ts.config.RateLimitRequests != 60 {
		t.Errorf("Expected default RateLimitRequests 60, got %d", ts.config.RateLimitRequests)
	}
}

func TestNewTagServiceWithConfig(t *testing.T) {
	mock := &mockLLMService{}
	config := &TagServiceConfig{
		MaxTagsPerRequest: 10,
		CacheTTL:          5 * time.Minute,
		MaxCacheSize:      500,
		RateLimitRequests: 30,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	}
	ts := NewTagService(mock, config)
	defer ts.Stop()

	if ts.config.MaxTagsPerRequest != 10 {
		t.Errorf("Expected MaxTagsPerRequest 10, got %d", ts.config.MaxTagsPerRequest)
	}

	if ts.config.RateLimitRequests != 30 {
		t.Errorf("Expected RateLimitRequests 30, got %d", ts.config.RateLimitRequests)
	}
}

func TestSuggestTags_Basic(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()
	result, err := ts.SuggestTags(ctx, 1, "This is test content about programming", nil)
	if err != nil {
		t.Fatalf("SuggestTags failed: %v", err)
	}

	if len(result.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(result.Tags))
	}

	if mock.GetCallCount() != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mock.GetCallCount())
	}
}

func TestSuggestTags_Caching(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()
	content := "This is test content for caching"

	// First call
	result1, err := ts.SuggestTags(ctx, 1, content, nil)
	if err != nil {
		t.Fatalf("First SuggestTags failed: %v", err)
	}

	// Second call with same content (should be cached)
	result2, err := ts.SuggestTags(ctx, 1, content, nil)
	if err != nil {
		t.Fatalf("Second SuggestTags failed: %v", err)
	}

	// Should only have called LLM once
	if mock.GetCallCount() != 1 {
		t.Errorf("Expected 1 LLM call (cached), got %d", mock.GetCallCount())
	}

	// Results should be the same
	if len(result1.Tags) != len(result2.Tags) {
		t.Errorf("Cached result should match: got %d vs %d tags", len(result1.Tags), len(result2.Tags))
	}
}

func TestSuggestTags_CacheWithDifferentTags(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()
	content := "This is test content"

	// First call with no existing tags
	_, err := ts.SuggestTags(ctx, 1, content, nil)
	if err != nil {
		t.Fatalf("First SuggestTags failed: %v", err)
	}

	// Second call with existing tags (different cache key)
	_, err = ts.SuggestTags(ctx, 1, content, []string{"existing"})
	if err != nil {
		t.Fatalf("Second SuggestTags failed: %v", err)
	}

	// Should have called LLM twice (different cache keys)
	if mock.GetCallCount() != 2 {
		t.Errorf("Expected 2 LLM calls (different cache keys), got %d", mock.GetCallCount())
	}
}

func TestSuggestTags_RateLimiting(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          1 * time.Millisecond, // Very short TTL to avoid caching
		MaxCacheSize:      100,
		RateLimitRequests: 3,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()

	// Make 3 requests (should succeed)
	for i := 0; i < 3; i++ {
		// Use different content each time to avoid cache
		content := "content " + string(rune('a'+i))
		time.Sleep(5 * time.Millisecond) // Wait for cache to expire
		_, err := ts.SuggestTags(ctx, 1, content, nil)
		if err != nil {
			t.Errorf("Request %d should succeed: %v", i+1, err)
		}
	}

	// 4th request should fail due to rate limit
	time.Sleep(5 * time.Millisecond) // Wait for cache to expire
	_, err := ts.SuggestTags(ctx, 1, "content d", nil)
	if err != ErrRateLimitExceeded {
		t.Errorf("Expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestSuggestTags_RateLimitPerUser(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          1 * time.Millisecond,
		MaxCacheSize:      100,
		RateLimitRequests: 2,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()

	// User 1 makes 2 requests
	for i := 0; i < 2; i++ {
		time.Sleep(5 * time.Millisecond)
		_, err := ts.SuggestTags(ctx, 1, "user1 content "+string(rune('a'+i)), nil)
		if err != nil {
			t.Errorf("User 1 request %d should succeed: %v", i+1, err)
		}
	}

	// User 1's 3rd request should fail
	time.Sleep(5 * time.Millisecond)
	_, err := ts.SuggestTags(ctx, 1, "user1 content c", nil)
	if err != ErrRateLimitExceeded {
		t.Errorf("User 1's 3rd request should be rate limited")
	}

	// User 2 should still be able to make requests
	_, err = ts.SuggestTags(ctx, 2, "user2 content", nil)
	if err != nil {
		t.Errorf("User 2 should not be rate limited: %v", err)
	}
}

func TestGetRateLimitStatus(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          1 * time.Millisecond,
		MaxCacheSize:      100,
		RateLimitRequests: 5,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	// Check status before any requests
	remaining, _ := ts.GetRateLimitStatus(1)
	if remaining != 5 {
		t.Errorf("Expected 5 remaining before requests, got %d", remaining)
	}

	// Make a request
	ctx := context.Background()
	_, _ = ts.SuggestTags(ctx, 1, "test content", nil)

	// Check status after request
	remaining, _ = ts.GetRateLimitStatus(1)
	if remaining != 4 {
		t.Errorf("Expected 4 remaining after 1 request, got %d", remaining)
	}
}

func TestSuggestTagsAsync(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       true,
		AsyncWorkers:      1,
		AsyncQueueSize:    10,
	})
	defer ts.Stop()

	job, err := ts.SuggestTagsAsync(1, 100, "Async content test", nil)
	if err != nil {
		t.Fatalf("SuggestTagsAsync failed: %v", err)
	}

	if job == nil {
		t.Fatal("Job should not be nil")
	}

	if job.MemoID != 100 {
		t.Errorf("Expected MemoID 100, got %d", job.MemoID)
	}

	// Wait for job to complete
	time.Sleep(100 * time.Millisecond)

	// Check job status
	completedJob, exists := ts.GetJob(job.ID)
	if !exists {
		t.Fatal("Job should exist")
	}

	if completedJob.Status != TagJobStatusCompleted {
		t.Errorf("Expected status Completed, got %s", completedJob.Status)
	}

	if completedJob.Result == nil {
		t.Error("Result should not be nil")
	}
}

func TestSuggestTagsAsync_Callback(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       true,
		AsyncWorkers:      1,
		AsyncQueueSize:    10,
	})
	defer ts.Stop()

	callbackCalled := make(chan *TagJob, 1)
	ts.SetJobCallback(func(job *TagJob) {
		callbackCalled <- job
	})

	_, err := ts.SuggestTagsAsync(1, 100, "Callback test content", nil)
	if err != nil {
		t.Fatalf("SuggestTagsAsync failed: %v", err)
	}

	select {
	case job := <-callbackCalled:
		if job.Status != TagJobStatusCompleted {
			t.Errorf("Expected completed job in callback, got %s", job.Status)
		}
	case <-time.After(1 * time.Second):
		t.Error("Callback was not called within timeout")
	}
}

func TestSuggestTagsAsync_CacheHit(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       true,
		AsyncWorkers:      1,
		AsyncQueueSize:    10,
	})
	defer ts.Stop()

	content := "Cache hit test content"

	// First call to populate cache
	ctx := context.Background()
	_, err := ts.SuggestTags(ctx, 1, content, nil)
	if err != nil {
		t.Fatalf("SuggestTags failed: %v", err)
	}

	// Second async call should return immediately from cache
	job, err := ts.SuggestTagsAsync(1, 100, content, nil)
	if err != nil {
		t.Fatalf("SuggestTagsAsync failed: %v", err)
	}

	// Job should be completed immediately (cache hit)
	if job.Status != TagJobStatusCompleted {
		t.Errorf("Expected completed status for cache hit, got %s", job.Status)
	}

	// Should only have made 1 LLM call (the sync one)
	if mock.GetCallCount() != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mock.GetCallCount())
	}
}

func TestSuggestTagsAsync_Disabled(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false, // Disabled
	})
	defer ts.Stop()

	_, err := ts.SuggestTagsAsync(1, 100, "Test content", nil)
	if err == nil {
		t.Error("Expected error when async is disabled")
	}
}

func TestCacheEviction(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      5, // Very small cache
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()

	// Fill the cache
	for i := 0; i < 10; i++ {
		content := "content " + string(rune('a'+i))
		_, _ = ts.SuggestTags(ctx, 1, content, nil)
	}

	// Check cache size
	size, maxSize := ts.GetCacheStats()
	if size > maxSize {
		t.Errorf("Cache size %d exceeds max %d", size, maxSize)
	}
}

func TestClearCache(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()

	// Populate cache
	_, _ = ts.SuggestTags(ctx, 1, "test content", nil)

	size, _ := ts.GetCacheStats()
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}

	// Clear cache
	ts.ClearCache()

	size, _ = ts.GetCacheStats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", size)
	}
}

func TestCleanupExpiredJobs(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      100,
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		EnableAsync:       true,
		AsyncWorkers:      1,
		AsyncQueueSize:    10,
	})
	defer ts.Stop()

	// Create a job
	_, err := ts.SuggestTagsAsync(1, 100, "Cleanup test", nil)
	if err != nil {
		t.Fatalf("SuggestTagsAsync failed: %v", err)
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	// Cleanup with 0 duration (should remove completed jobs)
	removed := ts.CleanupExpiredJobs(0)
	if removed != 1 {
		t.Errorf("Expected 1 job removed, got %d", removed)
	}
}

func TestConcurrentAccess(t *testing.T) {
	mock := &mockLLMService{}
	ts := NewTagService(mock, &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      1000,
		RateLimitRequests: 1000,
		RateLimitWindow:   time.Minute,
		EnableAsync:       false,
	})
	defer ts.Stop()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(userID int32) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, err := ts.SuggestTags(ctx, userID, "test content", nil)
				if err != nil && err != ErrRateLimitExceeded {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		}(int32(i % 5))
	}

	wg.Wait()
}

func TestCacheKey(t *testing.T) {
	// Same content and tags should produce same key
	key1 := cacheKey("content", []string{"tag1", "tag2"})
	key2 := cacheKey("content", []string{"tag1", "tag2"})
	if key1 != key2 {
		t.Error("Same inputs should produce same cache key")
	}

	// Different content should produce different key
	key3 := cacheKey("different content", []string{"tag1", "tag2"})
	if key1 == key3 {
		t.Error("Different content should produce different cache key")
	}

	// Different tags should produce different key
	key4 := cacheKey("content", []string{"tag1"})
	if key1 == key4 {
		t.Error("Different tags should produce different cache key")
	}
}

func TestGenerateJobID(t *testing.T) {
	// Job IDs should be unique
	id1 := generateJobID(1, "content")
	id2 := generateJobID(1, "content")

	// Even with same inputs, IDs should differ (includes timestamp)
	if id1 == id2 {
		t.Error("Job IDs should be unique")
	}

	// IDs should be 16 characters
	if len(id1) != 16 {
		t.Errorf("Expected job ID length 16, got %d", len(id1))
	}
}
