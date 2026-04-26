// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Observer pattern
// for real-time event notification systems with thread-safety, error handling,
// and flexible event filtering capabilities.
package behavioral

import (
	"fmt"
	"sync"
	"time"
)

// Event represents an event that can be observed.
// It provides a flexible interface for different event types.
type Event interface {
	// Type returns the event type identifier
	Type() string
	// Data returns the event payload
	Data() interface{}
	// Timestamp returns when the event was created
	Timestamp() time.Time
	// Metadata returns additional event metadata
	Metadata() map[string]interface{}
}

// Observer defines the interface for objects that should be notified
// of changes in the subject they're observing.
type Observer interface {
	// Update receives notifications when the subject's state changes
	Update(event Event) error
	// ID returns a unique identifier for the observer
	ID() string
}

// Subject defines the interface for objects that can be observed.
// It manages observer registration and notification.
type Subject interface {
	// Attach adds an observer to the subject
	Attach(observer Observer) error
	// Detach removes an observer from the subject
	Detach(observerID string) error
	// Notify sends an event to all registered observers
	Notify(event Event) error
	// NotifyFiltered sends an event to observers matching the filter
	NotifyFiltered(event Event, filter func(Observer) bool) error
	// GetObserverCount returns the number of registered observers
	GetObserverCount() int
}

// ConcreteEvent is a concrete implementation of the Event interface.
type ConcreteEvent struct {
	eventType string
	data      interface{}
	timestamp time.Time
	metadata  map[string]interface{}
}

// NewConcreteEvent creates a new event with the given type and data.
func NewConcreteEvent(eventType string, data interface{}) *ConcreteEvent {
	return &ConcreteEvent{
		eventType: eventType,
		data:      data,
		timestamp: time.Now(),
		metadata:  make(map[string]interface{}),
	}
}

// Type returns the event type identifier.
func (e *ConcreteEvent) Type() string {
	return e.eventType
}

// Data returns the event payload.
func (e *ConcreteEvent) Data() interface{} {
	return e.data
}

// Timestamp returns when the event was created.
func (e *ConcreteEvent) Timestamp() time.Time {
	return e.timestamp
}

// Metadata returns additional event metadata.
func (e *ConcreteEvent) Metadata() map[string]interface{} {
	return e.metadata
}

// SetMetadata adds metadata to the event.
func (e *ConcreteEvent) SetMetadata(key string, value interface{}) {
	e.metadata[key] = value
}

// BaseSubject provides a thread-safe implementation of the Subject interface.
// It can be embedded in concrete subjects to provide observer management.
type BaseSubject struct {
	observers map[string]Observer
	mu        sync.RWMutex
}

// NewBaseSubject creates a new BaseSubject instance.
func NewBaseSubject() *BaseSubject {
	return &BaseSubject{
		observers: make(map[string]Observer),
	}
}

// Attach adds an observer to the subject.
// Returns an error if the observer is nil or already attached.
func (s *BaseSubject) Attach(observer Observer) error {
	if observer == nil {
		return fmt.Errorf("cannot attach nil observer")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	observerID := observer.ID()
	if _, exists := s.observers[observerID]; exists {
		return fmt.Errorf("observer with ID '%s' is already attached", observerID)
	}

	s.observers[observerID] = observer
	return nil
}

// Detach removes an observer from the subject.
// Returns an error if the observer is not found.
func (s *BaseSubject) Detach(observerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.observers[observerID]; !exists {
		return fmt.Errorf("observer with ID '%s' not found", observerID)
	}

	delete(s.observers, observerID)
	return nil
}

// Notify sends an event to all registered observers concurrently.
// Errors from individual observers are collected and returned.
func (s *BaseSubject) Notify(event Event) error {
	s.mu.RLock()
	observers := make([]Observer, 0, len(s.observers))
	for _, observer := range s.observers {
		observers = append(observers, observer)
	}
	s.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(observers))

	for _, observer := range observers {
		wg.Add(1)
		go func(obs Observer) {
			defer wg.Done()
			if err := obs.Update(event); err != nil {
				errChan <- fmt.Errorf("observer %s error: %w", obs.ID(), err)
			}
		}(observer)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors occurred: %v", errs)
	}

	return nil
}

// NotifyFiltered sends an event to observers matching the filter function.
func (s *BaseSubject) NotifyFiltered(event Event, filter func(Observer) bool) error {
	s.mu.RLock()
	observers := make([]Observer, 0)
	for _, observer := range s.observers {
		if filter(observer) {
			observers = append(observers, observer)
		}
	}
	s.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(observers))

	for _, observer := range observers {
		wg.Add(1)
		go func(obs Observer) {
			defer wg.Done()
			if err := obs.Update(event); err != nil {
				errChan <- fmt.Errorf("observer %s error: %w", obs.ID(), err)
			}
		}(observer)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("filtered notification errors occurred: %v", errs)
	}

	return nil
}

