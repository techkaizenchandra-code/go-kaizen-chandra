// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Chain of Responsibility pattern
// for handling HTTP-like requests with authentication, authorization, validation, rate limiting, and logging.
package behavioral

import (
	"fmt"
	"sync"
	"time"
)

// RequestType defines the type of request being processed
type RequestType string

const (
	RequestTypeAPI   RequestType = "API"
	RequestTypeWeb   RequestType = "WEB"
	RequestTypeAdmin RequestType = "ADMIN"
)

// Priority defines the priority level of a request
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Request represents a request that flows through the chain of responsibility.
// It encapsulates all necessary information for processing.
type Request struct {
	ID          string
	Type        RequestType
	Priority    Priority
	UserID      string
	Token       string
	Role        string
	Data        map[string]interface{}
	Timestamp   time.Time
	IPAddress   string
	ErrorMsg    string
	IsProcessed bool
}

// NewRequest creates a new Request with validation.
func NewRequest(id, userID, token string, requestType RequestType, priority Priority) (*Request, error) {
	if id == "" {
		return nil, fmt.Errorf("request ID cannot be empty")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	return &Request{
		ID:        id,
		Type:      requestType,
		Priority:  priority,
		UserID:    userID,
		Token:     token,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}, nil
}

// Handler defines the interface for handlers in the chain of responsibility.
type Handler interface {
	// Handle processes the request or passes it to the next handler
	Handle(request *Request) error
	// SetNext sets the next handler in the chain
	SetNext(handler Handler) Handler
	// GetName returns the name of the handler for logging purposes
	GetName() string
}

// BaseHandler provides common functionality for all handlers in the chain.
// It implements thread-safe next handler management.
type BaseHandler struct {
	name string
	next Handler
	mu   sync.RWMutex
}

// NewBaseHandler creates a new BaseHandler with validation.
func NewBaseHandler(name string) (*BaseHandler, error) {
	if name == "" {
		return nil, fmt.Errorf("handler name cannot be empty")
	}
	return &BaseHandler{
		name: name,
	}, nil
}

// SetNext sets the next handler in the chain with thread safety.
func (h *BaseHandler) SetNext(handler Handler) Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.next = handler
	return handler
}

// GetName returns the name of the handler.
func (h *BaseHandler) GetName() string {
	return h.name
}

// callNext safely calls the next handler if it exists.
func (h *BaseHandler) callNext(request *Request) error {
	h.mu.RLock()
	next := h.next
	h.mu.RUnlock()

	if next != nil {
		return next.Handle(request)
	}
	return nil
}

// AuthenticationHandler validates user authentication tokens.
type AuthenticationHandler struct {
	*BaseHandler
	validTokens map[string]bool
	mu          sync.RWMutex
}

// NewAuthenticationHandler creates a new authentication handler.
func NewAuthenticationHandler() (*AuthenticationHandler, error) {
	base, err := NewBaseHandler("AuthenticationHandler")
	if err != nil {
		return nil, err
	}

	return &AuthenticationHandler{
		BaseHandler: base,
		validTokens: map[string]bool{
			"valid-token-123":   true,
			"valid-token-456":   true,
			"valid-token-admin": true,
		},
	}, nil
}

// Handle validates the authentication token.
func (h *AuthenticationHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] Processing request ID: %s\n", h.GetName(), request.ID)

	if request.Token == "" {
		request.ErrorMsg = "authentication token is missing"
		return fmt.Errorf("authentication failed: %s", request.ErrorMsg)
	}

	h.mu.RLock()
	isValid := h.validTokens[request.Token]
	h.mu.RUnlock()

	if !isValid {
		request.ErrorMsg = "invalid authentication token"
		return fmt.Errorf("authentication failed: %s", request.ErrorMsg)
	}

	fmt.Printf("[%s] Authentication successful for user: %s\n", h.GetName(), request.UserID)
	return h.callNext(request)
}

// AuthorizationHandler validates user permissions and roles.
type AuthorizationHandler struct {
	*BaseHandler
	rolePermissions map[string][]RequestType
	mu              sync.RWMutex
}

// NewAuthorizationHandler creates a new authorization handler.
func NewAuthorizationHandler() (*AuthorizationHandler, error) {
	base, err := NewBaseHandler("AuthorizationHandler")
	if err != nil {
		return nil, err
	}

	return &AuthorizationHandler{
		BaseHandler: base,
		rolePermissions: map[string][]RequestType{
			"admin": {RequestTypeAPI, RequestTypeWeb, RequestTypeAdmin},
			"user":  {RequestTypeAPI, RequestTypeWeb},
			"guest": {RequestTypeWeb},
		},
	}, nil
}

// Handle validates user authorization based on role.
func (h *AuthorizationHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] Processing request ID: %s\n", h.GetName(), request.ID)

	if request.Role == "" {
		request.ErrorMsg = "user role is not set"
		return fmt.Errorf("authorization failed: %s", request.ErrorMsg)
	}

	h.mu.RLock()
	permissions, exists := h.rolePermissions[request.Role]
	h.mu.RUnlock()

	if !exists {
		request.ErrorMsg = fmt.Sprintf("unknown role: %s", request.Role)
		return fmt.Errorf("authorization failed: %s", request.ErrorMsg)
	}

	authorized := false
	for _, allowedType := range permissions {
		if allowedType == request.Type {
			authorized = true
			break
		}
	}

	if !authorized {
		request.ErrorMsg = fmt.Sprintf("role '%s' not authorized for request type '%s'", request.Role, request.Type)
		return fmt.Errorf("authorization failed: %s", request.ErrorMsg)
	}

	fmt.Printf("[%s] Authorization successful for role: %s\n", h.GetName(), request.Role)
	return h.callNext(request)
}

