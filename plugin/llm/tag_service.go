package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded indicates the rate limit has been exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded for tag suggestions")

	// ErrTagServiceNotConfigured indicates the tag service is not properly configured.
	ErrTagServiceNotConfigured = errors.New("tag service not configured")
)

// TagServiceConfig holds configuration for the tag service.
type TagServiceConfig struct {
	// MaxTagsPerRequest is the maximum number of tags to return per request.
	MaxTagsPerRequest int

	// CacheTTL is how long to cache tag suggestions.
	CacheTTL time.Duration

	// MaxCacheSize is the maximum number of cached entries.
	MaxCacheSize int

	// RateLimitRequests is the number of requests allowed per window.
	RateLimitRequests int

	// RateLimitWindow is the time window for rate limiting.
	RateLimitWindow time.Duration

	// EnableAsync enables asynchronous tag generation.
	EnableAsync bool

	// AsyncWorkers is the number of async workers.
	AsyncWorkers int

	// AsyncQueueSize is the size of the async job queue.
	AsyncQueueSize int
}

// DefaultTagServiceConfig returns the default configuration.
func DefaultTagServiceConfig() *TagServiceConfig {
	return &TagServiceConfig{
		MaxTagsPerRequest: 5,
		CacheTTL:          15 * time.Minute,
		MaxCacheSize:      1000,
		RateLimitRequests: 60,
		RateLimitWindow:   time.Minute,
		EnableAsync:       true,
		AsyncWorkers:      2,
		AsyncQueueSize:    100,
	}
}

// cachedTags represents a cached tag suggestion result.
type cachedTags struct {
	tags      []string
	createdAt time.Time
}

// rateLimitEntry tracks rate limit state for a user.
type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

