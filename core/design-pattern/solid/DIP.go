package main

import (
	"fmt"
	"time"
)

// ============================================
// High-level abstractions (interfaces)
// ============================================

// Logger interface - abstraction for logging
type Logger interface {
	Log(level, message string)
	Error(message string)
	Info(message string)
	Debug(message string)
}

// DataRepository interface - abstraction for data storage
type DataRepository interface {
	Save(entity interface{}) error
	FindByID(id string) (interface{}, error)
	Delete(id string) error
}

// NotificationService interface - abstraction for notifications
type NotificationService interface {
	Send(recipient, message string) error
	SendBatch(recipients []string, message string) error
}

// CacheService interface - abstraction for caching
type CacheService interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
}

// ============================================
// Low-level implementations - Logger
// ============================================

// ConsoleLogger - concrete implementation for console logging
type ConsoleLogger struct {
	prefix string
}

func NewConsoleLogger(prefix string) *ConsoleLogger {
	return &ConsoleLogger{prefix: prefix}
}

func (l *ConsoleLogger) Log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] [%s] [%s] %s\n", timestamp, l.prefix, level, message)
}

func (l *ConsoleLogger) Error(message string) {
	l.Log("ERROR", message)
}

func (l *ConsoleLogger) Info(message string) {
	l.Log("INFO", message)
}

func (l *ConsoleLogger) Debug(message string) {
	l.Log("DEBUG", message)
}

// FileLogger - concrete implementation for file logging
type FileLogger struct {
	filename string
}

func NewFileLogger(filename string) *FileLogger {
	return &FileLogger{filename: filename}
}

func (l *FileLogger) Log(level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// In production, this would write to actual file
	fmt.Printf("[FILE: %s] [%s] [%s] %s\n", l.filename, timestamp, level, message)
}

func (l *FileLogger) Error(message string) {
	l.Log("ERROR", message)
}

func (l *FileLogger) Info(message string) {
	l.Log("INFO", message)
}

func (l *FileLogger) Debug(message string) {
	l.Log("DEBUG", message)
}

// ============================================
// Low-level implementations - DataRepository
// ============================================

// MySQLRepository - concrete implementation for MySQL
type MySQLRepository struct {
	connectionString string
	logger           Logger
}

func NewMySQLRepository(connStr string, logger Logger) *MySQLRepository {
	return &MySQLRepository{
		connectionString: connStr,
		logger:           logger,
	}
}

func (r *MySQLRepository) Save(entity interface{}) error {
	r.logger.Info(fmt.Sprintf("Saving entity to MySQL: %v", entity))
	// Actual MySQL save logic would go here
	return nil
}

func (r *MySQLRepository) FindByID(id string) (interface{}, error) {
	r.logger.Info(fmt.Sprintf("Finding entity by ID in MySQL: %s", id))
	// Actual MySQL find logic would go here
	return map[string]string{"id": id, "name": "User from MySQL"}, nil
}

func (r *MySQLRepository) Delete(id string) error {
	r.logger.Info(fmt.Sprintf("Deleting entity from MySQL: %s", id))
	// Actual MySQL delete logic would go here
	return nil
}

// PostgreSQLRepository - concrete implementation for PostgreSQL
type PostgreSQLRepository struct {
	connectionString string
	logger           Logger
}

func NewPostgreSQLRepository(connStr string, logger Logger) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		connectionString: connStr,
		logger:           logger,
	}
}

func (r *PostgreSQLRepository) Save(entity interface{}) error {
	r.logger.Info(fmt.Sprintf("Saving entity to PostgreSQL: %v", entity))
	// Actual PostgreSQL save logic would go here
	return nil
}

func (r *PostgreSQLRepository) FindByID(id string) (interface{}, error) {
	r.logger.Info(fmt.Sprintf("Finding entity by ID in PostgreSQL: %s", id))
	// Actual PostgreSQL find logic would go here
	return map[string]string{"id": id, "name": "User from PostgreSQL"}, nil
}

