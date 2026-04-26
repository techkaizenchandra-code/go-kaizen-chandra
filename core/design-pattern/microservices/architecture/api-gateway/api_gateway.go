// Package api_gateway implements a production-grade API Gateway pattern for microservices architecture.
//
// The API Gateway serves as a single entry point for client applications, providing:
// - Request routing to appropriate backend microservices
// - Authentication and authorization
// - Rate limiting and throttling
// - Circuit breaker pattern for resilience
// - Request/response transformation
// - Load balancing across service instances
// - CORS handling
// - Centralized logging and monitoring
// - Protocol translation (REST to gRPC, etc.)
//
// This implementation follows best practices for production deployments including
// graceful shutdown, configurable timeouts, middleware support, and comprehensive error handling.
package api_gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Route represents a routing configuration for a backend microservice.
type Route struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string
	// Path is the URL path pattern (supports wildcards)
	Path string
	// Target is the backend service URL
	Target string
	// Timeout for requests to this route
	Timeout time.Duration
	// StripPrefix removes the prefix before forwarding to backend
	StripPrefix string
	// RequireAuth indicates if authentication is required
	RequireAuth bool
	// RateLimitConfig for this specific route
	RateLimitConfig *RateLimitConfig
	// CircuitBreakerConfig for this route
	CircuitBreakerConfig *CircuitBreakerConfig
	// RequestTransformer for modifying requests
	RequestTransformer RequestTransformer
	// ResponseTransformer for modifying responses
	ResponseTransformer ResponseTransformer
}

// RateLimitConfig defines rate limiting configuration per route.
type RateLimitConfig struct {
	// RequestsPerSecond allowed for this route
	RequestsPerSecond int
	// BurstSize allows bursts above the rate limit
	BurstSize int
}

// CircuitBreakerConfig defines circuit breaker configuration.
type CircuitBreakerConfig struct {
	// MaxFailures before opening the circuit
	MaxFailures int
	// Timeout before attempting to close the circuit
	Timeout time.Duration
	// ResetTimeout after which failure count is reset
	ResetTimeout time.Duration
}

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState int

const (
	// CircuitClosed allows all requests
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen rejects all requests
	CircuitOpen
	// CircuitHalfOpen allows limited requests to test recovery
	CircuitHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        CircuitBreakerState
	failureCount int
	lastFailTime time.Time
	config       *CircuitBreakerConfig
	successCount int
}

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

// APIGateway is the main gateway struct that handles routing and middleware.
type APIGateway struct {
	mu              sync.RWMutex
	routes          []*Route
	client          *http.Client
	rateLimiters    map[string]*RateLimiter
	circuitBreakers map[string]*CircuitBreaker
	middleware      []Middleware
	authHandler     AuthenticationHandler
	server          *http.Server
	requestMetrics  *RequestMetrics
	shutdownTimeout time.Duration
	serviceRegistry ServiceRegistry
}

// GatewayConfig contains configuration for the API Gateway.
type GatewayConfig struct {
	// ServerAddress to listen on (e.g., ":8080")
	ServerAddress string
	// ReadTimeout for incoming requests
	ReadTimeout time.Duration
	// WriteTimeout for responses
	WriteTimeout time.Duration
	// IdleTimeout for keep-alive connections
	IdleTimeout time.Duration
	// MaxHeaderBytes limits request header size
	MaxHeaderBytes int
	// ShutdownTimeout for graceful shutdown
	ShutdownTimeout time.Duration
	// EnableTLS enables HTTPS
	EnableTLS bool
	// TLSCertFile path to TLS certificate
	TLSCertFile string
	// TLSKeyFile path to TLS private key
	TLSKeyFile string
	// CORSAllowedOrigins for CORS configuration
	CORSAllowedOrigins []string
	// CORSAllowedMethods for CORS configuration
	CORSAllowedMethods []string
	// CORSAllowedHeaders for CORS configuration
	CORSAllowedHeaders []string
}

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// AuthenticationHandler defines the interface for authentication mechanisms.
type AuthenticationHandler interface {
	// Authenticate validates the request and returns user context
	Authenticate(r *http.Request) (context.Context, error)
}

// RequestTransformer modifies requests before forwarding to backend.
type RequestTransformer interface {
	// Transform modifies the request
	Transform(r *http.Request) error
}

// ResponseTransformer modifies responses from backend services.
type ResponseTransformer interface {
	// Transform modifies the response
	Transform(resp *http.Response) error
}

