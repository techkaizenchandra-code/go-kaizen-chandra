// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the State pattern
// using an order processing system example with different order states.
package behavioral

import (
	"fmt"
	"time"
)

// State defines the interface for all concrete states in the state machine.
// Each state handles specific operations and determines valid state transitions.
type State interface {
	// Name returns the name of the current state
	Name() string
	// Handle processes the order in the current state
	Handle(ctx OrderContext) error
	// Next transitions to the next valid state
	Next(ctx OrderContext) error
	// Cancel attempts to cancel the order from the current state
	Cancel(ctx OrderContext) error
	// GetAllowedTransitions returns the list of states that can be transitioned to
	GetAllowedTransitions() []string
}

// OrderContext defines the interface for the context that maintains the current state.
// It delegates state-specific operations to the current state object.
type OrderContext interface {
	// GetState returns the current state
	GetState() State
	// SetState sets a new state with validation
	SetState(state State) error
	// GetOrderID returns the order identifier
	GetOrderID() string
	// GetTimestamp returns when the order was created
	GetTimestamp() time.Time
	// GetStateHistory returns the history of state transitions
	GetStateHistory() []StateTransition
	// Process processes the order in its current state
	Process() error
	// Cancel cancels the order
	Cancel() error
	// PrintStatus prints the current order status
	PrintStatus()
}

// StateTransition records a state transition event.
type StateTransition struct {
	FromState string
	ToState   string
	Timestamp time.Time
}

// Order represents a concrete implementation of OrderContext.
// It maintains the current state and handles state transitions.
type Order struct {
	orderID      string
	currentState State
	createdAt    time.Time
	history      []StateTransition
}

// PendingState represents the initial state when an order is created.
type PendingState struct{}

// ProcessingState represents the state when an order is being processed.
type ProcessingState struct{}

// ShippedState represents the state when an order has been shipped.
type ShippedState struct{}

// DeliveredState represents the final state when an order is delivered.
type DeliveredState struct{}

// CancelledState represents the state when an order is cancelled.
type CancelledState struct{}

// NewOrder creates a new Order instance with validation.
// Returns an error if the orderID is empty.
func NewOrder(orderID string) (*Order, error) {
	if orderID == "" {
		return nil, fmt.Errorf("order ID cannot be empty")
	}

	order := &Order{
		orderID:      orderID,
		currentState: &PendingState{},
		createdAt:    time.Now(),
		history:      make([]StateTransition, 0),
	}

	return order, nil
}

// GetState returns the current state of the order.
func (o *Order) GetState() State {
	return o.currentState
}

// SetState sets a new state for the order with validation.
// Returns an error if the state is nil or transition is not allowed.
func (o *Order) SetState(state State) error {
	if state == nil {
		return fmt.Errorf("cannot set nil state")
	}

	oldState := o.currentState.Name()
	newState := state.Name()

	// Check if transition is allowed
	allowedTransitions := o.currentState.GetAllowedTransitions()
	isAllowed := false
	for _, allowed := range allowedTransitions {
		if allowed == newState {
			isAllowed = true
			break
		}
	}

	if !isAllowed && newState != "Cancelled" {
		return fmt.Errorf("invalid state transition from '%s' to '%s'", oldState, newState)
	}

	// Record the transition
	o.history = append(o.history, StateTransition{
		FromState: oldState,
		ToState:   newState,
		Timestamp: time.Now(),
	})

	o.currentState = state
	return nil
}

// GetOrderID returns the order identifier.
func (o *Order) GetOrderID() string {
	return o.orderID
}

// GetTimestamp returns when the order was created.
func (o *Order) GetTimestamp() time.Time {
	return o.createdAt
}

// GetStateHistory returns the history of state transitions.
func (o *Order) GetStateHistory() []StateTransition {
	// Return a copy to prevent external modifications
	history := make([]StateTransition, len(o.history))
	copy(history, o.history)
	return history
}

// Process processes the order in its current state.
func (o *Order) Process() error {
	return o.currentState.Handle(o)
}

// Cancel attempts to cancel the order.
func (o *Order) Cancel() error {
	return o.currentState.Cancel(o)
}

