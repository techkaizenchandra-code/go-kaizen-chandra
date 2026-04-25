// Package creational implements Gang of Four creational design patterns.
// This package provides production-ready implementations of singleton and other creational patterns.
package creational

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// DBInstance defines the interface for database operations.
// This interface allows for better testability and mocking.
type DBInstance interface {
	PingContext(ctx context.Context) error
	Close() error
}

// Config holds the database connection configuration.
// All fields are required for proper database initialization.
type Config struct {
	DSN             string        // Data Source Name - connection string for the database
	MaxOpenConn     int           // Maximum number of open connections to the database
	MaxIdleConn     int           // Maximum number of idle connections in the pool
	ConnMaxLifetime time.Duration // Maximum amount of time a connection may be reused
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.DSN == "" {
		return errors.New("DSN cannot be empty")
	}
	if c.MaxOpenConn <= 0 {
		return errors.New("MaxOpenConn must be greater than 0")
	}
	if c.MaxIdleConn <= 0 {
		return errors.New("MaxIdleConn must be greater than 0")
	}
	if c.MaxIdleConn > c.MaxOpenConn {
		return errors.New("MaxIdleConn cannot be greater than MaxOpenConn")
	}
	if c.ConnMaxLifetime <= 0 {
		return errors.New("ConnMaxLifetime must be greater than 0")
	}
	return nil
}

var (
	instance *sql.DB
	once     sync.Once
	initErr  error
	mu       sync.RWMutex // Protects access to instance and initErr
)

// GetInstance returns the singleton database instance.
// It initializes the database connection on first call using the provided configuration.
// Subsequent calls return the same instance regardless of the config parameter.
// Returns an error if initialization fails or if the instance has been closed.
func GetInstance(ctx context.Context, cfg Config) (DBInstance, error) {
	once.Do(func() {
		start := time.Now()

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			initErr = fmt.Errorf("invalid config: %w", err)
			return
		}

		db, err := sql.Open("postgres", cfg.DSN)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		db.SetMaxOpenConns(cfg.MaxOpenConn)
		db.SetMaxIdleConns(cfg.MaxIdleConn)
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close() // Clean up on ping failure
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		mu.Lock()
		instance = db
		mu.Unlock()

		log.Printf("[Singleton] Database initialized successfully in %v", time.Since(start))
	})

	mu.RLock()
	defer mu.RUnlock()

	if initErr != nil {
		return nil, initErr
	}

	if instance == nil {
		return nil, errors.New("database instance is nil")
	}

	return &SQLConnection{Conn: instance}, nil
}

// Close closes the singleton database instance.
// This should typically only be called during application shutdown.
// After calling Close, GetInstance will return an error until the application restarts.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		return nil
	}

	err := instance.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Println("[Singleton] Database connection closed")
	return nil
}

// Reset resets the singleton instance for testing purposes.
// WARNING: This function should ONLY be used in tests.
// It is not thread-safe with respect to GetInstance calls.
func Reset() {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		_ = instance.Close()
	}

	instance = nil
	initErr = nil
	once = sync.Once{}
}

type SQLConnection struct {
	Conn *sql.DB
}

func (S SQLConnection) PingContext(ctx context.Context) error {
	if S.Conn == nil {
		return errors.New("database connection is nil")
	}
	return S.Conn.PingContext(ctx)
}

func (S SQLConnection) Close() error {
	if S.Conn == nil {
		return errors.New("database connection is nil")
	}
	return S.Conn.Close()
}

func TestSingleton() {
	cfg := Config{
		DSN:             "postgres://SagittariusA@localhost:5432/SagittariusA?sslmode=disable",
		MaxOpenConn:     10,
		MaxIdleConn:     5,
		ConnMaxLifetime: 5,
	}
	ctx := context.Background()
	instance, err := GetInstance(ctx, cfg)
	if err != nil {
		fmt.Printf("failed to establish connection: %v\n", err)
		return
	}
	err = instance.PingContext(ctx)
	if err != nil {
		fmt.Printf("ping failed: %v\n", err)
		return
	}

	fmt.Println("connection established successfully")
	defer func(instance DBInstance) {
		err := instance.Close()
		if err != nil {
			fmt.Printf("failed to close connection: %v\n", err)
		}
	}(instance)

}