// ServiceRegistry defines interface for service discovery.
type ServiceRegistry interface {
	// GetServiceURL returns the URL for a service instance
	GetServiceURL(serviceName string) (string, error)
	// RegisterService registers a service instance
	RegisterService(serviceName, url string) error
	// DeregisterService removes a service instance
	DeregisterService(serviceName, url string) error
}

// RequestMetrics tracks request statistics.
type RequestMetrics struct {
	mu             sync.RWMutex
	totalRequests  int64
	successfulReqs int64
	failedReqs     int64
	totalLatency   time.Duration
	routeMetrics   map[string]*RouteMetrics
}

// RouteMetrics tracks metrics per route.
type RouteMetrics struct {
	RequestCount   int64
	ErrorCount     int64
	AverageLatency time.Duration
	LastAccessed   time.Time
}

// NewAPIGateway creates a new API Gateway instance.
func NewAPIGateway(config *GatewayConfig) *APIGateway {
	gateway := &APIGateway{
		routes:          make([]*Route, 0),
		client:          &http.Client{Timeout: 30 * time.Second},
		rateLimiters:    make(map[string]*RateLimiter),
		circuitBreakers: make(map[string]*CircuitBreaker),
		middleware:      make([]Middleware, 0),
		shutdownTimeout: config.ShutdownTimeout,
		requestMetrics:  &RequestMetrics{routeMetrics: make(map[string]*RouteMetrics)},
	}

	gateway.server = &http.Server{
		Addr:           config.ServerAddress,
		Handler:        gateway,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	// Register default middleware
	gateway.RegisterMiddleware(loggingMiddleware)
	gateway.RegisterMiddleware(recoveryMiddleware)
	gateway.RegisterMiddleware(metricsMiddleware(gateway.requestMetrics))
	gateway.RegisterMiddleware(corsMiddleware(config.CORSAllowedOrigins, config.CORSAllowedMethods, config.CORSAllowedHeaders))

	return gateway
}

// RegisterRoute adds a new route to the gateway.
func (gw *APIGateway) RegisterRoute(route *Route) {
	gw.mu.Lock()
	defer gw.mu.Unlock()

	gw.routes = append(gw.routes, route)

	// Initialize rate limiter if configured
	if route.RateLimitConfig != nil {
		routeKey := route.Method + ":" + route.Path
		gw.rateLimiters[routeKey] = &RateLimiter{
			tokens:     float64(route.RateLimitConfig.BurstSize),
			maxTokens:  float64(route.RateLimitConfig.BurstSize),
			refillRate: float64(route.RateLimitConfig.RequestsPerSecond),
			lastRefill: time.Now(),
		}
	}

	// Initialize circuit breaker if configured
	if route.CircuitBreakerConfig != nil {
		routeKey := route.Method + ":" + route.Path
		gw.circuitBreakers[routeKey] = &CircuitBreaker{
			state:  CircuitClosed,
			config: route.CircuitBreakerConfig,
		}
	}

	log.Printf("registered route: %s %s -> %s", route.Method, route.Path, route.Target)
}

// RegisterMiddleware adds middleware to the gateway.
func (gw *APIGateway) RegisterMiddleware(mw Middleware) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.middleware = append(gw.middleware, mw)
}

// SetAuthenticationHandler sets the authentication handler.
func (gw *APIGateway) SetAuthenticationHandler(handler AuthenticationHandler) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.authHandler = handler
}

// SetServiceRegistry sets the service discovery registry.
func (gw *APIGateway) SetServiceRegistry(registry ServiceRegistry) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.serviceRegistry = registry
}

// Start begins serving requests.
func (gw *APIGateway) Start() error {
	log.Printf("starting API Gateway on %s", gw.server.Addr)
	return gw.server.ListenAndServe()
}

// StartTLS begins serving requests with TLS.
func (gw *APIGateway) StartTLS(certFile, keyFile string) error {
	log.Printf("starting API Gateway with TLS on %s", gw.server.Addr)
	return gw.server.ListenAndServeTLS(certFile, keyFile)
}

// Stop gracefully shuts down the gateway.
func (gw *APIGateway) Stop(ctx context.Context) error {
	log.Println("shutting down API Gateway...")

	shutdownCtx, cancel := context.WithTimeout(ctx, gw.shutdownTimeout)
	defer cancel()

	if err := gw.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("gateway shutdown failed: %w", err)
	}

	log.Println("API Gateway stopped gracefully")
	return nil
}

