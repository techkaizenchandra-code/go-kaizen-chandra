// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Interpreter pattern
// using an expression evaluator example with arithmetic operations and variables.
//
// The Interpreter pattern defines a representation for a grammar along with an
// interpreter that uses the representation to interpret sentences in the language.
// This implementation demonstrates:
// - Context management for variable resolution
// - Terminal and non-terminal expressions
// - Composite structure for complex expressions
// - Error handling and validation
package behavioral

import (
	"fmt"
	"sync"
)

// Context defines the interface for managing variables and their values
// during expression interpretation. It provides methods to get and set
// variable values in a thread-safe manner.
type Context interface {
	// SetVariable assigns a value to a variable name
	SetVariable(name string, value float64) error
	// GetVariable retrieves the value of a variable by name
	GetVariable(name string) (float64, error)
	// HasVariable checks if a variable exists in the context
	HasVariable(name string) bool
	// Clear removes all variables from the context
	Clear()
}

// DefaultContext is a thread-safe implementation of the Context interface.
// It uses a map to store variable-value pairs and a mutex for synchronization.
type DefaultContext struct {
	variables map[string]float64
	mu        sync.RWMutex
}

// NewContext creates a new DefaultContext instance with an empty variable map.
func NewContext() Context {
	return &DefaultContext{
		variables: make(map[string]float64),
	}
}

// SetVariable assigns a value to a variable name in the context.
// Returns an error if the variable name is empty.
func (c *DefaultContext) SetVariable(name string, value float64) error {
	if name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.variables[name] = value
	return nil
}

// GetVariable retrieves the value of a variable from the context.
// Returns an error if the variable doesn't exist.
func (c *DefaultContext) GetVariable(name string) (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.variables[name]
	if !exists {
		return 0, fmt.Errorf("variable '%s' not found in context", name)
	}

	return value, nil
}

// HasVariable checks if a variable exists in the context.
func (c *DefaultContext) HasVariable(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.variables[name]
	return exists
}

// Clear removes all variables from the context.
func (c *DefaultContext) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.variables = make(map[string]float64)
}

// Expression defines the interface for all expressions in the grammar.
// Each expression can interpret itself given a context.
type Expression interface {
	// Interpret evaluates the expression using the provided context
	// and returns the result or an error if interpretation fails
	Interpret(ctx Context) (float64, error)
	// String returns a string representation of the expression
	String() string
}

// VariableExpression represents a terminal expression that resolves
// a variable's value from the context.
type VariableExpression struct {
	name string
}

// NewVariableExpression creates a new VariableExpression with validation.
// Returns an error if the variable name is empty.
func NewVariableExpression(name string) (*VariableExpression, error) {
	if name == "" {
		return nil, fmt.Errorf("variable name cannot be empty")
	}

	return &VariableExpression{
		name: name,
	}, nil
}

// Interpret retrieves the variable's value from the context.
func (v *VariableExpression) Interpret(ctx Context) (float64, error) {
	return ctx.GetVariable(v.name)
}

// String returns the variable name.
func (v *VariableExpression) String() string {
	return v.name
}

// ConstantExpression represents a terminal expression that holds
// a constant numeric value.
type ConstantExpression struct {
	value float64
}

// NewConstantExpression creates a new ConstantExpression.
func NewConstantExpression(value float64) *ConstantExpression {
	return &ConstantExpression{
		value: value,
	}
}

// Interpret returns the constant value without using the context.
func (c *ConstantExpression) Interpret(ctx Context) (float64, error) {
	return c.value, nil
}

// String returns the string representation of the constant value.
func (c *ConstantExpression) String() string {
	return fmt.Sprintf("%.2f", c.value)
}

// AddExpression represents a non-terminal expression that performs
// addition of two sub-expressions.
type AddExpression struct {
	left  Expression
	right Expression
}

// NewAddExpression creates a new AddExpression with validation.
// Returns an error if either operand is nil.
func NewAddExpression(left, right Expression) (*AddExpression, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both operands must be non-nil")
	}

	return &AddExpression{
		left:  left,
		right: right,
	}, nil
}

// Interpret evaluates both sub-expressions and returns their sum.
func (a *AddExpression) Interpret(ctx Context) (float64, error) {
	leftVal, err := a.left.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating left operand: %w", err)
	}

	rightVal, err := a.right.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating right operand: %w", err)
	}

	return leftVal + rightVal, nil
}

// String returns a string representation of the addition expression.
func (a *AddExpression) String() string {
	return fmt.Sprintf("(%s + %s)", a.left.String(), a.right.String())
}

// SubtractExpression represents a non-terminal expression that performs
// subtraction of two sub-expressions.
type SubtractExpression struct {
	left  Expression
	right Expression
}

// NewSubtractExpression creates a new SubtractExpression with validation.
// Returns an error if either operand is nil.
func NewSubtractExpression(left, right Expression) (*SubtractExpression, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both operands must be non-nil")
	}

	return &SubtractExpression{
		left:  left,
		right: right,
	}, nil
}

// Interpret evaluates both sub-expressions and returns their difference.
func (s *SubtractExpression) Interpret(ctx Context) (float64, error) {
	leftVal, err := s.left.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating left operand: %w", err)
	}

	rightVal, err := s.right.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating right operand: %w", err)
	}

	return leftVal - rightVal, nil
}

// String returns a string representation of the subtraction expression.
func (s *SubtractExpression) String() string {
	return fmt.Sprintf("(%s - %s)", s.left.String(), s.right.String())
}

// MultiplyExpression represents a non-terminal expression that performs
// multiplication of two sub-expressions.
type MultiplyExpression struct {
	left  Expression
	right Expression
}

// NewMultiplyExpression creates a new MultiplyExpression with validation.
// Returns an error if either operand is nil.
func NewMultiplyExpression(left, right Expression) (*MultiplyExpression, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both operands must be non-nil")
	}

	return &MultiplyExpression{
		left:  left,
		right: right,
	}, nil
}

// Interpret evaluates both sub-expressions and returns their product.
func (m *MultiplyExpression) Interpret(ctx Context) (float64, error) {
	leftVal, err := m.left.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating left operand: %w", err)
	}

	rightVal, err := m.right.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating right operand: %w", err)
	}

	return leftVal * rightVal, nil
}

// String returns a string representation of the multiplication expression.
func (m *MultiplyExpression) String() string {
	return fmt.Sprintf("(%s * %s)", m.left.String(), m.right.String())
}

// DivideExpression represents a non-terminal expression that performs
// division of two sub-expressions with zero-division protection.
type DivideExpression struct {
	left  Expression
	right Expression
}

// NewDivideExpression creates a new DivideExpression with validation.
// Returns an error if either operand is nil.
func NewDivideExpression(left, right Expression) (*DivideExpression, error) {
	if left == nil || right == nil {
		return nil, fmt.Errorf("both operands must be non-nil")
	}

	return &DivideExpression{
		left:  left,
		right: right,
	}, nil
}

// Interpret evaluates both sub-expressions and returns their quotient.
// Returns an error if the right operand evaluates to zero.
func (d *DivideExpression) Interpret(ctx Context) (float64, error) {
	leftVal, err := d.left.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating left operand: %w", err)
	}

	rightVal, err := d.right.Interpret(ctx)
	if err != nil {
		return 0, fmt.Errorf("error evaluating right operand: %w", err)
	}

	if rightVal == 0 {
		return 0, fmt.Errorf("division by zero")
	}

	return leftVal / rightVal, nil
}

// String returns a string representation of the division expression.
func (d *DivideExpression) String() string {
	return fmt.Sprintf("(%s / %s)", d.left.String(), d.right.String())
}
