// Package structural provides implementations of structural design patterns.
// This file contains a production-grade implementation of the Proxy pattern
// demonstrating virtual proxy (lazy initialization), protection proxy (access control),
// caching proxy (performance optimization), and logging proxy (audit trail).
package structural

import (
	"fmt"
	"sync"
	"time"
)

// Database defines the interface for database operations.
// This is the Subject interface in the Proxy pattern.
type Database interface {
	// Connect establishes a connection to the database
	Connect() error
	// Query executes a query and returns the result
	Query(query string) (string, error)
	// Close closes the database connection
	Close() error
}

// RealDatabase represents the actual database implementation.
// This is the RealSubject in the Proxy pattern.
type RealDatabase struct {
	connectionString string
	connected        bool
	mu               sync.RWMutex
}

// VirtualProxy implements lazy initialization of the database connection.
// The real database is only created when first accessed.
type VirtualProxy struct {
	connectionString string
	realDB           *RealDatabase
	mu               sync.Mutex
}

// User represents a user with role-based access control.
type User struct {
	ID   string
	Name string
	Role string // "admin", "user", "guest"
}

// ProtectionProxy implements access control for database operations.
// It checks user permissions before delegating to the real subject.
type ProtectionProxy struct {
	database Database
	user     *User
}

// CacheEntry represents a cached query result with expiration.
type CacheEntry struct {
	Result    string
	ExpiresAt time.Time
}

// CachingProxy implements caching of query results to improve performance.
type CachingProxy struct {
	database      Database
	cache         map[string]*CacheEntry
	cacheDuration time.Duration
	mu            sync.RWMutex
}

// LoggingProxy implements logging of all database operations for audit trails.
type LoggingProxy struct {
	database Database
	logFile  string
	mu       sync.Mutex
}

// NewRealDatabase creates a new RealDatabase instance with validation.
func NewRealDatabase(connectionString string) (*RealDatabase, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("connection string cannot be empty")
	}
	return &RealDatabase{
		connectionString: connectionString,
		connected:        false,
	}, nil
}

// Connect establishes a connection to the database.
func (db *RealDatabase) Connect() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.connected {
		return fmt.Errorf("already connected")
	}

	// Simulate connection delay
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("[RealDatabase] Connected to: %s\n", db.connectionString)
	db.connected = true
	return nil
}

// Query executes a database query.
func (db *RealDatabase) Query(query string) (string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if !db.connected {
		return "", fmt.Errorf("database not connected")
	}

	if query == "" {
		return "", fmt.Errorf("query cannot be empty")
	}

	// Simulate query execution delay
	time.Sleep(200 * time.Millisecond)

	result := fmt.Sprintf("Result for query: %s", query)
	fmt.Printf("[RealDatabase] Executed query: %s\n", query)
	return result, nil
}

// Close closes the database connection.
func (db *RealDatabase) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if !db.connected {
		return fmt.Errorf("database not connected")
	}

	fmt.Printf("[RealDatabase] Closing connection to: %s\n", db.connectionString)
	db.connected = false
	return nil
}

// NewVirtualProxy creates a new VirtualProxy instance.
// The real database connection is created lazily on first use.
func NewVirtualProxy(connectionString string) (*VirtualProxy, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("connection string cannot be empty")
	}
	return &VirtualProxy{
		connectionString: connectionString,
	}, nil
}

// getRealDatabase returns the real database instance, creating it if necessary.
func (vp *VirtualProxy) getRealDatabase() (*RealDatabase, error) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.realDB == nil {
		fmt.Println("[VirtualProxy] Creating real database connection (lazy initialization)")
		db, err := NewRealDatabase(vp.connectionString)
		if err != nil {
			return nil, err
		}
		vp.realDB = db
	}
	return vp.realDB, nil
}

// Connect establishes a connection through the virtual proxy.
func (vp *VirtualProxy) Connect() error {
	db, err := vp.getRealDatabase()
	if err != nil {
		return err
	}
	return db.Connect()
}

// Query executes a query through the virtual proxy.
func (vp *VirtualProxy) Query(query string) (string, error) {
	db, err := vp.getRealDatabase()
	if err != nil {
		return "", err
	}
	return db.Query(query)
}

// Close closes the database connection through the virtual proxy.
func (vp *VirtualProxy) Close() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.realDB == nil {
		return fmt.Errorf("database not initialized")
	}
	return vp.realDB.Close()
}

// NewProtectionProxy creates a new ProtectionProxy with access control.
func NewProtectionProxy(database Database, user *User) (*ProtectionProxy, error) {
	if database == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}
	return &ProtectionProxy{
		database: database,
		user:     user,
	}, nil
}

// hasPermission checks if the user has permission for the operation.
func (pp *ProtectionProxy) hasPermission(operation string) bool {
	switch operation {
	case "connect", "query":
		// All authenticated users can connect and query
		return pp.user.Role == "admin" || pp.user.Role == "user" || pp.user.Role == "guest"
	case "close":
		// Only admin and regular users can close connections
		return pp.user.Role == "admin" || pp.user.Role == "user"
	default:
		return false
	}
}

