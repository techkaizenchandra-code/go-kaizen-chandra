package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	DSN string
}

type SessionFactory interface {
	PingContext(ctx context.Context) error
	Close() error
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type EntityManager struct {
	Conn *sql.DB
}

func (e EntityManager) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if e.Conn == nil {
		return nil
	}
	return e.Conn.QueryRowContext(ctx, query, args...)
}

func (e EntityManager) PingContext(ctx context.Context) error {
	if e.Conn == nil {
		return fmt.Errorf("database connection is nil")
	}
	return e.Conn.PingContext(ctx)
}
func (e EntityManager) Close() error {
	if e.Conn == nil {
		return fmt.Errorf("database connection is nil")
	}
	return e.Conn.Close()
}

var (
	instance *sql.DB
	once     sync.Once
	initErr  error
	mu       sync.RWMutex
)

func GetSessionFactory(ctx context.Context, config Config) (SessionFactory, error) {
	once.Do(func() {
		db, err := sql.Open("postgres", config.DSN)
		if err != nil {
			initErr = err
			return
		}
		pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			initErr = err
			return
		}

		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}
		mu.Lock()
		instance = db
		mu.Unlock()
		log.Println("Database connection established successfully")
	})
	mu.RLock()
	defer mu.RLock()
	if initErr != nil {
		return nil, initErr
	}
	if instance == nil {
		return nil, fmt.Errorf("database instance is nil")
	}
	return &EntityManager{Conn: instance}, nil
}

func TestSingleton() {
	cfg := Config{
		DSN: "postgres://SagittariusA@localhost:5432/SagittariusA?sslmode=disable",
	}
	ctx := context.Background()
	entityManager, err := GetSessionFactory(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = entityManager.PingContext(ctx)
	if err != nil {
		log.Fatal(err)
	}

	queryContext := entityManager.QueryRowContext(ctx, "SELECT 1")
	err = queryContext.Scan(&queryContext)
	if err != nil {
		log.Fatal(err)
	}

}