// GetObserverCount returns the number of registered observers.
func (s *BaseSubject) GetObserverCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.observers)
}

// StockTicker is a concrete subject that publishes stock price updates.
type StockTicker struct {
	*BaseSubject
	symbol       string
	currentPrice float64
	mu           sync.RWMutex
}

// NewStockTicker creates a new StockTicker for the given symbol.
func NewStockTicker(symbol string, initialPrice float64) *StockTicker {
	return &StockTicker{
		BaseSubject:  NewBaseSubject(),
		symbol:       symbol,
		currentPrice: initialPrice,
	}
}

// UpdatePrice updates the stock price and notifies all observers.
func (st *StockTicker) UpdatePrice(newPrice float64) error {
	st.mu.Lock()
	oldPrice := st.currentPrice
	st.currentPrice = newPrice
	st.mu.Unlock()

	event := NewConcreteEvent("price_update", map[string]interface{}{
		"symbol":    st.symbol,
		"oldPrice":  oldPrice,
		"newPrice":  newPrice,
		"change":    newPrice - oldPrice,
		"changePct": ((newPrice - oldPrice) / oldPrice) * 100,
	})
	event.SetMetadata("source", "stock_ticker")
	event.SetMetadata("symbol", st.symbol)

	return st.Notify(event)
}

// EmailNotifier is a concrete observer that sends email notifications.
type EmailNotifier struct {
	id           string
	emailAddress string
	interestedIn []string
	mu           sync.Mutex
}

// NewEmailNotifier creates a new EmailNotifier.
func NewEmailNotifier(id, email string, interestedSymbols []string) *EmailNotifier {
	return &EmailNotifier{
		id:           id,
		emailAddress: email,
		interestedIn: interestedSymbols,
	}
}

// ID returns the unique identifier for this observer.
func (en *EmailNotifier) ID() string {
	return en.id
}

// Update processes the event and sends an email if interested.
func (en *EmailNotifier) Update(event Event) error {
	en.mu.Lock()
	defer en.mu.Unlock()

	// Filter by symbol if specified
	if symbol, ok := event.Metadata()["symbol"].(string); ok {
		interested := false
		for _, s := range en.interestedIn {
			if s == symbol {
				interested = true
				break
			}
		}
		if !interested {
			return nil
		}
	}

	// Simulate sending email
	fmt.Printf("[EMAIL to %s] Event: %s, Data: %v, Time: %s\n",
		en.emailAddress, event.Type(), event.Data(), event.Timestamp().Format(time.RFC3339))

	return nil
}

// MobileNotifier is a concrete observer that sends push notifications to mobile devices.
type MobileNotifier struct {
	id             string
	deviceToken    string
	priceChangeMin float64
	mu             sync.Mutex
}

// NewMobileNotifier creates a new MobileNotifier with a minimum price change threshold.
func NewMobileNotifier(id, deviceToken string, minPriceChange float64) *MobileNotifier {
	return &MobileNotifier{
		id:             id,
		deviceToken:    deviceToken,
		priceChangeMin: minPriceChange,
	}
}

// ID returns the unique identifier for this observer.
func (mn *MobileNotifier) ID() string {
	return mn.id
}

// Update processes the event and sends a push notification if threshold is met.
func (mn *MobileNotifier) Update(event Event) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	if event.Type() == "price_update" {
		if data, ok := event.Data().(map[string]interface{}); ok {
			if changePct, ok := data["changePct"].(float64); ok {
				if changePct >= mn.priceChangeMin || changePct <= -mn.priceChangeMin {
					fmt.Printf("[PUSH to %s] %s changed by %.2f%%, Time: %s\n",
						mn.deviceToken, data["symbol"], changePct, event.Timestamp().Format(time.RFC3339))
				}
			}
		}
	}

	return nil
}

// AnalyticsCollector is a concrete observer that collects events for analytics.
type AnalyticsCollector struct {
	id     string
	events []Event
	mu     sync.Mutex
}

// NewAnalyticsCollector creates a new AnalyticsCollector.
func NewAnalyticsCollector(id string) *AnalyticsCollector {
	return &AnalyticsCollector{
		id:     id,
		events: make([]Event, 0),
	}
}

// ID returns the unique identifier for this observer.
func (ac *AnalyticsCollector) ID() string {
	return ac.id
}

// Update processes and stores the event for analytics.
func (ac *AnalyticsCollector) Update(event Event) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.events = append(ac.events, event)
	fmt.Printf("[ANALYTICS] Collected event: %s, Total events: %d\n", event.Type(), len(ac.events))

	return nil
}

// GetEventCount returns the number of collected events.
func (ac *AnalyticsCollector) GetEventCount() int {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return len(ac.events)
}
