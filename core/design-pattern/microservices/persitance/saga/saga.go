// Package persitance provides production-grade implementations of microservices persistence patterns.
// This file implements the Saga pattern for managing distributed transactions across microservices.
//
// The Saga pattern ensures data consistency in distributed systems by coordinating a sequence of
// local transactions. If any step fails, compensating transactions are executed to undo completed steps.
//
// This implementation supports:
// - Orchestration-based saga coordination
// - Forward transaction execution
// - Backward compensation on failure
// - Persistent saga state management
// - Thread-safe concurrent execution
package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SagaStatus represents the current state of a saga execution
type SagaStatus string

const (
	// SagaStatusPending indicates saga is created but not started
	SagaStatusPending SagaStatus = "PENDING"
	// SagaStatusInProgress indicates saga is currently executing
	SagaStatusInProgress SagaStatus = "IN_PROGRESS"
	// SagaStatusCompleted indicates all steps completed successfully
	SagaStatusCompleted SagaStatus = "COMPLETED"
	// SagaStatusFailed indicates saga failed and requires compensation
	SagaStatusFailed SagaStatus = "FAILED"
	// SagaStatusCompensating indicates compensation is in progress
	SagaStatusCompensating SagaStatus = "COMPENSATING"
	// SagaStatusCompensated indicates all compensations completed
	SagaStatusCompensated SagaStatus = "COMPENSATED"
)

// SagaStep represents a single step in a saga transaction
type SagaStep interface {
	// Execute performs the forward transaction
	Execute(ctx context.Context) error
	// Compensate undoes the transaction if saga fails
	Compensate(ctx context.Context) error
}

// SagaStepResult holds the result of a saga step execution
type SagaStepResult struct {
	StepName    string
	Success     bool
	Error       error
	ExecutedAt  time.Time
	CompletedAt time.Time
}

// SagaDefinition defines the structure of a saga
type SagaDefinition struct {
	ID       string
	Name     string
	Steps    []SagaStep
	Metadata map[string]interface{}
}

// SagaExecution represents a running saga instance
type SagaExecution struct {
	ID            string
	DefinitionID  string
	Status        SagaStatus
	CurrentStep   int
	StepResults   []SagaStepResult
	StartedAt     time.Time
	CompletedAt   *time.Time
	CompensatedAt *time.Time
	Error         error
	Metadata      map[string]interface{}
	mu            sync.RWMutex
}

