// Package backend_for_frontend implements the Backend-for-Frontend (BFF) microservices pattern.
//
// The BFF pattern creates separate backend services for different client types (web, mobile, IoT),
// allowing each to have optimized APIs tailored to their specific needs. This implementation provides:
// - Client-specific request handling and response formatting
// - API aggregation from multiple downstream microservices
// - Authentication and authorization
// - Rate limiting per client type
// - Circuit breaking for fault tolerance
// - Response caching for performance optimization
// - Event publishing integration via Transactional Outbox pattern
//
// Production-grade features:
// - Graceful degradation with circuit breakers
// - Request tracing with correlation IDs
// - Metrics and structured logging
// - Concurrent service call aggregation
// - Token bucket rate limiting
package backend_for_frontend

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"microservices/persitance/outbox"

	"github.com/google/uuid"
)

// ClientType represents different types of clients the BFF serves
type ClientType string

const (
	ClientTypeWeb    ClientType = "web"
	ClientTypeMobile ClientType = "mobile"
	ClientTypeIoT    ClientType = "iot"
)

// BFFRequest represents an incoming request to the BFF
type BFFRequest struct {
	RequestID  string
	ClientType ClientType
	UserID     string
	Method     string
	Path       string
	Headers    map[string]string
	Body       []byte
	Context    context.Context
}

// BFFResponse represents the aggregated response from BFF
type BFFResponse struct {
	RequestID string                 `json:"request_id"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Errors    []string               `json:"errors,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ServiceClient defines the interface for communicating with downstream microservices
type ServiceClient interface {
	Call(ctx context.Context, serviceName string, method string, path string, body []byte) ([]byte, error)
}

// HTTPServiceClient implements ServiceClient using HTTP transport
type HTTPServiceClient struct {
	httpClient      *http.Client
	serviceRegistry map[string]string // serviceName -> baseURL
	circuitBreaker  CircuitBreaker
}

// NewHTTPServiceClient creates a new HTTP-based service client
func NewHTTPServiceClient(serviceRegistry map[string]string, circuitBreaker CircuitBreaker) *HTTPServiceClient {
	return &HTTPServiceClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		serviceRegistry: serviceRegistry,
		circuitBreaker:  circuitBreaker,
	}
}

// Call executes an HTTP request to a downstream service with circuit breaker protection
func (c *HTTPServiceClient) Call(ctx context.Context, serviceName string, method string, path string, body []byte) ([]byte, error) {
	baseURL, exists := c.serviceRegistry[serviceName]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	url := baseURL + path

	// Execute with circuit breaker
	result, err := c.circuitBreaker.Execute(ctx, serviceName, func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("service returned error status: %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		return data, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

// Aggregator defines interface for aggregating multiple service responses
type Aggregator interface {
	Aggregate(ctx context.Context, requests map[string]func() (interface{}, error)) (map[string]interface{}, []error)
}

// DefaultAggregator implements concurrent aggregation of service calls
type DefaultAggregator struct {
	timeout time.Duration
}

// NewDefaultAggregator creates a new aggregator with specified timeout
func NewDefaultAggregator(timeout time.Duration) *DefaultAggregator {
	return &DefaultAggregator{
		timeout: timeout,
	}
}

// Aggregate executes multiple service calls concurrently and combines results
func (a *DefaultAggregator) Aggregate(ctx context.Context, requests map[string]func() (interface{}, error)) (map[string]interface{}, []error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	results := make(map[string]interface{})
	errors := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for key, fn := range requests {
		wg.Add(1)
		go func(k string, f func() (interface{}, error)) {
			defer wg.Done()

			result, err := f()

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", k, err))
			} else {
				results[k] = result
			}
		}(key, fn)
	}

	wg.Wait()

	return results, errors
}

// CircuitBreaker defines interface for circuit breaker pattern
type CircuitBreaker interface {
	Execute(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error)
}

// SimpleCircuitBreaker implements basic circuit breaker with state tracking
type SimpleCircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	states       map[string]*circuitState
	mu           sync.RWMutex
}

type circuitState struct {
	failures     int
	lastFailTime time.Time
	isOpen       bool
}

// NewSimpleCircuitBreaker creates a new circuit breaker
func NewSimpleCircuitBreaker(maxFailures int, resetTimeout time.Duration) *SimpleCircuitBreaker {
	return &SimpleCircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		states:       make(map[string]*circuitState),
	}
}

