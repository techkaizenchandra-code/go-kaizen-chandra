// Package persitance provides event sourcing implementation for microservices.
// Event sourcing is a pattern where state changes are stored as a sequence of events.
// The current state is derived by replaying these events from the beginning.
package event_sourcing

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event that has occurred in the system.
type Event interface {
	GetEventID() string
	GetAggregateID() string
	GetEventType() string
	GetTimestamp() time.Time
	GetVersion() int
	GetData() ([]byte, error)
}

// BaseEvent provides common event fields for all domain events.
type BaseEvent struct {
	EventID     string    `json:"event_id"`
	AggregateID string    `json:"aggregate_id"`
	EventType   string    `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int       `json:"version"`
	Data        []byte    `json:"data"`
}

// GetEventID returns the unique identifier for this event.
func (e *BaseEvent) GetEventID() string {
	return e.EventID
}

// GetAggregateID returns the ID of the aggregate this event belongs to.
func (e *BaseEvent) GetAggregateID() string {
	return e.AggregateID
}

// GetEventType returns the type of this event.
func (e *BaseEvent) GetEventType() string {
	return e.EventType
}

// GetTimestamp returns when this event occurred.
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetVersion returns the version of the aggregate after this event.
func (e *BaseEvent) GetVersion() int {
	return e.Version
}

// GetData returns the serialized event data.
func (e *BaseEvent) GetData() ([]byte, error) {
	return e.Data, nil
}

// EventStore defines the interface for event persistence.
type EventStore interface {
	// Save persists an event to the store
	Save(event Event) error
	// GetByAggregateID retrieves all events for a specific aggregate
	GetByAggregateID(aggregateID string) ([]Event, error)
	// GetByAggregateIDAfterVersion retrieves events after a specific version
	GetByAggregateIDAfterVersion(aggregateID string, version int) ([]Event, error)
}

// InMemoryEventStore is a thread-safe in-memory implementation of EventStore.
// For production, use a persistent store like PostgreSQL, EventStoreDB, or Kafka.
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]Event // aggregateID -> events
}

// NewInMemoryEventStore creates a new in-memory event store.
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]Event),
	}
}

// Save stores an event in the event store with optimistic concurrency control.
func (s *InMemoryEventStore) Save(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aggregateID := event.GetAggregateID()
	existingEvents := s.events[aggregateID]

	// Optimistic concurrency check
	expectedVersion := len(existingEvents)
	if event.GetVersion() != expectedVersion+1 {
		return fmt.Errorf("concurrency conflict: expected version %d, got %d", expectedVersion+1, event.GetVersion())
	}

	s.events[aggregateID] = append(existingEvents, event)
	return nil
}

// GetByAggregateID retrieves all events for a given aggregate.
func (s *InMemoryEventStore) GetByAggregateID(aggregateID string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[aggregateID]
	if events == nil {
		return []Event{}, nil
	}

	// Return a copy to prevent external modifications
	result := make([]Event, len(events))
	copy(result, events)
	return result, nil
}

// GetByAggregateIDAfterVersion retrieves events after a specific version.
func (s *InMemoryEventStore) GetByAggregateIDAfterVersion(aggregateID string, version int) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[aggregateID]
	if events == nil {
		return []Event{}, nil
	}

	var result []Event
	for _, event := range events {
		if event.GetVersion() > version {
			result = append(result, event)
		}
	}
	return result, nil
}

// EventHandler processes events.
type EventHandler interface {
	Handle(event Event) error
}

// EventBus defines the interface for publishing and subscribing to events.
type EventBus interface {
	Publish(event Event) error
	Subscribe(eventType string, handler EventHandler) error
}

// InMemoryEventBus is a simple in-memory event bus implementation.
// For production, use message brokers like RabbitMQ, Kafka, or NATS.
type InMemoryEventBus struct {
	mu         sync.RWMutex
	handlers   map[string][]EventHandler // eventType -> handlers
	handlersMu sync.Mutex
}

// NewInMemoryEventBus creates a new in-memory event bus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
	}
}

// Publish sends an event to all subscribed handlers asynchronously.
func (b *InMemoryEventBus) Publish(event Event) error {
	b.mu.RLock()
	handlers := b.handlers[event.GetEventType()]
	b.mu.RUnlock()

	// Publish to handlers asynchronously
	for _, handler := range handlers {
		go func(h EventHandler, e Event) {
			// In production, add proper error handling and retry logic
			_ = h.Handle(e)
		}(handler, event)
	}

	return nil
}

// Subscribe registers a handler for a specific event type.
func (b *InMemoryEventBus) Subscribe(eventType string, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	return nil
}

// AggregateRoot represents the root entity of an aggregate.
type AggregateRoot interface {
	GetID() string
	GetVersion() int
	GetUncommittedEvents() []Event
	MarkEventsAsCommitted()
	LoadFromHistory(events []Event) error
}

// BaseAggregate provides common functionality for all aggregates.
type BaseAggregate struct {
	ID                string
	Version           int
	uncommittedEvents []Event
}

func (a *BaseAggregate) GetID() string {
	return a.ID
}

func (a *BaseAggregate) GetVersion() int {
	return a.Version
}

// NewBaseAggregate creates a new base aggregate.
func NewBaseAggregate(id string) *BaseAggregate {
	return &BaseAggregate{
		ID:                id,
		Version:           0,
		uncommittedEvents: make([]Event, 0),
	}
}

// TrackChange records an event that hasn't been persisted yet.
func (a *BaseAggregate) TrackChange(event Event) {
	a.uncommittedEvents = append(a.uncommittedEvents, event)
	a.Version++
}

// GetUncommittedEvents returns events that haven't been saved to the event store.
func (a *BaseAggregate) GetUncommittedEvents() []Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents removes all uncommitted events.
func (a *BaseAggregate) ClearUncommittedEvents() {
	a.uncommittedEvents = make([]Event, 0)
}

// MarkEventsAsCommitted clears uncommitted events after successful persistence.
func (a *BaseAggregate) MarkEventsAsCommitted() {
	a.ClearUncommittedEvents()
}

// LoadFromHistory rebuilds aggregate state from historical events.
func (a *BaseAggregate) LoadFromHistory(events []Event) error {
	for _, event := range events {
		a.Version = event.GetVersion()
	}
	return nil
}

// Snapshot represents a point-in-time state of an aggregate.
type Snapshot interface {
	GetAggregateID() string
	GetVersion() int
	GetTimestamp() time.Time
	GetState() []byte
}

// AggregateSnapshot stores the state of an aggregate at a specific version.
type AggregateSnapshot struct {
	AggregateID string    `json:"aggregate_id"`
	Version     int       `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	State       []byte    `json:"state"`
}

