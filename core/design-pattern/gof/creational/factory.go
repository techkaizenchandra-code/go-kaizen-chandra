// Package creational implements creational design patterns including
// the Factory pattern for production-grade applications.
// This implementation provides thread-safe product creation with
// registration capabilities and comprehensive error handling.
package creational

import (
	"errors"
	"fmt"
	"sync"
)

// Product defines the interface that all concrete products must implement.
// This ensures consistent behavior across different product types.
type Product interface {
	// GetName returns the name/identifier of the product
	GetName() string
	// GetType returns the type of the product
	GetType() ProductType
	// Initialize performs any necessary initialization
	Initialize() error
	// Execute performs the main operation of the product
	Execute() error
}

// ProductType represents the type of product to be created.
// Using a custom type provides type safety and prevents invalid values.
type ProductType string

const (
	// ProductTypeA represents the first product type
	ProductTypeA ProductType = "ProductA"
	// ProductTypeB represents the second product type
	ProductTypeB ProductType = "ProductB"
)

// ConcreteProductA is a concrete implementation of Product interface
type ConcreteProductA struct {
	name        string
	initialized bool
}

// GetName returns the name of ConcreteProductA
func (p *ConcreteProductA) GetName() string {
	return p.name
}

// GetType returns the type of ConcreteProductA
func (p *ConcreteProductA) GetType() ProductType {
	return ProductTypeA
}

// Initialize performs initialization for ConcreteProductA
func (p *ConcreteProductA) Initialize() error {
	if p.initialized {
		return errors.New("product already initialized")
	}
	p.initialized = true
	return nil
}

// Execute performs the main operation for ConcreteProductA
func (p *ConcreteProductA) Execute() error {
	if !p.initialized {
		return errors.New("product not initialized")
	}
	fmt.Printf("Executing ConcreteProductA: %s\n", p.name)
	return nil
}

// ConcreteProductB is another concrete implementation of Product interface
type ConcreteProductB struct {
	name        string
	initialized bool
}

// GetName returns the name of ConcreteProductB
func (p *ConcreteProductB) GetName() string {
	return p.name
}

// GetType returns the type of ConcreteProductB
func (p *ConcreteProductB) GetType() ProductType {
	return ProductTypeB
}

// Initialize performs initialization for ConcreteProductB
func (p *ConcreteProductB) Initialize() error {
	if p.initialized {
		return errors.New("product already initialized")
	}
	p.initialized = true
	return nil
}

// Execute performs the main operation for ConcreteProductB
func (p *ConcreteProductB) Execute() error {
	if !p.initialized {
		return errors.New("product not initialized")
	}
	fmt.Printf("Executing ConcreteProductB: %s\n", p.name)
	return nil
}

// ProductCreator is a function type that creates a Product instance
type ProductCreator func(name string) (Product, error)

// Factory defines the interface for creating products.
// This allows for multiple factory implementations.
type Factory interface {
	// CreateProduct creates a product of the specified type
	CreateProduct(productType ProductType, name string) (Product, error)
	// RegisterProductType allows dynamic registration of new product types
	RegisterProductType(productType ProductType, creator ProductCreator) error
	// GetSupportedTypes returns all registered product types
	GetSupportedTypes() []ProductType
}

// ConcreteFactory is a thread-safe implementation of the Factory interface
type ConcreteFactory struct {
	mu       sync.RWMutex
	registry map[ProductType]ProductCreator
}

// NewConcreteFactory creates a new ConcreteFactory with default product types registered
func NewConcreteFactory() *ConcreteFactory {
	factory := &ConcreteFactory{
		registry: make(map[ProductType]ProductCreator),
	}

	// Register default product types
	factory.registry[ProductTypeA] = func(name string) (Product, error) {
		if name == "" {
			return nil, errors.New("product name cannot be empty")
		}
		return &ConcreteProductA{name: name}, nil
	}

	factory.registry[ProductTypeB] = func(name string) (Product, error) {
		if name == "" {
			return nil, errors.New("product name cannot be empty")
		}
		return &ConcreteProductB{name: name}, nil
	}

	return factory
}

// CreateProduct creates a product of the specified type with thread-safety and validation
func (f *ConcreteFactory) CreateProduct(productType ProductType, name string) (Product, error) {
	f.mu.RLock()
	creator, exists := f.registry[productType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported product type: %s", productType)
	}

	if name == "" {
		return nil, errors.New("product name cannot be empty")
	}

	product, err := creator(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Initialize the product
	if err := product.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize product: %w", err)
	}

	return product, nil
}

// RegisterProductType allows dynamic registration of new product types at runtime
func (f *ConcreteFactory) RegisterProductType(productType ProductType, creator ProductCreator) error {
	if productType == "" {
		return errors.New("product type cannot be empty")
	}
	if creator == nil {
		return errors.New("product creator cannot be nil")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.registry[productType]; exists {
		return fmt.Errorf("product type %s is already registered", productType)
	}

	f.registry[productType] = creator
	return nil
}

// GetSupportedTypes returns a list of all registered product types
func (f *ConcreteFactory) GetSupportedTypes() []ProductType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]ProductType, 0, len(f.registry))
	for productType := range f.registry {
		types = append(types, productType)
	}
	return types
}

// TestFactory ExampleFactoryUsage demonstrates how to use the factory pattern in production
func TestFactory() error {
	// Create factory instance
	factory := NewConcreteFactory()

	// Create products using the factory
	productA, err := factory.CreateProduct(ProductTypeA, "MyProductA")
	if err != nil {
		return fmt.Errorf("failed to create ProductA: %w", err)
	}

	productB, err := factory.CreateProduct(ProductTypeB, "MyProductB")
	if err != nil {
		return fmt.Errorf("failed to create ProductB: %w", err)
	}

	// Execute products
	if err := productA.Execute(); err != nil {
		return fmt.Errorf("failed to execute ProductA: %w", err)
	}

	if err := productB.Execute(); err != nil {
		return fmt.Errorf("failed to execute ProductB: %w", err)
	}

	// List supported types
	fmt.Println("Supported product types:", factory.GetSupportedTypes())

	return nil
}