func (r *PostgreSQLRepository) Delete(id string) error {
	r.logger.Info(fmt.Sprintf("Deleting entity from PostgreSQL: %s", id))
	// Actual PostgreSQL delete logic would go here
	return nil
}

// ============================================
// Low-level implementations - NotificationService
// ============================================

// EmailNotificationService - concrete implementation for email
type EmailNotificationService struct {
	smtpServer string
	logger     Logger
}

func NewEmailNotificationService(smtpServer string, logger Logger) *EmailNotificationService {
	return &EmailNotificationService{
		smtpServer: smtpServer,
		logger:     logger,
	}
}

func (s *EmailNotificationService) Send(recipient, message string) error {
	s.logger.Info(fmt.Sprintf("Sending email to %s: %s", recipient, message))
	// Actual email sending logic would go here
	return nil
}

func (s *EmailNotificationService) SendBatch(recipients []string, message string) error {
	for _, recipient := range recipients {
		if err := s.Send(recipient, message); err != nil {
			return err
		}
	}
	return nil
}

// SMSNotificationService - concrete implementation for SMS
type SMSNotificationService struct {
	apiKey string
	logger Logger
}

func NewSMSNotificationService(apiKey string, logger Logger) *SMSNotificationService {
	return &SMSNotificationService{
		apiKey: apiKey,
		logger: logger,
	}
}

func (s *SMSNotificationService) Send(recipient, message string) error {
	s.logger.Info(fmt.Sprintf("Sending SMS to %s: %s", recipient, message))
	// Actual SMS sending logic would go here
	return nil
}

func (s *SMSNotificationService) SendBatch(recipients []string, message string) error {
	for _, recipient := range recipients {
		if err := s.Send(recipient, message); err != nil {
			return err
		}
	}
	return nil
}

// ============================================
// Low-level implementations - CacheService
// ============================================

// RedisCache - concrete implementation for Redis
type RedisCache struct {
	host   string
	logger Logger
}

func NewRedisCache(host string, logger Logger) *RedisCache {
	return &RedisCache{
		host:   host,
		logger: logger,
	}
}

func (c *RedisCache) Get(key string) (interface{}, bool) {
	c.logger.Debug(fmt.Sprintf("Getting key from Redis: %s", key))
	// Actual Redis get logic would go here
	return nil, false
}

func (c *RedisCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.logger.Debug(fmt.Sprintf("Setting key in Redis: %s with TTL: %v", key, ttl))
	// Actual Redis set logic would go here
	return nil
}

func (c *RedisCache) Delete(key string) error {
	c.logger.Debug(fmt.Sprintf("Deleting key from Redis: %s", key))
	// Actual Redis delete logic would go here
	return nil
}

// ============================================
// High-level module - UserService
// ============================================

// UserService - high-level business logic that depends on abstractions
type UserService struct {
	repository   DataRepository
	notification NotificationService
	cache        CacheService
	logger       Logger
}

// NewUserService - constructor using dependency injection
func NewUserService(
	repo DataRepository,
	notif NotificationService,
	cache CacheService,
	logger Logger,
) *UserService {
	return &UserService{
		repository:   repo,
		notification: notif,
		cache:        cache,
		logger:       logger,
	}
}

func (s *UserService) CreateUser(id, name, contact string) error {
	s.logger.Info(fmt.Sprintf("Creating user: %s", name))

	user := map[string]string{
		"id":      id,
		"name":    name,
		"contact": contact,
	}

	// Save to database
	if err := s.repository.Save(user); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to save user: %v", err))
		return err
	}

	// Cache the user
	if err := s.cache.Set(fmt.Sprintf("user:%s", id), user, 1*time.Hour); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to cache user: %v", err))
	}

	// Send notification
	message := fmt.Sprintf("Welcome %s! Your account has been created.", name)
	if err := s.notification.Send(contact, message); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to send notification: %v", err))
	}

	s.logger.Info(fmt.Sprintf("User created successfully: %s", id))
	return nil
}