// ServeHTTP implements the http.Handler interface.
func (gw *APIGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware chain
	var handler http.Handler = http.HandlerFunc(gw.handleRouting)
	gw.mu.RLock()
	middleware := gw.middleware
	gw.mu.RUnlock()

	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	handler.ServeHTTP(w, r)
}

// handleRouting routes the request to the appropriate backend service.
func (gw *APIGateway) handleRouting(w http.ResponseWriter, r *http.Request) {
	gw.mu.RLock()
	route := gw.findRoute(r.Method, r.URL.Path)
	gw.mu.RUnlock()

	if route == nil {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	// Check authentication if required
	if route.RequireAuth && gw.authHandler != nil {
		ctx, err := gw.authHandler.Authenticate(r)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
		r = r.WithContext(ctx)
	}

	// Apply rate limiting
	routeKey := route.Method + ":" + route.Path
	if err := gw.applyRateLimit(routeKey); err != nil {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Check circuit breaker
	if err := gw.checkCircuitBreaker(routeKey); err != nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Handle the request
	startTime := time.Now()
	err := gw.handleRequest(w, r, route)
	latency := time.Since(startTime)

	// Update circuit breaker state
	if err != nil {
		gw.recordFailure(routeKey)
		log.Printf("request failed: %v (latency: %v)", err, latency)
	} else {
		gw.recordSuccess(routeKey)
	}
}

// findRoute finds a matching route for the given method and path.
func (gw *APIGateway) findRoute(method, path string) *Route {
	for _, route := range gw.routes {
		if route.Method == method && gw.matchPath(route.Path, path) {
			return route
		}
	}
	return nil
}

// matchPath checks if the request path matches the route path pattern.
func (gw *APIGateway) matchPath(pattern, path string) bool {
	// Simple prefix matching (can be extended with regex or path parameters)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

// handleRequest proxies the request to the backend service.
func (gw *APIGateway) handleRequest(w http.ResponseWriter, r *http.Request, route *Route) error {
	targetURL, err := url.Parse(route.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Configure proxy
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Strip prefix if configured
		if route.StripPrefix != "" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, route.StripPrefix)
		}

		// Add tracing headers
		req.Header.Set("X-Request-ID", uuid.New().String())
		req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

		// Apply request transformation
		if route.RequestTransformer != nil {
			if err := route.RequestTransformer.Transform(req); err != nil {
				log.Printf("request transformation failed: %v", err)
			}
		}
	}

	// Configure response modifier
	proxy.ModifyResponse = func(resp *http.Response) error {
		if route.ResponseTransformer != nil {
			return route.ResponseTransformer.Transform(resp)
		}
		return nil
	}

	// Configure error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}

	// Set timeout if configured
	if route.Timeout > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), route.Timeout)
		defer cancel()
		r = r.WithContext(ctx)
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// applyRateLimit checks and applies rate limiting.
func (gw *APIGateway) applyRateLimit(routeKey string) error {
	gw.mu.RLock()
	limiter, exists := gw.rateLimiters[routeKey]
	gw.mu.RUnlock()

	if !exists {
		return nil // No rate limit configured
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill).Seconds()

	// Refill tokens
	limiter.tokens += elapsed * limiter.refillRate
	if limiter.tokens > limiter.maxTokens {
		limiter.tokens = limiter.maxTokens
	}
	limiter.lastRefill = now

	// Check if we have tokens
	if limiter.tokens < 1 {
		return fmt.Errorf("rate limit exceeded")
	}

	limiter.tokens--
	return nil
}

// checkCircuitBreaker checks if the circuit breaker allows the request.
func (gw *APIGateway) checkCircuitBreaker(routeKey string) error {
	gw.mu.RLock()
	cb, exists := gw.circuitBreakers[routeKey]
	gw.mu.RUnlock()

	if !exists {
		return nil // No circuit breaker configured
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case CircuitOpen:
		// Check if timeout has passed
		if now.Sub(cb.lastFailTime) > cb.config.Timeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			log.Printf("circuit breaker %s moved to half-open state", routeKey)
			return nil
		}
		return fmt.Errorf("circuit breaker is open")

	case CircuitHalfOpen:
		// Allow limited requests to test recovery
		if cb.successCount >= 3 {
			cb.state = CircuitClosed
			cb.failureCount = 0
			log.Printf("circuit breaker %s closed", routeKey)
		}
		return nil

	case CircuitClosed:
		// Reset failure count after reset timeout
		if now.Sub(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.failureCount = 0
		}
		return nil
	}

	return nil
}