// GetAggregateID returns the aggregate ID.
func (s *AggregateSnapshot) GetAggregateID() string {
	return s.AggregateID
}

// GetVersion returns the version at which the snapshot was taken.
func (s *AggregateSnapshot) GetVersion() int {
	return s.Version
}

// GetTimestamp returns when the snapshot was created.
func (s *AggregateSnapshot) GetTimestamp() time.Time {
	return s.Timestamp
}

// GetState returns the serialized aggregate state.
func (s *AggregateSnapshot) GetState() []byte {
	return s.State
}

// SnapshotStore defines the interface for snapshot persistence.
type SnapshotStore interface {
	SaveSnapshot(snapshot Snapshot) error
	GetSnapshot(aggregateID string) (Snapshot, error)
}

// InMemorySnapshotStore is an in-memory snapshot store implementation.
type InMemorySnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[string]Snapshot // aggregateID -> latest snapshot
}

// NewInMemorySnapshotStore creates a new in-memory snapshot store.
func NewInMemorySnapshotStore() *InMemorySnapshotStore {
	return &InMemorySnapshotStore{
		snapshots: make(map[string]Snapshot),
	}
}

// SaveSnapshot persists a snapshot.
func (s *InMemorySnapshotStore) SaveSnapshot(snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshots[snapshot.GetAggregateID()] = snapshot
	return nil
}

