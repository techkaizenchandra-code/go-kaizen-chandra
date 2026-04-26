package main

import (
	"errors"
	"fmt"
	"time"
)

// PaymentProcessor defines the contract that all payment processors must follow
// This is the base abstraction that ensures LSP compliance
type PaymentProcessor interface {
	ProcessPayment(amount float64) (*PaymentResult, error)
	RefundPayment(transactionID string, amount float64) (*RefundResult, error)
	GetProcessorName() string
	ValidateAmount(amount float64) error
}

// PaymentResult represents the result of a payment transaction
type PaymentResult struct {
	TransactionID string
	Amount        float64
	Status        string
	ProcessedAt   time.Time
	ProcessorName string
}

// RefundResult represents the result of a refund transaction
type RefundResult struct {
	RefundID      string
	TransactionID string
	Amount        float64
	Status        string
	RefundedAt    time.Time
}

// BasePaymentProcessor provides common functionality for all payment processors
// This ensures consistent behavior across all implementations
type BasePaymentProcessor struct {
	Name           string
	MinAmount      float64
	MaxAmount      float64
	TransactionFee float64
}

// ValidateAmount is a common validation method that all processors can use
func (b *BasePaymentProcessor) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if amount < b.MinAmount {
		return fmt.Errorf("amount must be at least %.2f", b.MinAmount)
	}
	if amount > b.MaxAmount {
		return fmt.Errorf("amount cannot exceed %.2f", b.MaxAmount)
	}
	return nil
}

func (b *BasePaymentProcessor) GetProcessorName() string {
	return b.Name
}

// CreditCardProcessor handles credit card payments
// Substitutable for PaymentProcessor without breaking functionality
type CreditCardProcessor struct {
	BasePaymentProcessor
	CardNetwork string
}

func NewCreditCardProcessor() *CreditCardProcessor {
	return &CreditCardProcessor{
		BasePaymentProcessor: BasePaymentProcessor{
			Name:           "CreditCard",
			MinAmount:      1.00,
			MaxAmount:      50000.00,
			TransactionFee: 2.9,
		},
		CardNetwork: "Visa/MasterCard",
	}
}

func (c *CreditCardProcessor) ProcessPayment(amount float64) (*PaymentResult, error) {
	if err := c.ValidateAmount(amount); err != nil {
		return nil, err
	}

	// Simulate credit card processing
	transactionID := fmt.Sprintf("CC-%d", time.Now().Unix())

	return &PaymentResult{
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "SUCCESS",
		ProcessedAt:   time.Now(),
		ProcessorName: c.GetProcessorName(),
	}, nil
}

func (c *CreditCardProcessor) RefundPayment(transactionID string, amount float64) (*RefundResult, error) {
	if err := c.ValidateAmount(amount); err != nil {
		return nil, err
	}

	refundID := fmt.Sprintf("REF-CC-%d", time.Now().Unix())

	return &RefundResult{
		RefundID:      refundID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "REFUNDED",
		RefundedAt:    time.Now(),
	}, nil
}

// PayPalProcessor handles PayPal payments
// Fully substitutable for PaymentProcessor
type PayPalProcessor struct {
	BasePaymentProcessor
	EmailRequired bool
}

func NewPayPalProcessor() *PayPalProcessor {
	return &PayPalProcessor{
		BasePaymentProcessor: BasePaymentProcessor{
			Name:           "PayPal",
			MinAmount:      0.50,
			MaxAmount:      100000.00,
			TransactionFee: 3.5,
		},
		EmailRequired: true,
	}
}

func (p *PayPalProcessor) ProcessPayment(amount float64) (*PaymentResult, error) {
	if err := p.ValidateAmount(amount); err != nil {
		return nil, err
	}

	// Simulate PayPal processing
	transactionID := fmt.Sprintf("PP-%d", time.Now().Unix())

	return &PaymentResult{
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "SUCCESS",
		ProcessedAt:   time.Now(),
		ProcessorName: p.GetProcessorName(),
	}, nil
}

func (p *PayPalProcessor) RefundPayment(transactionID string, amount float64) (*RefundResult, error) {
	if err := p.ValidateAmount(amount); err != nil {
		return nil, err
	}

	refundID := fmt.Sprintf("REF-PP-%d", time.Now().Unix())

	return &RefundResult{
		RefundID:      refundID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "REFUNDED",
		RefundedAt:    time.Now(),
	}, nil
}

// CryptoCurrencyProcessor handles cryptocurrency payments
// Demonstrates LSP by being fully substitutable despite different internal behavior
type CryptoCurrencyProcessor struct {
	BasePaymentProcessor
	BlockchainNetwork string
}

func NewCryptoCurrencyProcessor() *CryptoCurrencyProcessor {
	return &CryptoCurrencyProcessor{
		BasePaymentProcessor: BasePaymentProcessor{
			Name:           "Cryptocurrency",
			MinAmount:      10.00,
			MaxAmount:      1000000.00,
			TransactionFee: 1.0,
		},
		BlockchainNetwork: "Ethereum",
	}
}