// Connect establishes a connection with permission check.
func (pp *ProtectionProxy) Connect() error {
	if !pp.hasPermission("connect") {
		return fmt.Errorf("user '%s' with role '%s' does not have permission to connect", pp.user.Name, pp.user.Role)
	}
	fmt.Printf("[ProtectionProxy] User '%s' (%s) authorized to connect\n", pp.user.Name, pp.user.Role)
	return pp.database.Connect()
}

// Query executes a query with permission check.
func (pp *ProtectionProxy) Query(query string) (string, error) {
	if !pp.hasPermission("query") {
		return "", fmt.Errorf("user '%s' with role '%s' does not have permission to query", pp.user.Name, pp.user.Role)
	}
	fmt.Printf("[ProtectionProxy] User '%s' (%s) authorized to query\n", pp.user.Name, pp.user.Role)
	return pp.database.Query(query)
}

// Close closes the connection with permission check.
func (pp *ProtectionProxy) Close() error {
	if !pp.hasPermission("close") {
		return fmt.Errorf("user '%s' with role '%s' does not have permission to close connection", pp.user.Name, pp.user.Role)
	}
	fmt.Printf("[ProtectionProxy] User '%s' (%s) authorized to close connection\n", pp.user.Name, pp.user.Role)
	return pp.database.Close()
}

// NewCachingProxy creates a new CachingProxy with specified cache duration.
func NewCachingProxy(database Database, cacheDuration time.Duration) (*CachingProxy, error) {
	if database == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	if cacheDuration <= 0 {
		return nil, fmt.Errorf("cache duration must be positive")
	}
	return &CachingProxy{
		database:      database,
		cache:         make(map[string]*CacheEntry),
		cacheDuration: cacheDuration,
	}, nil
}

// Connect establishes a connection through the caching proxy.
func (cp *CachingProxy) Connect() error {
	return cp.database.Connect()
}

// Query executes a query with caching.
func (cp *CachingProxy) Query(query string) (string, error) {
	cp.mu.RLock()
	entry, exists := cp.cache[query]
	cp.mu.RUnlock()

	// Check if cached result exists and is not expired
	if exists && time.Now().Before(entry.ExpiresAt) {
		fmt.Printf("[CachingProxy] Cache hit for query: %s\n", query)
		return entry.Result, nil
	}

	fmt.Printf("[CachingProxy] Cache miss for query: %s\n", query)
	result, err := cp.database.Query(query)
	if err != nil {
		return "", err
	}

	// Store result in cache
	cp.mu.Lock()
	cp.cache[query] = &CacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(cp.cacheDuration),
	}
	cp.mu.Unlock()

	return result, nil
}

// Close closes the connection and clears the cache.
func (cp *CachingProxy) Close() error {
	cp.mu.Lock()
	cp.cache = make(map[string]*CacheEntry)
	cp.mu.Unlock()

	fmt.Println("[CachingProxy] Cache cleared")
	return cp.database.Close()
}

// ClearCache manually clears all cached entries.
func (cp *CachingProxy) ClearCache() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.cache = make(map[string]*CacheEntry)
	fmt.Println("[CachingProxy] Cache manually cleared")
}

// NewLoggingProxy creates a new LoggingProxy that logs all operations.
func NewLoggingProxy(database Database, logFile string) (*LoggingProxy, error) {
	if database == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	if logFile == "" {
		logFile = "database.log"
	}
	return &LoggingProxy{
		database: database,
		logFile:  logFile,
	}, nil
}

// log writes a log entry with timestamp.
func (lp *LoggingProxy) log(operation, message string) {
	lp.mu.Lock()
	defer lp.mu.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[LoggingProxy] %s | %s | %s\n", timestamp, operation, message)
}

// Connect establishes a connection with logging.
func (lp *LoggingProxy) Connect() error {
	lp.log("CONNECT", "Attempting to connect to database")
	err := lp.database.Connect()
	if err != nil {
		lp.log("CONNECT", fmt.Sprintf("Failed: %v", err))
		return err
	}
	lp.log("CONNECT", "Successfully connected")
	return nil
}

// Query executes a query with logging.
func (lp *LoggingProxy) Query(query string) (string, error) {
	lp.log("QUERY", fmt.Sprintf("Executing: %s", query))
	result, err := lp.database.Query(query)
	if err != nil {
		lp.log("QUERY", fmt.Sprintf("Failed: %v", err))
		return "", err
	}
	lp.log("QUERY", "Successfully executed")
	return result, nil
}

// Close closes the connection with logging.
func (lp *LoggingProxy) Close() error {
	lp.log("CLOSE", "Attempting to close database connection")
	err := lp.database.Close()
	if err != nil {
		lp.log("CLOSE", fmt.Sprintf("Failed: %v", err))
		return err
	}
	lp.log("CLOSE", "Successfully closed connection")
	return nil
}