// SagaOrchestrator coordinates saga execution
type SagaOrchestrator struct {
	definition *SagaDefinition
	execution  *SagaExecution
	repository SagaRepository
	mu         sync.Mutex
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(definition *SagaDefinition, repository SagaRepository) *SagaOrchestrator {
	execution := &SagaExecution{
		ID:           uuid.New().String(),
		DefinitionID: definition.ID,
		Status:       SagaStatusPending,
		CurrentStep:  0,
		StepResults:  make([]SagaStepResult, 0),
		StartedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	return &SagaOrchestrator{
		definition: definition,
		execution:  execution,
		repository: repository,
	}
}

// Execute runs the saga forward, executing all steps
func (o *SagaOrchestrator) Execute(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.execution.mu.Lock()
	o.execution.Status = SagaStatusInProgress
	o.execution.mu.Unlock()

	if err := o.repository.Save(ctx, o.execution); err != nil {
		return fmt.Errorf("failed to save saga execution: %w", err)
	}

	for i, step := range o.definition.Steps {
		o.execution.mu.Lock()
		o.execution.CurrentStep = i
		o.execution.mu.Unlock()

		result := SagaStepResult{
			StepName:   fmt.Sprintf("Step_%d", i),
			ExecutedAt: time.Now(),
		}

		err := step.Execute(ctx)
		result.CompletedAt = time.Now()

		if err != nil {
			result.Success = false
			result.Error = err

			o.execution.mu.Lock()
			o.execution.StepResults = append(o.execution.StepResults, result)
			o.execution.Status = SagaStatusFailed
			o.execution.Error = err
			o.execution.mu.Unlock()

			if saveErr := o.repository.Update(ctx, o.execution); saveErr != nil {
				return fmt.Errorf("failed to update saga execution: %w", saveErr)
			}

			// Execute compensation
			if compErr := o.compensate(ctx, i); compErr != nil {
				return fmt.Errorf("saga failed and compensation failed: %w, original error: %v", compErr, err)
			}

			return fmt.Errorf("saga failed at step %d: %w", i, err)
		}

		result.Success = true
		o.execution.mu.Lock()
		o.execution.StepResults = append(o.execution.StepResults, result)
		o.execution.mu.Unlock()

		if err := o.repository.Update(ctx, o.execution); err != nil {
			return fmt.Errorf("failed to update saga execution: %w", err)
		}
	}

	now := time.Now()
	o.execution.mu.Lock()
	o.execution.Status = SagaStatusCompleted
	o.execution.CompletedAt = &now
	o.execution.mu.Unlock()

	if err := o.repository.Update(ctx, o.execution); err != nil {
		return fmt.Errorf("failed to update saga execution: %w", err)
	}

	return nil
}

// compensate executes compensating transactions for completed steps
func (o *SagaOrchestrator) compensate(ctx context.Context, failedStepIndex int) error {
	o.execution.mu.Lock()
	o.execution.Status = SagaStatusCompensating
	o.execution.mu.Unlock()

	if err := o.repository.Update(ctx, o.execution); err != nil {
		return fmt.Errorf("failed to update saga status: %w", err)
	}

	// Compensate in reverse order
	for i := failedStepIndex - 1; i >= 0; i-- {
		step := o.definition.Steps[i]
		if err := step.Compensate(ctx); err != nil {
			return fmt.Errorf("compensation failed at step %d: %w", i, err)
		}
	}

	now := time.Now()
	o.execution.mu.Lock()
	o.execution.Status = SagaStatusCompensated
	o.execution.CompensatedAt = &now
	o.execution.mu.Unlock()

	if err := o.repository.Update(ctx, o.execution); err != nil {
		return fmt.Errorf("failed to update saga execution: %w", err)
	}

	return nil
}

// GetStatus returns the current saga execution status
func (o *SagaOrchestrator) GetStatus() *SagaExecution {
	o.execution.mu.RLock()
	defer o.execution.mu.RUnlock()

	// Return a copy to prevent race conditions
	execCopy := *o.execution
	return &execCopy
}

// SagaRepository defines persistence operations for saga executions
type SagaRepository interface {
	Save(ctx context.Context, execution *SagaExecution) error
	FindByID(ctx context.Context, id string) (*SagaExecution, error)
	Update(ctx context.Context, execution *SagaExecution) error
}

// InMemorySagaRepository is an in-memory implementation of SagaRepository
type InMemorySagaRepository struct {
	executions map[string]*SagaExecution
	mu         sync.RWMutex
}

// NewInMemorySagaRepository creates a new in-memory saga repository
func NewInMemorySagaRepository() *InMemorySagaRepository {
	return &InMemorySagaRepository{
		executions: make(map[string]*SagaExecution),
	}
}

// Save stores a new saga execution
func (r *InMemorySagaRepository) Save(ctx context.Context, execution *SagaExecution) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.executions[execution.ID]; exists {
		return errors.New("saga execution already exists")
	}

	r.executions[execution.ID] = execution
	return nil
}

// FindByID retrieves a saga execution by ID
func (r *InMemorySagaRepository) FindByID(ctx context.Context, id string) (*SagaExecution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, exists := r.executions[id]
	if !exists {
		return nil, errors.New("saga execution not found")
	}

	return execution, nil
}

// Update updates an existing saga execution
func (r *InMemorySagaRepository) Update(ctx context.Context, execution *SagaExecution) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.executions[execution.ID]; !exists {
		return errors.New("saga execution not found")
	}

	r.executions[execution.ID] = execution
	return nil
}

// Example saga steps for an order processing saga

// OrderSagaStep creates an order
type OrderSagaStep struct {
	OrderID string
}

// Execute creates the order
func (s *OrderSagaStep) Execute(ctx context.Context) error {
	// Simulate order creation
	s.OrderID = uuid.New().String()
	fmt.Printf("Order created: %s\n", s.OrderID)
	return nil
}

// Compensate cancels the order
func (s *OrderSagaStep) Compensate(ctx context.Context) error {
	// Simulate order cancellation
	fmt.Printf("Order cancelled: %s\n", s.OrderID)
	return nil
}

// PaymentSagaStep processes payment
type PaymentSagaStep struct {
	PaymentID string
	Amount    float64
}

// Execute processes the payment
func (s *PaymentSagaStep) Execute(ctx context.Context) error {
	// Simulate payment processing
	s.PaymentID = uuid.New().String()
	fmt.Printf("Payment processed: %s, amount: %.2f\n", s.PaymentID, s.Amount)
	return nil
}

// Compensate refunds the payment
func (s *PaymentSagaStep) Compensate(ctx context.Context) error {
	// Simulate payment refund
	fmt.Printf("Payment refunded: %s\n", s.PaymentID)
	return nil
}

// InventorySagaStep reserves inventory
type InventorySagaStep struct {
	ReservationID string
	ProductID     string
	Quantity      int
}

// Execute reserves inventory
func (s *InventorySagaStep) Execute(ctx context.Context) error {
	// Simulate inventory reservation
	s.ReservationID = uuid.New().String()
	fmt.Printf("Inventory reserved: %s, product: %s, quantity: %d\n", s.ReservationID, s.ProductID, s.Quantity)
	return nil
}

// Compensate releases inventory reservation
func (s *InventorySagaStep) Compensate(ctx context.Context) error {
	// Simulate inventory release
	fmt.Printf("Inventory released: %s\n", s.ReservationID)
	return nil
}