func (c *CryptoCurrencyProcessor) ProcessPayment(amount float64) (*PaymentResult, error) {
	if err := c.ValidateAmount(amount); err != nil {
		return nil, err
	}

	// Simulate crypto processing with different transaction ID format
	transactionID := fmt.Sprintf("CRYPTO-0x%x", time.Now().Unix())

	return &PaymentResult{
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "SUCCESS",
		ProcessedAt:   time.Now(),
		ProcessorName: c.GetProcessorName(),
	}, nil
}

func (c *CryptoCurrencyProcessor) RefundPayment(transactionID string, amount float64) (*RefundResult, error) {
	if err := c.ValidateAmount(amount); err != nil {
		return nil, err
	}

	refundID := fmt.Sprintf("REF-CRYPTO-0x%x", time.Now().Unix())

	return &RefundResult{
		RefundID:      refundID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        "REFUNDED",
		RefundedAt:    time.Now(),
	}, nil
}

// PaymentService uses PaymentProcessor abstraction
// Can work with any PaymentProcessor implementation without knowing the specifics
type PaymentService struct {
	processor PaymentProcessor
}

func NewPaymentService(processor PaymentProcessor) *PaymentService {
	return &PaymentService{
		processor: processor,
	}
}

// ProcessTransaction demonstrates LSP - works with any PaymentProcessor
func (ps *PaymentService) ProcessTransaction(amount float64) (*PaymentResult, error) {
	fmt.Printf("\nProcessing payment of $%.2f using %s...\n", amount, ps.processor.GetProcessorName())

	result, err := ps.processor.ProcessPayment(amount)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}

	fmt.Printf("✓ Payment processed successfully - Transaction ID: %s\n", result.TransactionID)
	return result, nil
}

// ProcessRefund demonstrates LSP - works with any PaymentProcessor
func (ps *PaymentService) ProcessRefund(transactionID string, amount float64) (*RefundResult, error) {
	fmt.Printf("\nProcessing refund of $%.2f for transaction %s...\n", amount, transactionID)

	result, err := ps.processor.RefundPayment(transactionID, amount)
	if err != nil {
		return nil, fmt.Errorf("refund failed: %w", err)
	}

	fmt.Printf("✓ Refund processed successfully - Refund ID: %s\n", result.RefundID)
	return result, nil
}

// SetProcessor allows changing the payment processor at runtime
// Demonstrates that all processors are truly substitutable
func (ps *PaymentService) SetProcessor(processor PaymentProcessor) {
	ps.processor = processor
}

// TestLiskovSubstitution demonstrates the Liskov Substitution Principle
func TestLiskovSubstitution() {
	fmt.Println("=== Liskov Substitution Principle Demo ===")
	fmt.Println("All payment processors can be substituted without breaking functionality\n")

	// Create different payment processors
	processors := []PaymentProcessor{
		NewCreditCardProcessor(),
		NewPayPalProcessor(),
		NewCryptoCurrencyProcessor(),
	}

	// Create payment service
	service := NewPaymentService(processors[0])

	// Test with different amounts
	testAmounts := []float64{100.00, 500.00, 1500.00}

	// Demonstrate that all processors are substitutable
	for _, processor := range processors {
		fmt.Printf("\n--- Testing with %s Processor ---\n", processor.GetProcessorName())
		service.SetProcessor(processor)

		for _, amount := range testAmounts {
			// Process payment
			result, err := service.ProcessTransaction(amount)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
				continue
			}

			// Process refund
			_, err = service.ProcessRefund(result.TransactionID, amount)
			if err != nil {
				fmt.Printf("✗ Error: %v\n", err)
			}
		}
	}

	// Demonstrate validation behavior is consistent across all processors
	fmt.Println("\n--- Testing Validation (LSP Compliance) ---")
	for _, processor := range processors {
		fmt.Printf("\nTesting %s with invalid amount:\n", processor.GetProcessorName())
		service.SetProcessor(processor)
		_, err := service.ProcessTransaction(-50.00)
		if err != nil {
			fmt.Printf("✓ Validation works correctly: %v\n", err)
		}
	}

	fmt.Println("\n=== LSP Benefits Demonstrated ===")
	fmt.Println("✓ All payment processors are substitutable without breaking the client code")
	fmt.Println("✓ Each processor maintains the contract defined by PaymentProcessor interface")
	fmt.Println("✓ Pre-conditions are not strengthened (all accept valid amounts)")
	fmt.Println("✓ Post-conditions are not weakened (all return valid results or errors)")
	fmt.Println("✓ Invariants are preserved (validation rules are consistent)")
	fmt.Println("✓ Client code (PaymentService) works correctly with any processor")
}
