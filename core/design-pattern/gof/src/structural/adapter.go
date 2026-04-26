package structural

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// Adapter Pattern: Converts the interface of a class into another interface clients expect.
// Real-world use case: Integrating multiple third-party payment gateways with different APIs
// into a unified payment processing interface.

// Custom errors for production-grade error handling
var (
	ErrInvalidAmount      = errors.New("invalid payment amount")
	ErrInvalidCurrency    = errors.New("invalid currency code")
	ErrPaymentFailed      = errors.New("payment processing failed")
	ErrAdapteeUnavailable = errors.New("payment service unavailable")
	ErrContextCancelled   = errors.New("payment context cancelled")
)

// PaymentProcessor is the Target interface that our application expects.
// This is the standard interface all payment processors must implement.
type PaymentProcessor interface {
	ProcessPayment(ctx context.Context, amount float64, currency string, customerID string) (transactionID string, err error)
	RefundPayment(ctx context.Context, transactionID string) error
	GetPaymentStatus(ctx context.Context, transactionID string) (string, error)
}

// StripeAPIClient represents the Adaptee - a third-party payment service (Stripe)
// with its own interface that is incompatible with our PaymentProcessor interface.
type StripeAPIClient struct {
	apiKey string
	logger *log.Logger
}

// NewStripeAPIClient creates a new Stripe API client
func NewStripeAPIClient(apiKey string, logger *log.Logger) *StripeAPIClient {
	if logger == nil {
		logger = log.Default()
	}
	return &StripeAPIClient{
		apiKey: apiKey,
		logger: logger,
	}
}

// CreateCharge is Stripe's method for processing payments (incompatible with our interface)
func (s *StripeAPIClient) CreateCharge(amountInCents int64, curr string, custID string, description string) (string, error) {
	s.logger.Printf("Stripe: Creating charge for customer %s, amount: %d %s", custID, amountInCents, curr)

	// Simulate Stripe API call
	if amountInCents <= 0 {
		return "", fmt.Errorf("stripe error: %w", ErrInvalidAmount)
	}

	// Simulate successful charge
	chargeID := fmt.Sprintf("ch_stripe_%d", time.Now().UnixNano())
	s.logger.Printf("Stripe: Charge created successfully: %s", chargeID)
	return chargeID, nil
}

// CancelCharge is Stripe's method for refunds
func (s *StripeAPIClient) CancelCharge(chargeID string) error {
	s.logger.Printf("Stripe: Cancelling charge %s", chargeID)
	// Simulate Stripe API refund
	if chargeID == "" {
		return errors.New("stripe error: invalid charge ID")
	}
	s.logger.Printf("Stripe: Charge cancelled successfully: %s", chargeID)
	return nil
}

// RetrieveCharge gets charge details from Stripe
func (s *StripeAPIClient) RetrieveCharge(chargeID string) (map[string]interface{}, error) {
	s.logger.Printf("Stripe: Retrieving charge %s", chargeID)
	// Simulate Stripe API response
	return map[string]interface{}{
		"id":     chargeID,
		"status": "succeeded",
		"amount": 10000,
	}, nil
}

// StripePaymentAdapter is the Adapter that makes StripeAPIClient compatible with PaymentProcessor
type StripePaymentAdapter struct {
	stripeClient *StripeAPIClient
	logger       *log.Logger
}

// NewStripePaymentAdapter creates a new adapter for Stripe payments
func NewStripePaymentAdapter(apiKey string, logger *log.Logger) *StripePaymentAdapter {
	if logger == nil {
		logger = log.Default()
	}
	return &StripePaymentAdapter{
		stripeClient: NewStripeAPIClient(apiKey, logger),
		logger:       logger,
	}
}

// ProcessPayment adapts our interface to Stripe's CreateCharge method
func (a *StripePaymentAdapter) ProcessPayment(ctx context.Context, amount float64, currency string, customerID string) (string, error) {
	// Validation
	if amount <= 0 {
		a.logger.Printf("Payment validation failed: invalid amount %.2f", amount)
		return "", ErrInvalidAmount
	}

	if currency == "" {
		a.logger.Printf("Payment validation failed: empty currency")
		return "", ErrInvalidCurrency
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		a.logger.Printf("Payment cancelled via context for customer %s", customerID)
		return "", ErrContextCancelled
	default:
	}

	// Convert amount to cents (Stripe expects amounts in smallest currency unit)
	amountInCents := int64(amount * 100)

	a.logger.Printf("Adapter: Processing payment for customer %s, amount: %.2f %s", customerID, amount, currency)

	// Adapt the call to Stripe's interface
	chargeID, err := a.stripeClient.CreateCharge(amountInCents, currency, customerID, "Payment via adapter")
	if err != nil {
		a.logger.Printf("Adapter: Payment failed for customer %s: %v", customerID, err)
		return "", fmt.Errorf("%w: %v", ErrPaymentFailed, err)
	}

	a.logger.Printf("Adapter: Payment processed successfully for customer %s, transaction ID: %s", customerID, chargeID)
	return chargeID, nil
}