// PrintStatus prints the current order status with history.
func (o *Order) PrintStatus() {
	fmt.Printf("📦 Order ID: %s\n", o.orderID)
	fmt.Printf("🕐 Created: %s\n", o.createdAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("📊 Current State: %s\n", o.currentState.Name())

	if len(o.history) > 0 {
		fmt.Println("📜 State History:")
		for i, transition := range o.history {
			fmt.Printf("   %d. %s → %s (%s)\n",
				i+1,
				transition.FromState,
				transition.ToState,
				transition.Timestamp.Format("15:04:05"))
		}
	}
	fmt.Println()
}

// Name returns the name of the PendingState.
func (s *PendingState) Name() string {
	return "Pending"
}

// Handle processes the order in the PendingState.
func (s *PendingState) Handle(ctx OrderContext) error {
	fmt.Printf("🔄 Processing order %s...\n", ctx.GetOrderID())
	return s.Next(ctx)
}

// Next transitions to ProcessingState.
func (s *PendingState) Next(ctx OrderContext) error {
	fmt.Println("✅ Order confirmed, moving to processing")
	return ctx.SetState(&ProcessingState{})
}

// Cancel cancels the order from PendingState.
func (s *PendingState) Cancel(ctx OrderContext) error {
	fmt.Println("❌ Cancelling pending order")
	return ctx.SetState(&CancelledState{})
}

// GetAllowedTransitions returns valid transitions from PendingState.
func (s *PendingState) GetAllowedTransitions() []string {
	return []string{"Processing"}
}

// Name returns the name of the ProcessingState.
func (s *ProcessingState) Name() string {
	return "Processing"
}

// Handle processes the order in the ProcessingState.
func (s *ProcessingState) Handle(ctx OrderContext) error {
	fmt.Printf("📦 Preparing order %s for shipment...\n", ctx.GetOrderID())
	return s.Next(ctx)
}

// Next transitions to ShippedState.
func (s *ProcessingState) Next(ctx OrderContext) error {
	fmt.Println("🚚 Order shipped")
	return ctx.SetState(&ShippedState{})
}

// Cancel cancels the order from ProcessingState.
func (s *ProcessingState) Cancel(ctx OrderContext) error {
	fmt.Println("❌ Cancelling order in processing")
	return ctx.SetState(&CancelledState{})
}

// GetAllowedTransitions returns valid transitions from ProcessingState.
func (s *ProcessingState) GetAllowedTransitions() []string {
	return []string{"Shipped"}
}

// Name returns the name of the ShippedState.
func (s *ShippedState) Name() string {
	return "Shipped"
}

// Handle processes the order in the ShippedState.
func (s *ShippedState) Handle(ctx OrderContext) error {
	fmt.Printf("🚚 Order %s is in transit...\n", ctx.GetOrderID())
	return s.Next(ctx)
}

// Next transitions to DeliveredState.
func (s *ShippedState) Next(ctx OrderContext) error {
	fmt.Println("✅ Order delivered")
	return ctx.SetState(&DeliveredState{})
}

// Cancel returns an error as shipped orders cannot be cancelled.
func (s *ShippedState) Cancel(ctx OrderContext) error {
	return fmt.Errorf("cannot cancel order that has already been shipped")
}

// GetAllowedTransitions returns valid transitions from ShippedState.
func (s *ShippedState) GetAllowedTransitions() []string {
	return []string{"Delivered"}
}

// Name returns the name of the DeliveredState.
func (s *DeliveredState) Name() string {
	return "Delivered"
}

// Handle processes the order in the DeliveredState.
func (s *DeliveredState) Handle(ctx OrderContext) error {
	fmt.Printf("✅ Order %s has been delivered\n", ctx.GetOrderID())
	return nil
}

// Next returns an error as Delivered is a final state.
func (s *DeliveredState) Next(ctx OrderContext) error {
	return fmt.Errorf("order is already delivered, no further transitions allowed")
}

// Cancel returns an error as delivered orders cannot be cancelled.
func (s *DeliveredState) Cancel(ctx OrderContext) error {
	return fmt.Errorf("cannot cancel order that has already been delivered")
}

// GetAllowedTransitions returns empty slice as Delivered is a final state.
func (s *DeliveredState) GetAllowedTransitions() []string {
	return []string{}
}

// Name returns the name of the CancelledState.
func (s *CancelledState) Name() string {
	return "Cancelled"
}

// Handle processes the order in the CancelledState.
func (s *CancelledState) Handle(ctx OrderContext) error {
	fmt.Printf("❌ Order %s has been cancelled\n", ctx.GetOrderID())
	return nil
}

// Next returns an error as Cancelled is a final state.
func (s *CancelledState) Next(ctx OrderContext) error {
	return fmt.Errorf("order is cancelled, no further transitions allowed")
}

// Cancel returns an error as the order is already cancelled.
func (s *CancelledState) Cancel(ctx OrderContext) error {
	return fmt.Errorf("order is already cancelled")
}

// GetAllowedTransitions returns empty slice as Cancelled is a final state.
func (s *CancelledState) GetAllowedTransitions() []string {
	return []string{}
}