// recordSuccess records a successful request for circuit breaker.
func (gw *APIGateway) recordSuccess(routeKey string) {
	gw.mu.RLock()
	cb, exists := gw.circuitBreakers[routeKey]
	gw.mu.RUnlock()

	if !exists {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.successCount++
	}
}

// recordFailure records a failed request for circuit breaker.
func (gw *APIGateway) recordFailure(routeKey string) {
	gw.mu.RLock()
	cb, exists := gw.circuitBreakers[routeKey]
	gw.mu.RUnlock()

	if !exists {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.failureCount >= cb.config.MaxFailures {
		cb.state = CircuitOpen
		log.Printf("circuit breaker %s opened after %d failures", routeKey, cb.failureCount)
	}
}

// loggingMiddleware logs incoming requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		log.Printf("completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// recoveryMiddleware recovers from panics.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// metricsMiddleware collects request metrics.
func metricsMiddleware(metrics *RequestMetrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			metrics.mu.Lock()
			metrics.totalRequests++
			metrics.mu.Unlock()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			latency := time.Since(start)

			metrics.mu.Lock()
			metrics.totalLatency += latency
			if wrapped.statusCode >= 400 {
				metrics.failedReqs++
			} else {
				metrics.successfulReqs++
			}
			metrics.mu.Unlock()
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// corsMiddleware adds CORS headers.
func corsMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetMetrics returns current gateway metrics.
func (gw *APIGateway) GetMetrics() map[string]interface{} {
	gw.requestMetrics.mu.RLock()
	defer gw.requestMetrics.mu.RUnlock()

	avgLatency := time.Duration(0)
	if gw.requestMetrics.totalRequests > 0 {
		avgLatency = gw.requestMetrics.totalLatency / time.Duration(gw.requestMetrics.totalRequests)
	}

	return map[string]interface{}{
		"total_requests":      gw.requestMetrics.totalRequests,
		"successful_requests": gw.requestMetrics.successfulReqs,
		"failed_requests":     gw.requestMetrics.failedReqs,
		"average_latency":     avgLatency.String(),
	}
}

// HealthCheck returns the health status of the gateway.
func (gw *APIGateway) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"routes":    len(gw.routes),
	}

	json.NewEncoder(w).Encode(status)
}

// ReadinessCheck checks if the gateway is ready to serve traffic.
func (gw *APIGateway) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := len(gw.routes) > 0

	status := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().UTC(),
	}

	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(status)
}

// SimpleAuthHandler implements basic token-based authentication.
type SimpleAuthHandler struct {
	validTokens map[string]string
}

// NewSimpleAuthHandler creates a new simple authentication handler.
func NewSimpleAuthHandler(tokens map[string]string) *SimpleAuthHandler {
	return &SimpleAuthHandler{
		validTokens: tokens,
	}
}

// Authenticate validates the bearer token.
func (h *SimpleAuthHandler) Authenticate(r *http.Request) (context.Context, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := parts[1]
	userID, valid := h.validTokens[token]
	if !valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Add user context
	ctx := context.WithValue(r.Context(), "user_id", userID)
	return ctx, nil
}

// HeaderRequestTransformer adds/modifies headers in the request.
type HeaderRequestTransformer struct {
	Headers map[string]string
}

// Transform adds configured headers to the request.
func (t *HeaderRequestTransformer) Transform(r *http.Request) error {
	for key, value := range t.Headers {
		r.Header.Set(key, value)
	}
	return nil
}

// HeaderResponseTransformer adds/modifies headers in the response.
type HeaderResponseTransformer struct {
	Headers map[string]string
}

// Transform adds configured headers to the response.
func (t *HeaderResponseTransformer) Transform(resp *http.Response) error {
	for key, value := range t.Headers {
		resp.Header.Set(key, value)
	}
	return nil
}

// BodyRewriteTransformer can modify response body.
type BodyRewriteTransformer struct {
	Replacements map[string]string
}

// Transform rewrites the response body.
func (t *BodyRewriteTransformer) Transform(resp *http.Response) error {
	if len(t.Replacements) == 0 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()

	content := string(body)
	for old, new := range t.Replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	resp.Body = io.NopCloser(strings.NewReader(content))
	resp.ContentLength = int64(len(content))
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(content)))

	return nil
}