// RefundPayment adapts our interface to Stripe's CancelCharge method
func (a *StripePaymentAdapter) RefundPayment(ctx context.Context, transactionID string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		a.logger.Printf("Refund cancelled via context for transaction %s", transactionID)
		return ErrContextCancelled
	default:
	}

	a.logger.Printf("Adapter: Processing refund for transaction %s", transactionID)

	err := a.stripeClient.CancelCharge(transactionID)
	if err != nil {
		a.logger.Printf("Adapter: Refund failed for transaction %s: %v", transactionID, err)
		return fmt.Errorf("%w: %v", ErrPaymentFailed, err)
	}

	a.logger.Printf("Adapter: Refund processed successfully for transaction %s", transactionID)
	return nil
}

// GetPaymentStatus adapts our interface to Stripe's RetrieveCharge method
func (a *StripePaymentAdapter) GetPaymentStatus(ctx context.Context, transactionID string) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		a.logger.Printf("Status check cancelled via context for transaction %s", transactionID)
		return "", ErrContextCancelled
	default:
	}

	a.logger.Printf("Adapter: Retrieving payment status for transaction %s", transactionID)

	charge, err := a.stripeClient.RetrieveCharge(transactionID)
	if err != nil {
		a.logger.Printf("Adapter: Failed to retrieve status for transaction %s: %v", transactionID, err)
		return "", fmt.Errorf("%w: %v", ErrPaymentFailed, err)
	}

	status, ok := charge["status"].(string)
	if !ok {
		status = "unknown"
	}

	a.logger.Printf("Adapter: Payment status for transaction %s: %s", transactionID, status)
	return status, nil
}

// PaymentService demonstrates how the application uses the adapter pattern
type PaymentService struct {
	processor PaymentProcessor
	logger    *log.Logger
}

// NewPaymentService creates a new payment service with the given processor
func NewPaymentService(processor PaymentProcessor, logger *log.Logger) *PaymentService {
	if logger == nil {
		logger = log.Default()
	}
	return &PaymentService{
		processor: processor,
		logger:    logger,
	}
}

// Charge processes a payment using the configured processor
func (s *PaymentService) Charge(ctx context.Context, amount float64, currency string, customerID string) (string, error) {
	s.logger.Printf("PaymentService: Initiating charge for customer %s", customerID)

	// Add timeout to context if not already present
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	transactionID, err := s.processor.ProcessPayment(ctx, amount, currency, customerID)
	if err != nil {
		s.logger.Printf("PaymentService: Charge failed for customer %s: %v", customerID, err)
		return "", err
	}

	s.logger.Printf("PaymentService: Charge successful for customer %s, transaction: %s", customerID, transactionID)
	return transactionID, nil
}

// Example usage demonstrating the adapter pattern in action
func AdapterPatternExample() {
	logger := log.Default()

	// Create a Stripe payment adapter
	stripeAdapter := NewStripePaymentAdapter("sk_test_stripe_key", logger)

	// Create payment service that works with any PaymentProcessor
	paymentService := NewPaymentService(stripeAdapter, logger)

	// Process a payment - the service doesn't know it's using Stripe
	ctx := context.Background()
	transactionID, err := paymentService.Charge(ctx, 99.99, "USD", "customer_12345")
	if err != nil {
		logger.Printf("Error processing payment: %v", err)
		return
	}

	logger.Printf("Payment processed successfully. Transaction ID: %s", transactionID)

	// Get payment status
	status, err := stripeAdapter.GetPaymentStatus(ctx, transactionID)
	if err != nil {
		logger.Printf("Error getting payment status: %v", err)
		return
	}
	logger.Printf("Payment status: %s", status)

	// Process refund
	err = stripeAdapter.RefundPayment(ctx, transactionID)
	if err != nil {
		logger.Printf("Error processing refund: %v", err)
		return
	}
	logger.Printf("Refund processed successfully")
}
