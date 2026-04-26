// Package persitance provides persistence layer implementation for CQRS microservices
// including the Transactional Outbox pattern for reliable event publishing.
//
// The Outbox pattern ensures atomicity between database writes and event publishing
// by storing events in an outbox table within the same transaction as the business data.
// A background processor polls the outbox table and publishes events to the message broker.
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent represents an event stored in the outbox table
// waiting to be published to the message broker.
type OutboxEvent struct {
	ID          string
	AggregateID string
	EventType   string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt *time.Time
	IsPublished bool
	RetryCount  int
	LastError   *string
}

// OutboxRepository defines operations for managing outbox events.
type OutboxRepository interface {
	// Save stores a new outbox event within a transaction
	Save(ctx context.Context, tx *sql.Tx, event *OutboxEvent) error

	// GetPendingEvents retrieves events that haven't been published yet
	GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error)

	// MarkAsPublished updates the event status after successful publishing
	MarkAsPublished(ctx context.Context, eventID string) error

	// Delete removes published events from the outbox table
	Delete(ctx context.Context, eventID string) error
}

// outboxRepositoryImpl implements OutboxRepository interface.
type outboxRepositoryImpl struct {
	db *sql.DB
}

// NewOutboxRepository creates a new instance of OutboxRepository.
func NewOutboxRepository(db *sql.DB) OutboxRepository {
	return &outboxRepositoryImpl{
		db: db,
	}
}

// Save persists an outbox event to the database within a transaction.
func (r *outboxRepositoryImpl) Save(ctx context.Context, tx *sql.Tx, event *OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, aggregate_id, event_type, payload, created_at, is_published, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := tx.ExecContext(ctx, query,
		event.ID,
		event.AggregateID,
		event.EventType,
		event.Payload,
		event.CreatedAt,
		event.IsPublished,
		event.RetryCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}

	return nil
}

// GetPendingEvents retrieves unpublished events from the outbox table.
func (r *outboxRepositoryImpl) GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	query := `
		SELECT id, aggregate_id, event_type, payload, created_at, retry_count, last_error
		FROM outbox_events
		WHERE is_published = false
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending events: %w", err)
	}
	defer rows.Close()

	var events []*OutboxEvent
	for rows.Next() {
		event := &OutboxEvent{}
		var lastError sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.CreatedAt,
			&event.RetryCount,
			&lastError,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}

		if lastError.Valid {
			event.LastError = &lastError.String
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// MarkAsPublished updates the event status after successful publishing.
func (r *outboxRepositoryImpl) MarkAsPublished(ctx context.Context, eventID string) error {
	query := `
		UPDATE outbox_events
		SET is_published = true, published_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as published: %w", err)
	}

	return nil
}

// Delete removes a published event from the outbox table.
func (r *outboxRepositoryImpl) Delete(ctx context.Context, eventID string) error {
	query := `DELETE FROM outbox_events WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete outbox event: %w", err)
	}

	return nil
}

// EventPublisher defines the interface for publishing events to a message broker.
type EventPublisher interface {
	// Publish sends an event to the message broker
	Publish(ctx context.Context, event *OutboxEvent) error
}

// OutboxProcessor polls the outbox table and publishes events.
type OutboxProcessor struct {
	repository   OutboxRepository
	publisher    EventPublisher
	pollInterval time.Duration
	batchSize    int
	maxRetries   int
	stopChan     chan struct{}
	doneChan     chan struct{}
}

// NewOutboxProcessor creates a new outbox event processor.
func NewOutboxProcessor(
	repository OutboxRepository,
	publisher EventPublisher,
	pollInterval time.Duration,
	batchSize int,
	maxRetries int,
) *OutboxProcessor {
	return &OutboxProcessor{
		repository:   repository,
		publisher:    publisher,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		maxRetries:   maxRetries,
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
	}
}

// Start begins the background processing of outbox events.
func (p *OutboxProcessor) Start(ctx context.Context) {
	go func() {
		defer close(p.doneChan)
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("outbox processor context cancelled")
				return
			case <-p.stopChan:
				log.Println("outbox processor stopped")
				return
			case <-ticker.C:
				if err := p.processEvents(ctx); err != nil {
					log.Printf("error processing outbox events: %v", err)
				}
			}
		}
	}()
}

// Stop gracefully shuts down the outbox processor.
func (p *OutboxProcessor) Stop() {
	close(p.stopChan)
	<-p.doneChan
}

// processEvents retrieves pending events and publishes them.
func (p *OutboxProcessor) processEvents(ctx context.Context) error {
	events, err := p.repository.GetPendingEvents(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	for _, event := range events {
		if event.RetryCount >= p.maxRetries {
			log.Printf("event %s exceeded max retries, skipping", event.ID)
			continue
		}

		if err := p.publisher.Publish(ctx, event); err != nil {
			log.Printf("failed to publish event %s: %v", event.ID, err)
			continue
		}

		if err := p.repository.MarkAsPublished(ctx, event.ID); err != nil {
			log.Printf("failed to mark event %s as published: %v", event.ID, err)
			continue
		}

		log.Printf("successfully published event %s (type: %s)", event.ID, event.EventType)
	}

	return nil
}

// SerializeEventPayload converts an event payload to JSON bytes.
func SerializeEventPayload(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event payload: %w", err)
	}
	return data, nil
}

// CreateOutboxEvent creates a new outbox event with generated ID and timestamp.
func CreateOutboxEvent(aggregateID, eventType string, payload interface{}) (*OutboxEvent, error) {
	payloadBytes, err := SerializeEventPayload(payload)
	if err != nil {
		return nil, err
	}

	return &OutboxEvent{
		ID:          uuid.New().String(),
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payloadBytes,
		CreatedAt:   time.Now(),
		IsPublished: false,
		RetryCount:  0,
	}, nil
}