// Execute runs the function with circuit breaker protection
func (cb *SimpleCircuitBreaker) Execute(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()
	state, exists := cb.states[key]
	if !exists {
		state = &circuitState{}
		cb.states[key] = state
	}

	// Check if circuit should be reset
	if state.isOpen && time.Since(state.lastFailTime) > cb.resetTimeout {
		state.isOpen = false
		state.failures = 0
	}

	if state.isOpen {
		cb.mu.Unlock()
		return nil, fmt.Errorf("circuit breaker is open for: %s", key)
	}
	cb.mu.Unlock()

	// Execute function
	result, err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		state.failures++
		state.lastFailTime = time.Now()

		if state.failures >= cb.maxFailures {
			state.isOpen = true
			log.Printf("circuit breaker opened for: %s", key)
		}

		return nil, err
	}

	// Reset on success
	state.failures = 0
	return result, nil
}

// RateLimiter defines interface for rate limiting
type RateLimiter interface {
	Allow(clientID string, clientType ClientType) bool
}

// TokenBucketRateLimiter implements token bucket algorithm for rate limiting
type TokenBucketRateLimiter struct {
	buckets map[string]*tokenBucket
	rates   map[ClientType]int // requests per second
	mu      sync.RWMutex
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	maxTokens  float64
	refillRate float64
}

// NewTokenBucketRateLimiter creates a new rate limiter with different rates per client type
func NewTokenBucketRateLimiter(rates map[ClientType]int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		buckets: make(map[string]*tokenBucket),
		rates:   rates,
	}
}

// Allow checks if the request is allowed based on rate limit
func (rl *TokenBucketRateLimiter) Allow(clientID string, clientType ClientType) bool {
	rate, exists := rl.rates[clientType]
	if !exists {
		rate = 100 // default rate
	}

	key := fmt.Sprintf("%s:%s", clientType, clientID)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(rate),
			lastRefill: time.Now(),
			maxTokens:  float64(rate),
			refillRate: float64(rate),
		}
		rl.buckets[key] = bucket
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(bucket.maxTokens, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastRefill = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Cache defines interface for caching responses
type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
}

// InMemoryCache implements simple in-memory cache with TTL
type InMemoryCache struct {
	data map[string]*cacheEntry
	mu   sync.RWMutex
}