// TagJob represents an asynchronous tag generation job.
type TagJob struct {
	ID           string
	MemoID       int32
	Content      string
	ExistingTags []string
	UserID       int32
	Status       TagJobStatus
	Result       *SuggestTagsResponse
	Error        error
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// TagJobStatus represents the status of a tag job.
type TagJobStatus string

const (
	TagJobStatusPending   TagJobStatus = "pending"
	TagJobStatusRunning   TagJobStatus = "running"
	TagJobStatusCompleted TagJobStatus = "completed"
	TagJobStatusFailed    TagJobStatus = "failed"
)

// TagJobCallback is called when an async tag job completes.
type TagJobCallback func(job *TagJob)

// TagService provides tag suggestion functionality with caching and rate limiting.
type TagService struct {
	llmService Service
	config     *TagServiceConfig

	// Cache
	cache   map[string]*cachedTags
	cacheMu sync.RWMutex

	// Rate limiting
	rateLimits   map[int32]*rateLimitEntry
	rateLimitsMu sync.Mutex

	// Async job handling
	jobQueue    chan *TagJob
	jobs        map[string]*TagJob
	jobsMu      sync.RWMutex
	jobCallback TagJobCallback
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewTagService creates a new tag service.
func NewTagService(llmService Service, config *TagServiceConfig) *TagService {
	if config == nil {
		config = DefaultTagServiceConfig()
	}

	ts := &TagService{
		llmService: llmService,
		config:     config,
		cache:      make(map[string]*cachedTags),
		rateLimits: make(map[int32]*rateLimitEntry),
		jobs:       make(map[string]*TagJob),
		stopCh:     make(chan struct{}),
	}

	if config.EnableAsync {
		ts.jobQueue = make(chan *TagJob, config.AsyncQueueSize)
		ts.startWorkers()
	}

	return ts
}

// startWorkers starts the async job workers.
func (ts *TagService) startWorkers() {
	for i := 0; i < ts.config.AsyncWorkers; i++ {
		ts.wg.Add(1)
		go ts.worker(i)
	}
	slog.Info("Tag service async workers started",
		slog.Int("workers", ts.config.AsyncWorkers))
}

// worker processes async tag jobs.
func (ts *TagService) worker(id int) {
	defer ts.wg.Done()

	for {
		select {
		case job := <-ts.jobQueue:
			ts.processJob(job)
		case <-ts.stopCh:
			slog.Info("Tag service worker stopping", slog.Int("worker_id", id))
			return
		}
	}
}

// processJob processes a single tag job.
func (ts *TagService) processJob(job *TagJob) {
	ts.updateJobStatus(job.ID, TagJobStatusRunning)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := ts.llmService.SuggestTags(ctx, &SuggestTagsRequest{
		Content:      job.Content,
		ExistingTags: job.ExistingTags,
		MaxTags:      ts.config.MaxTagsPerRequest,
	})

	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		job.Status = TagJobStatusFailed
		job.Error = err
		slog.Error("Tag job failed",
			slog.String("job_id", job.ID),
			slog.Int("memo_id", int(job.MemoID)),
			slog.String("error", err.Error()))
	} else {
		job.Status = TagJobStatusCompleted
		job.Result = result
		// Cache the result
		ts.cacheResult(job.Content, job.ExistingTags, result.Tags)
		slog.Info("Tag job completed",
			slog.String("job_id", job.ID),
			slog.Int("memo_id", int(job.MemoID)),
			slog.Int("tags_count", len(result.Tags)))
	}

	ts.jobsMu.Lock()
	ts.jobs[job.ID] = job
	ts.jobsMu.Unlock()

	if ts.jobCallback != nil {
		ts.jobCallback(job)
	}
}

// updateJobStatus updates the status of a job.
func (ts *TagService) updateJobStatus(jobID string, status TagJobStatus) {
	ts.jobsMu.Lock()
	defer ts.jobsMu.Unlock()

	if job, exists := ts.jobs[jobID]; exists {
		job.Status = status
	}
}

// Stop gracefully stops the tag service.
func (ts *TagService) Stop() {
	close(ts.stopCh)
	ts.wg.Wait()
	slog.Info("Tag service stopped")
}

// SetJobCallback sets the callback for job completion.
func (ts *TagService) SetJobCallback(cb TagJobCallback) {
	ts.jobCallback = cb
}

// SuggestTags suggests tags for the given content with caching and rate limiting.
func (ts *TagService) SuggestTags(ctx context.Context, userID int32, content string, existingTags []string) (*SuggestTagsResponse, error) {
	// Check rate limit
	if !ts.checkRateLimit(userID) {
		return nil, ErrRateLimitExceeded
	}

	// Check cache
	if cached := ts.getFromCache(content, existingTags); cached != nil {
		slog.Debug("Tag suggestion cache hit",
			slog.Int("user_id", int(userID)),
			slog.Int("tags_count", len(cached)))
		return &SuggestTagsResponse{Tags: cached}, nil
	}

	// Call LLM service
	result, err := ts.llmService.SuggestTags(ctx, &SuggestTagsRequest{
		Content:      content,
		ExistingTags: existingTags,
		MaxTags:      ts.config.MaxTagsPerRequest,
	})
	if err != nil {
		return nil, err
	}

	// Cache the result
	ts.cacheResult(content, existingTags, result.Tags)

	slog.Info("Tag suggestion generated",
		slog.Int("user_id", int(userID)),
		slog.Int("tags_count", len(result.Tags)))

	return result, nil
}

// SuggestTagsAsync queues an async tag suggestion job.
func (ts *TagService) SuggestTagsAsync(userID int32, memoID int32, content string, existingTags []string) (*TagJob, error) {
	if !ts.config.EnableAsync {
		return nil, errors.New("async tag generation is disabled")
	}

	// Check rate limit
	if !ts.checkRateLimit(userID) {
		return nil, ErrRateLimitExceeded
	}

	// Check cache first
	if cached := ts.getFromCache(content, existingTags); cached != nil {
		// Return completed job immediately
		now := time.Now()
		job := &TagJob{
			ID:           generateJobID(memoID, content),
			MemoID:       memoID,
			Content:      content,
			ExistingTags: existingTags,
			UserID:       userID,
			Status:       TagJobStatusCompleted,
			Result:       &SuggestTagsResponse{Tags: cached},
			CreatedAt:    now,
			CompletedAt:  &now,
		}
		return job, nil
	}

	job := &TagJob{
		ID:           generateJobID(memoID, content),
		MemoID:       memoID,
		Content:      content,
		ExistingTags: existingTags,
		UserID:       userID,
		Status:       TagJobStatusPending,
		CreatedAt:    time.Now(),
	}

	ts.jobsMu.Lock()
	ts.jobs[job.ID] = job
	ts.jobsMu.Unlock()

	select {
	case ts.jobQueue <- job:
		slog.Info("Tag job queued",
			slog.String("job_id", job.ID),
			slog.Int("memo_id", int(memoID)))
		return job, nil
	default:
		return nil, errors.New("job queue is full")
	}
}

// GetJob retrieves a job by ID.
func (ts *TagService) GetJob(jobID string) (*TagJob, bool) {
	ts.jobsMu.RLock()
	defer ts.jobsMu.RUnlock()

	job, exists := ts.jobs[jobID]
	return job, exists
}

// generateJobID creates a unique job ID.
func generateJobID(memoID int32, content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	h.Write([]byte{byte(memoID >> 24), byte(memoID >> 16), byte(memoID >> 8), byte(memoID)})
	h.Write([]byte(time.Now().String()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// cacheKey generates a cache key from content and existing tags.
func cacheKey(content string, existingTags []string) string {
	h := sha256.New()
	h.Write([]byte(content))
	for _, tag := range existingTags {
		h.Write([]byte(tag))
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// getFromCache retrieves tags from cache if available and not expired.
func (ts *TagService) getFromCache(content string, existingTags []string) []string {
	key := cacheKey(content, existingTags)

	ts.cacheMu.RLock()
	defer ts.cacheMu.RUnlock()

	cached, exists := ts.cache[key]
	if !exists {
		return nil
	}

	if time.Since(cached.createdAt) > ts.config.CacheTTL {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]string, len(cached.tags))
	copy(result, cached.tags)
	return result
}

// cacheResult stores tags in the cache.
func (ts *TagService) cacheResult(content string, existingTags []string, tags []string) {
	key := cacheKey(content, existingTags)

	ts.cacheMu.Lock()
	defer ts.cacheMu.Unlock()

	// Evict old entries if cache is full
	if len(ts.cache) >= ts.config.MaxCacheSize {
		ts.evictOldestEntries()
	}

	ts.cache[key] = &cachedTags{
		tags:      tags,
		createdAt: time.Now(),
	}
}

// evictOldestEntries removes the oldest cache entries.
func (ts *TagService) evictOldestEntries() {
	// Remove expired entries first
	now := time.Now()
	for key, entry := range ts.cache {
		if now.Sub(entry.createdAt) > ts.config.CacheTTL {
			delete(ts.cache, key)
		}
	}

	// If still over limit, remove oldest entries
	if len(ts.cache) >= ts.config.MaxCacheSize {
		// Find and remove the 10% oldest entries
		toRemove := ts.config.MaxCacheSize / 10
		if toRemove < 1 {
			toRemove = 1
		}

		type keyTime struct {
			key       string
			createdAt time.Time
		}
		entries := make([]keyTime, 0, len(ts.cache))
		for key, entry := range ts.cache {
			entries = append(entries, keyTime{key, entry.createdAt})
		}

		// Sort by creation time and remove oldest
		for i := 0; i < toRemove && i < len(entries); i++ {
			oldest := i
			for j := i + 1; j < len(entries); j++ {
				if entries[j].createdAt.Before(entries[oldest].createdAt) {
					oldest = j
				}
			}
			if oldest != i {
				entries[i], entries[oldest] = entries[oldest], entries[i]
			}
			delete(ts.cache, entries[i].key)
		}
	}
}

// checkRateLimit checks if the user has exceeded the rate limit.
func (ts *TagService) checkRateLimit(userID int32) bool {
	ts.rateLimitsMu.Lock()
	defer ts.rateLimitsMu.Unlock()

	now := time.Now()
	entry, exists := ts.rateLimits[userID]

	if !exists || now.After(entry.windowEnd) {
		// Start new window
		ts.rateLimits[userID] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(ts.config.RateLimitWindow),
		}
		return true
	}

	if entry.count >= ts.config.RateLimitRequests {
		return false
	}

	entry.count++
	return true
}

// GetRateLimitStatus returns the current rate limit status for a user.
func (ts *TagService) GetRateLimitStatus(userID int32) (remaining int, resetAt time.Time) {
	ts.rateLimitsMu.Lock()
	defer ts.rateLimitsMu.Unlock()

	now := time.Now()
	entry, exists := ts.rateLimits[userID]

	if !exists || now.After(entry.windowEnd) {
		return ts.config.RateLimitRequests, now.Add(ts.config.RateLimitWindow)
	}

	remaining = ts.config.RateLimitRequests - entry.count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, entry.windowEnd
}

// ClearCache clears the tag suggestion cache.
func (ts *TagService) ClearCache() {
	ts.cacheMu.Lock()
	defer ts.cacheMu.Unlock()

	ts.cache = make(map[string]*cachedTags)
	slog.Info("Tag service cache cleared")
}

// GetCacheStats returns cache statistics.
func (ts *TagService) GetCacheStats() (size int, maxSize int) {
	ts.cacheMu.RLock()
	defer ts.cacheMu.RUnlock()

	return len(ts.cache), ts.config.MaxCacheSize
}

// CleanupExpiredJobs removes old completed/failed jobs.
func (ts *TagService) CleanupExpiredJobs(maxAge time.Duration) int {
	ts.jobsMu.Lock()
	defer ts.jobsMu.Unlock()

	now := time.Now()
	removed := 0

	for id, job := range ts.jobs {
		if job.Status == TagJobStatusCompleted || job.Status == TagJobStatusFailed {
			if job.CompletedAt != nil && now.Sub(*job.CompletedAt) > maxAge {
				delete(ts.jobs, id)
				removed++
			}
		}
	}

	if removed > 0 {
		slog.Info("Cleaned up expired tag jobs", slog.Int("removed", removed))
	}

	return removed
}