// GetSnapshot retrieves the latest snapshot for an aggregate.
func (s *InMemorySnapshotStore) GetSnapshot(aggregateID string) (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, exists := s.snapshots[aggregateID]
	if !exists {
		return nil, errors.New("snapshot not found")
	}
	return snapshot, nil
}

// Repository defines the interface for aggregate persistence.
type Repository interface {
	Save(aggregate AggregateRoot) error
	GetByID(id string) (AggregateRoot, error)
}

// EventSourcedRepository implements Repository using event sourcing.
type EventSourcedRepository struct {
	eventStore    EventStore
	snapshotStore SnapshotStore
	eventBus      EventBus
	// Snapshot interval: create snapshot every N events
	snapshotInterval int
}

// NewEventSourcedRepository creates a new event-sourced repository.
func NewEventSourcedRepository(eventStore EventStore, snapshotStore SnapshotStore, eventBus EventBus, snapshotInterval int) *EventSourcedRepository {
	return &EventSourcedRepository{
		eventStore:       eventStore,
		snapshotStore:    snapshotStore,
		eventBus:         eventBus,
		snapshotInterval: snapshotInterval,
	}
}

// Save persists an aggregate by saving its uncommitted events.
func (r *EventSourcedRepository) Save(aggregate AggregateRoot) error {
	events := aggregate.GetUncommittedEvents()

	// Save all uncommitted events
	for _, event := range events {
		if err := r.eventStore.Save(event); err != nil {
			return fmt.Errorf("failed to save event: %w", err)
		}

		// Publish event to event bus
		if err := r.eventBus.Publish(event); err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}
	}

	// Mark events as committed
	aggregate.MarkEventsAsCommitted()

	// Create snapshot if needed
	if r.snapshotInterval > 0 && aggregate.GetVersion()%r.snapshotInterval == 0 {
		// In production, implement snapshot creation based on aggregate type
		// For now, this is a placeholder
	}

	return nil
}