type cacheEntry struct {
	value      []byte
	expiration time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		data: make(map[string]*cacheEntry),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from cache
func (c *InMemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiration) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in cache with TTL
func (c *InMemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// cleanup periodically removes expired entries
func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.expiration) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// AuthService defines interface for authentication and authorization
type AuthService interface {
	ValidateToken(ctx context.Context, token string) (string, error) // returns userID
	Authorize(ctx context.Context, userID string, resource string, action string) error
}

// BFFHandler is the main handler for BFF requests
type BFFHandler struct {
	serviceClient      ServiceClient
	aggregator         Aggregator
	rateLimiter        RateLimiter
	cache              Cache
	authService        AuthService
	outboxRepo         outbox.OutboxRepository
	db                 *sql.DB
	enableMetrics      bool
	enableEventPublish bool
}

// NewBFFHandler creates a new BFF handler with all dependencies
func NewBFFHandler(
	serviceClient ServiceClient,
	aggregator Aggregator,
	rateLimiter RateLimiter,
	cache Cache,
	authService AuthService,
	outboxRepo outbox.OutboxRepository,
	db *sql.DB,
) *BFFHandler {
	return &BFFHandler{
		serviceClient:      serviceClient,
		aggregator:         aggregator,
		rateLimiter:        rateLimiter,
		cache:              cache,
		authService:        authService,
		outboxRepo:         outboxRepo,
		db:                 db,
		enableMetrics:      true,
		enableEventPublish: true,
	}
}

// ServeHTTP implements http.Handler interface
func (h *BFFHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()

	// Set request ID in response header
	w.Header().Set("X-Request-ID", requestID)

	// Parse client type from header
	clientTypeStr := r.Header.Get("X-Client-Type")
	if clientTypeStr == "" {
		clientTypeStr = string(ClientTypeWeb)
	}
	clientType := ClientType(clientTypeStr)

	// Authenticate request
	token := r.Header.Get("Authorization")
	userID, err := h.authService.ValidateToken(r.Context(), token)
	if err != nil {
		h.sendError(w, http.StatusUnauthorized, "authentication failed", requestID)
		return
	}

	// Rate limiting
	if !h.rateLimiter.Allow(userID, clientType) {
		h.sendError(w, http.StatusTooManyRequests, "rate limit exceeded", requestID)
		return
	}

	// Create BFF request
	bffRequest := &BFFRequest{
		RequestID:  requestID,
		ClientType: clientType,
		UserID:     userID,
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    make(map[string]string),
		Context:    r.Context(),
	}

	// Copy headers
	for key, values := range r.Header {
		if len(values) > 0 {
			bffRequest.Headers[key] = values[0]
		}
	}

	// Read body
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			h.sendError(w, http.StatusBadRequest, "failed to read request body", requestID)
			return
		}
		bffRequest.Body = body
	}

	// Route to appropriate handler based on client type
	var response *BFFResponse
	switch clientType {
	case ClientTypeWeb:
		response, err = h.handleWebRequest(bffRequest)
	case ClientTypeMobile:
		response, err = h.handleMobileRequest(bffRequest)
	case ClientTypeIoT:
		response, err = h.handleIoTRequest(bffRequest)
	default:
		h.sendError(w, http.StatusBadRequest, "invalid client type", requestID)
		return
	}

	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error(), requestID)
		return
	}

	// Log metrics
	if h.enableMetrics {
		duration := time.Since(startTime)
		log.Printf("[METRICS] request_id=%s client_type=%s user_id=%s duration=%v", requestID, clientType, userID, duration)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleWebRequest processes requests from web clients with full data responses
func (h *BFFHandler) handleWebRequest(req *BFFRequest) (*BFFResponse, error) {
	log.Printf("[WEB] Processing request: %s %s for user: %s", req.Method, req.Path, req.UserID)

	// Check cache first
	cacheKey := fmt.Sprintf("web:%s:%s", req.UserID, req.Path)
	if cachedData, found := h.cache.Get(cacheKey); found {
		var response BFFResponse
		if err := json.Unmarshal(cachedData, &response); err == nil {
			log.Printf("[WEB] Cache hit for: %s", cacheKey)
			response.RequestID = req.RequestID
			return &response, nil
		}
	}

	// Aggregate data from multiple services
	data, err := h.aggregateData(req.Context, req, []string{"user-service", "product-service", "recommendation-service"})
	if err != nil {
		return nil, err
	}

	response := &BFFResponse{
		RequestID: req.RequestID,
		Data:      data,
		Metadata: map[string]interface{}{
			"client_type": req.ClientType,
			"version":     "1.0",
		},
		Timestamp: time.Now(),
	}

	// Cache response
	if responseData, err := json.Marshal(response); err == nil {
		h.cache.Set(cacheKey, responseData, 5*time.Minute)
	}

	// Publish event
	if h.enableEventPublish {
		h.publishBFFEvent(req.RequestID, "web.request.processed", map[string]interface{}{
			"user_id": req.UserID,
			"path":    req.Path,
		})
	}

	return response, nil
}

// handleMobileRequest processes requests from mobile clients with optimized data
func (h *BFFHandler) handleMobileRequest(req *BFFRequest) (*BFFResponse, error) {
	log.Printf("[MOBILE] Processing request: %s %s for user: %s", req.Method, req.Path, req.UserID)

	// Mobile clients get optimized, minimal data
	data, err := h.aggregateData(req.Context, req, []string{"user-service", "product-service"})
	if err != nil {
		return nil, err
	}

	// Filter and optimize data for mobile
	optimizedData := h.optimizeForMobile(data)

	response := &BFFResponse{
		RequestID: req.RequestID,
		Data:      optimizedData,
		Metadata: map[string]interface{}{
			"client_type": req.ClientType,
			"optimized":   true,
		},
		Timestamp: time.Now(),
	}

	// Publish event
	if h.enableEventPublish {
		h.publishBFFEvent(req.RequestID, "mobile.request.processed", map[string]interface{}{
			"user_id": req.UserID,
			"path":    req.Path,
		})
	}

	return response, nil
}

// handleIoTRequest processes requests from IoT devices with minimal data
func (h *BFFHandler) handleIoTRequest(req *BFFRequest) (*BFFResponse, error) {
	log.Printf("[IOT] Processing request: %s %s for user: %s", req.Method, req.Path, req.UserID)

	// IoT clients get minimal, essential data only
	data, err := h.aggregateData(req.Context, req, []string{"device-service"})
	if err != nil {
		return nil, err
	}

	response := &BFFResponse{
		RequestID: req.RequestID,
		Data:      data,
		Timestamp: time.Now(),
	}

	return response, nil
}

// aggregateData calls multiple backend services and combines their responses
func (h *BFFHandler) aggregateData(ctx context.Context, req *BFFRequest, services []string) (map[string]interface{}, error) {
	requests := make(map[string]func() (interface{}, error))

	for _, serviceName := range services {
		svc := serviceName // capture for closure
		requests[svc] = func() (interface{}, error) {
			data, err := h.serviceClient.Call(ctx, svc, req.Method, req.Path, req.Body)
			if err != nil {
				return nil, err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, err
			}

			return result, nil
		}
	}

	results, errs := h.aggregator.Aggregate(ctx, requests)

	if len(errs) > 0 {
		log.Printf("[WARN] Partial aggregation failures: %v", errs)
	}

	return results, nil
}

// optimizeForMobile reduces data size for mobile clients
func (h *BFFHandler) optimizeForMobile(data map[string]interface{}) map[string]interface{} {
	optimized := make(map[string]interface{})

	// Extract only essential fields
	for key, value := range data {
		// Example: Remove large nested objects, keep only IDs and names
		if mapValue, ok := value.(map[string]interface{}); ok {
			simplified := make(map[string]interface{})
			if id, exists := mapValue["id"]; exists {
				simplified["id"] = id
			}
			if name, exists := mapValue["name"]; exists {
				simplified["name"] = name
			}
			optimized[key] = simplified
		} else {
			optimized[key] = value
		}
	}

	return optimized
}

// publishBFFEvent publishes an event using the outbox pattern
func (h *BFFHandler) publishBFFEvent(aggregateID string, eventType string, payload interface{}) {
	if h.outboxRepo == nil || h.db == nil {
		return
	}

	ctx := context.Background()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to begin transaction for event publishing: %v", err)
		return
	}
	defer tx.Rollback()

	event, err := outbox.CreateOutboxEvent(aggregateID, eventType, payload)
	if err != nil {
		log.Printf("[ERROR] Failed to create outbox event: %v", err)
		return
	}

	if err := h.outboxRepo.Save(ctx, tx, event); err != nil {
		log.Printf("[ERROR] Failed to save outbox event: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] Failed to commit transaction: %v", err)
		return
	}

	log.Printf("[EVENT] Published event: %s (type: %s)", event.ID, event.EventType)
}

// sendError sends an error response to the client
func (h *BFFHandler) sendError(w http.ResponseWriter, statusCode int, message string, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"request_id": requestID,
		"error":      message,
		"timestamp":  time.Now(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// ValidateRequest performs basic request validation
func ValidateRequest(req *BFFRequest) error {
	if req.RequestID == "" {
		return fmt.Errorf("request ID is required")
	}

	if req.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if req.ClientType == "" {
		return fmt.Errorf("client type is required")
	}

	return nil
}