func (s *UserService) GetUser(id string) (interface{}, error) {
	s.logger.Info(fmt.Sprintf("Getting user: %s", id))

	// Try cache first
	cacheKey := fmt.Sprintf("user:%s", id)
	if cached, found := s.cache.Get(cacheKey); found {
		s.logger.Debug("User found in cache")
		return cached, nil
	}

	// Fetch from database
	user, err := s.repository.FindByID(id)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to fetch user: %v", err))
		return nil, err
	}

	// Update cache
	s.cache.Set(cacheKey, user, 1*time.Hour)

	return user, nil
}

func (s *UserService) DeleteUser(id string) error {
	s.logger.Info(fmt.Sprintf("Deleting user: %s", id))

	// Delete from database
	if err := s.repository.Delete(id); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to delete user: %v", err))
		return err
	}

	// Remove from cache
	cacheKey := fmt.Sprintf("user:%s", id)
	s.cache.Delete(cacheKey)

	s.logger.Info(fmt.Sprintf("User deleted successfully: %s", id))
	return nil
}

// ============================================
// Dependency Injection Container
// ============================================

// ServiceContainer - manages dependencies
type ServiceContainer struct {
	logger       Logger
	repository   DataRepository
	notification NotificationService
	cache        CacheService
}

func NewServiceContainer(
	logger Logger,
	repo DataRepository,
	notif NotificationService,
	cache CacheService,
) *ServiceContainer {
	return &ServiceContainer{
		logger:       logger,
		repository:   repo,
		notification: notif,
		cache:        cache,
	}
}

func (c *ServiceContainer) GetUserService() *UserService {
	return NewUserService(c.repository, c.notification, c.cache, c.logger)
}

// ============================================
// Test function
// ============================================

// TestDIP demonstrates the Dependency Inversion Principle
func TestDependencyInversion() {
	fmt.Println("=== Dependency Inversion Principle Demo ===\n")

	// Configuration 1: MySQL + Email + Redis + Console Logger
	fmt.Println("--- Configuration 1: MySQL + Email + Redis + Console Logger ---")
	logger1 := NewConsoleLogger("UserService")
	repo1 := NewMySQLRepository("mysql://localhost:3306/mydb", logger1)
	notif1 := NewEmailNotificationService("smtp.example.com:587", logger1)
	cache1 := NewRedisCache("redis://localhost:6379", logger1)

	container1 := NewServiceContainer(logger1, repo1, notif1, cache1)
	userService1 := container1.GetUserService()

	userService1.CreateUser("user-001", "John Doe", "john@example.com")
	userService1.GetUser("user-001")
	fmt.Println()

	// Configuration 2: PostgreSQL + SMS + Redis + File Logger
	fmt.Println("--- Configuration 2: PostgreSQL + SMS + Redis + File Logger ---")
	logger2 := NewFileLogger("app.log")
	repo2 := NewPostgreSQLRepository("postgres://localhost:5432/mydb", logger2)
	notif2 := NewSMSNotificationService("sync-key-12345", logger2)
	cache2 := NewRedisCache("redis://localhost:6379", logger2)

	container2 := NewServiceContainer(logger2, repo2, notif2, cache2)
	userService2 := container2.GetUserService()

	userService2.CreateUser("user-002", "Jane Smith", "+1234567890")
	userService2.GetUser("user-002")
	userService2.DeleteUser("user-002")
	fmt.Println()

	fmt.Println("✓ Dependency Inversion Principle:")
	fmt.Println("  - High-level modules (UserService) don't depend on low-level modules")
	fmt.Println("  - Both depend on abstractions (interfaces)")
	fmt.Println("  - Dependencies are injected, making the system flexible and testable")
	fmt.Println("  - Easy to swap implementations without changing business logic")
}