// GetByID retrieves an aggregate by replaying its events (with snapshot optimization).
func (r *EventSourcedRepository) GetByID(id string) (AggregateRoot, error) {
	var startVersion int

	// Try to load from snapshot first
	snapshot, err := r.snapshotStore.GetSnapshot(id)
	if err == nil && snapshot != nil {
		startVersion = snapshot.GetVersion()
		// In production, deserialize snapshot and initialize aggregate
	}

	// Load events after snapshot version
	events, err := r.eventStore.GetByAggregateIDAfterVersion(id, startVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 && startVersion == 0 {
		return nil, errors.New("aggregate not found")
	}

	// Create aggregate and replay events
	// In production, use factory pattern based on aggregate type
	aggregate := NewBaseAggregate(id)
	if err := aggregate.LoadFromHistory(events); err != nil {
		return nil, fmt.Errorf("failed to load from history: %w", err)
	}

	return aggregate, nil
}

// Example Domain Events

// AccountCreatedEvent represents an account creation event.
type AccountCreatedEvent struct {
	BaseEvent
	Owner          string  `json:"owner"`
	InitialBalance float64 `json:"initial_balance"`
}

// MoneyDepositedEvent represents a money deposit event.
type MoneyDepositedEvent struct {
	BaseEvent
	Amount float64 `json:"amount"`
}

// MoneyWithdrawnEvent represents a money withdrawal event.
type MoneyWithdrawnEvent struct {
	BaseEvent
	Amount float64 `json:"amount"`
}

// Example Aggregate

// Account is an example aggregate demonstrating event sourcing.
type Account struct {
	*BaseAggregate
	Owner   string
	Balance float64
}

// NewAccount creates a new account aggregate.
func NewAccount(id string) *Account {
	return &Account{
		BaseAggregate: NewBaseAggregate(id),
	}
}

// CreateAccount handles the account creation command.
func (a *Account) CreateAccount(owner string, initialBalance float64) error {
	if a.Version > 0 {
		return errors.New("account already exists")
	}

	eventData, _ := json.Marshal(AccountCreatedEvent{
		Owner:          owner,
		InitialBalance: initialBalance,
	})

	event := &BaseEvent{
		EventID:     uuid.New().String(),
		AggregateID: a.ID,
		EventType:   "AccountCreated",
		Timestamp:   time.Now(),
		Version:     a.Version + 1,
		Data:        eventData,
	}

	a.ApplyEvent(event)
	a.TrackChange(event)
	return nil
}

// Deposit handles the deposit command.
func (a *Account) Deposit(amount float64) error {
	if amount <= 0 {
		return errors.New("deposit amount must be positive")
	}

	eventData, _ := json.Marshal(MoneyDepositedEvent{
		Amount: amount,
	})

	event := &BaseEvent{
		EventID:     uuid.New().String(),
		AggregateID: a.ID,
		EventType:   "MoneyDeposited",
		Timestamp:   time.Now(),
		Version:     a.Version + 1,
		Data:        eventData,
	}

	a.ApplyEvent(event)
	a.TrackChange(event)
	return nil
}

// Withdraw handles the withdrawal command.
func (a *Account) Withdraw(amount float64) error {
	if amount <= 0 {
		return errors.New("withdrawal amount must be positive")
	}

	if a.Balance < amount {
		return errors.New("insufficient funds")
	}

	eventData, _ := json.Marshal(MoneyWithdrawnEvent{
		Amount: amount,
	})

	event := &BaseEvent{
		EventID:     uuid.New().String(),
		AggregateID: a.ID,
		EventType:   "MoneyWithdrawn",
		Timestamp:   time.Now(),
		Version:     a.Version + 1,
		Data:        eventData,
	}

	a.ApplyEvent(event)
	a.TrackChange(event)
	return nil
}

// ApplyEvent applies an event to the aggregate state.
func (a *Account) ApplyEvent(event Event) {
	switch event.GetEventType() {
	case "AccountCreated":
		a.applyAccountCreated(event)
	case "MoneyDeposited":
		a.applyMoneyDeposited(event)
	case "MoneyWithdrawn":
		a.applyMoneyWithdrawn(event)
	}
}

// applyAccountCreated applies the AccountCreated event.
func (a *Account) applyAccountCreated(event Event) {
	data, _ := event.GetData()
	var accountCreated AccountCreatedEvent
	json.Unmarshal(data, &accountCreated)

	a.Owner = accountCreated.Owner
	a.Balance = accountCreated.InitialBalance
}

// applyMoneyDeposited applies the MoneyDeposited event.
func (a *Account) applyMoneyDeposited(event Event) {
	data, _ := event.GetData()
	var moneyDeposited MoneyDepositedEvent
	json.Unmarshal(data, &moneyDeposited)

	a.Balance += moneyDeposited.Amount
}

// applyMoneyWithdrawn applies the MoneyWithdrawn event.
func (a *Account) applyMoneyWithdrawn(event Event) {
	data, _ := event.GetData()
	var moneyWithdrawn MoneyWithdrawnEvent
	json.Unmarshal(data, &moneyWithdrawn)

	a.Balance -= moneyWithdrawn.Amount
}

// GetBalance returns the current balance (query).
func (a *Account) GetBalance() float64 {
	return a.Balance
}

// CreateSnapshot creates a snapshot of the current account state.
func (a *Account) CreateSnapshot() (*AggregateSnapshot, error) {
	stateData, err := json.Marshal(map[string]interface{}{
		"owner":   a.Owner,
		"balance": a.Balance,
	})
	if err != nil {
		return nil, err
	}

	return &AggregateSnapshot{
		AggregateID: a.ID,
		Version:     a.Version,
		Timestamp:   time.Now(),
		State:       stateData,
	}, nil
}

// LoadFromSnapshot restores the account state from a snapshot.
func (a *Account) LoadFromSnapshot(snapshot Snapshot) error {
	var state map[string]interface{}
	if err := json.Unmarshal(snapshot.GetState(), &state); err != nil {
		return err
	}

	a.Owner = state["owner"].(string)
	a.Balance = state["balance"].(float64)
	a.Version = snapshot.GetVersion()

	return nil
}