// ValidationHandler validates request data and structure.
type ValidationHandler struct {
	*BaseHandler
	requiredFields map[RequestType][]string
	mu             sync.RWMutex
}

// NewValidationHandler creates a new validation handler.
func NewValidationHandler() (*ValidationHandler, error) {
	base, err := NewBaseHandler("ValidationHandler")
	if err != nil {
		return nil, err
	}

	return &ValidationHandler{
		BaseHandler: base,
		requiredFields: map[RequestType][]string{
			RequestTypeAPI:   {"endpoint", "method"},
			RequestTypeWeb:   {"page"},
			RequestTypeAdmin: {"action", "target"},
		},
	}, nil
}

// Handle validates the request data.
func (h *ValidationHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] Processing request ID: %s\n", h.GetName(), request.ID)

	if request.Data == nil {
		request.ErrorMsg = "request data is nil"
		return fmt.Errorf("validation failed: %s", request.ErrorMsg)
	}

	h.mu.RLock()
	required, exists := h.requiredFields[request.Type]
	h.mu.RUnlock()

	if exists {
		for _, field := range required {
			if _, ok := request.Data[field]; !ok {
				request.ErrorMsg = fmt.Sprintf("required field '%s' is missing", field)
				return fmt.Errorf("validation failed: %s", request.ErrorMsg)
			}
		}
	}

	fmt.Printf("[%s] Validation successful\n", h.GetName())
	return h.callNext(request)
}

// RateLimitHandler implements rate limiting using token bucket algorithm.
type RateLimitHandler struct {
	*BaseHandler
	tokens         map[string]int
	maxTokens      int
	refillRate     int
	lastRefill     time.Time
	mu             sync.Mutex
	refillInterval time.Duration
}

// NewRateLimitHandler creates a new rate limit handler.
func NewRateLimitHandler(maxTokens, refillRate int, refillInterval time.Duration) (*RateLimitHandler, error) {
	base, err := NewBaseHandler("RateLimitHandler")
	if err != nil {
		return nil, err
	}

	if maxTokens <= 0 {
		return nil, fmt.Errorf("maxTokens must be positive")
	}
	if refillRate <= 0 {
		return nil, fmt.Errorf("refillRate must be positive")
	}

	return &RateLimitHandler{
		BaseHandler:    base,
		tokens:         make(map[string]int),
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefill:     time.Now(),
		refillInterval: refillInterval,
	}, nil
}

// Handle checks and enforces rate limits.
func (h *RateLimitHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] Processing request ID: %s\n", h.GetName(), request.ID)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Refill tokens if needed
	now := time.Now()
	if now.Sub(h.lastRefill) >= h.refillInterval {
		for key := range h.tokens {
			h.tokens[key] = min(h.tokens[key]+h.refillRate, h.maxTokens)
		}
		h.lastRefill = now
	}

	// Initialize tokens for new users
	if _, exists := h.tokens[request.UserID]; !exists {
		h.tokens[request.UserID] = h.maxTokens
	}

	// Check if user has tokens available
	if h.tokens[request.UserID] <= 0 {
		request.ErrorMsg = "rate limit exceeded"
		return fmt.Errorf("rate limiting failed: %s", request.ErrorMsg)
	}

	// Consume a token
	h.tokens[request.UserID]--

	fmt.Printf("[%s] Rate limit check passed, remaining tokens: %d\n", h.GetName(), h.tokens[request.UserID])
	return h.callNext(request)
}

// LoggingHandler logs request and response information.
type LoggingHandler struct {
	*BaseHandler
	logLevel string
}

// NewLoggingHandler creates a new logging handler.
func NewLoggingHandler(logLevel string) (*LoggingHandler, error) {
	base, err := NewBaseHandler("LoggingHandler")
	if err != nil {
		return nil, err
	}

	return &LoggingHandler{
		BaseHandler: base,
		logLevel:    logLevel,
	}, nil
}

// Handle logs the request details.
func (h *LoggingHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] ========== Request Log ==========\n", h.GetName())
	fmt.Printf("[%s] Request ID: %s\n", h.GetName(), request.ID)
	fmt.Printf("[%s] User ID: %s\n", h.GetName(), request.UserID)
	fmt.Printf("[%s] Type: %s\n", h.GetName(), request.Type)
	fmt.Printf("[%s] Priority: %d\n", h.GetName(), request.Priority)
	fmt.Printf("[%s] Timestamp: %s\n", h.GetName(), request.Timestamp.Format(time.RFC3339))
	fmt.Printf("[%s] IP Address: %s\n", h.GetName(), request.IPAddress)
	fmt.Printf("[%s] ==================================\n", h.GetName())

	err := h.callNext(request)

	if err != nil {
		fmt.Printf("[%s] Request processing failed: %v\n", h.GetName(), err)
	} else {
		fmt.Printf("[%s] Request processing completed successfully\n", h.GetName())
	}

	return err
}

// ProcessingHandler is the final handler that processes the actual request.
type ProcessingHandler struct {
	*BaseHandler
	processingTime time.Duration
}

// NewProcessingHandler creates a new processing handler.
func NewProcessingHandler(processingTime time.Duration) (*ProcessingHandler, error) {
	base, err := NewBaseHandler("ProcessingHandler")
	if err != nil {
		return nil, err
	}

	return &ProcessingHandler{
		BaseHandler:    base,
		processingTime: processingTime,
	}, nil
}

// Handle processes the actual request logic.
func (h *ProcessingHandler) Handle(request *Request) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	fmt.Printf("[%s] Processing request ID: %s\n", h.GetName(), request.ID)

	// Simulate processing time
	time.Sleep(h.processingTime)

	request.IsProcessed = true

	fmt.Printf("[%s] Request processed successfully\n", h.GetName())
	return nil
}

// Helper function for min calculation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
