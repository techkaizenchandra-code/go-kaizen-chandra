// Package creational implements the Prototype design pattern for production-grade applications.
// The Prototype pattern allows objects to be cloned without coupling to their specific classes.
// This implementation provides thread-safe cloning with deep copy support and a registry system
// for managing prototype instances.
package creational

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Prototype defines the interface for cloneable objects.
// All prototypes must implement this interface to support cloning.
type Prototype interface {
	// Clone creates a shallow copy of the prototype
	Clone() (Prototype, error)
	// GetID returns the unique identifier of the prototype
	GetID() string
}

// Cloneable extends Prototype with deep copy capabilities.
// This is useful for prototypes with complex nested structures.
type Cloneable interface {
	Prototype
	// DeepCopy creates a deep copy of the prototype including all nested objects
	DeepCopy() (Cloneable, error)
}

// ConcretePrototypeA represents a concrete implementation of a prototype
// with metadata and configuration that can be cloned.
type ConcretePrototypeA struct {
	mu          sync.RWMutex
	id          string
	name        string
	metadata    map[string]interface{}
	createdAt   time.Time
	cloneCount  int
	initialized bool
}

// NewConcretePrototypeA creates a new instance of ConcretePrototypeA
func NewConcretePrototypeA(id, name string) *ConcretePrototypeA {
	return &ConcretePrototypeA{
		id:          id,
		name:        name,
		metadata:    make(map[string]interface{}),
		createdAt:   time.Now(),
		initialized: true,
	}
}

// Clone creates a shallow copy of ConcretePrototypeA
func (p *ConcretePrototypeA) Clone() (Prototype, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("cannot clone uninitialized prototype")
	}

	clone := &ConcretePrototypeA{
		id:          fmt.Sprintf("%s-clone-%d", p.id, p.cloneCount+1),
		name:        p.name,
		metadata:    p.metadata, // Shallow copy - shares the same map
		createdAt:   time.Now(),
		cloneCount:  0,
		initialized: true,
	}

	p.cloneCount++
	return clone, nil
}

// DeepCopy creates a deep copy of ConcretePrototypeA including all nested data
func (p *ConcretePrototypeA) DeepCopy() (Cloneable, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("cannot deep copy uninitialized prototype")
	}

	// Deep copy metadata
	metadataCopy := make(map[string]interface{})
	for k, v := range p.metadata {
		metadataCopy[k] = v
	}

	clone := &ConcretePrototypeA{
		id:          fmt.Sprintf("%s-deepcopy-%d", p.id, p.cloneCount+1),
		name:        p.name,
		metadata:    metadataCopy, // Deep copy - new map with copied values
		createdAt:   time.Now(),
		cloneCount:  0,
		initialized: true,
	}

	p.cloneCount++
	return clone, nil
}

// GetID returns the unique identifier of the prototype
func (p *ConcretePrototypeA) GetID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.id
}

// GetMetadata returns a copy of the metadata map
func (p *ConcretePrototypeA) GetMetadata() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metadataCopy := make(map[string]interface{})
	for k, v := range p.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// SetMetadata sets a metadata value
func (p *ConcretePrototypeA) SetMetadata(key string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if key == "" {
		return errors.New("metadata key cannot be empty")
	}
	p.metadata[key] = value
	return nil
}

// ConcretePrototypeB represents another concrete implementation with different behavior
type ConcretePrototypeB struct {
	mu        sync.RWMutex
	id        string
	config    []string
	version   int
	timestamp time.Time
}

// NewConcretePrototypeB creates a new instance of ConcretePrototypeB
func NewConcretePrototypeB(id string, config []string) *ConcretePrototypeB {
	return &ConcretePrototypeB{
		id:        id,
		config:    config,
		version:   1,
		timestamp: time.Now(),
	}
}

// Clone creates a shallow copy of ConcretePrototypeB
func (p *ConcretePrototypeB) Clone() (Prototype, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	clone := &ConcretePrototypeB{
		id:        fmt.Sprintf("%s-clone", p.id),
		config:    p.config, // Shallow copy - shares the same slice
		version:   p.version + 1,
		timestamp: time.Now(),
	}

	return clone, nil
}

// DeepCopy creates a deep copy of ConcretePrototypeB
func (p *ConcretePrototypeB) DeepCopy() (Cloneable, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Deep copy config slice
	configCopy := make([]string, len(p.config))
	copy(configCopy, p.config)

	clone := &ConcretePrototypeB{
		id:        fmt.Sprintf("%s-deepcopy", p.id),
		config:    configCopy, // Deep copy - new slice with copied values
		version:   p.version + 1,
		timestamp: time.Now(),
	}

	return clone, nil
}

// GetID returns the unique identifier of the prototype
func (p *ConcretePrototypeB) GetID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.id
}

// GetConfig returns a copy of the configuration
func (p *ConcretePrototypeB) GetConfig() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	configCopy := make([]string, len(p.config))
	copy(configCopy, p.config)
	return configCopy
}

// UpdateConfig updates the configuration
func (p *ConcretePrototypeB) UpdateConfig(config []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
	p.version++
}

// PrototypeRegistry defines the interface for managing prototype instances
type PrototypeRegistry interface {
	// RegisterPrototype registers a prototype with a given name
	RegisterPrototype(name string, prototype Prototype) error
	// GetPrototype retrieves and clones a registered prototype
	GetPrototype(name string) (Prototype, error)
	// UnregisterPrototype removes a prototype from the registry
	UnregisterPrototype(name string) error
	// ListPrototypes returns all registered prototype names
	ListPrototypes() []string
}

// ConcretePrototypeRegistry is a thread-safe implementation of PrototypeRegistry
type ConcretePrototypeRegistry struct {
	mu         sync.RWMutex
	prototypes map[string]Prototype
}

// NewPrototypeRegistry creates a new prototype registry with default prototypes
func NewPrototypeRegistry() *ConcretePrototypeRegistry {
	registry := &ConcretePrototypeRegistry{
		prototypes: make(map[string]Prototype),
	}

	// Register default prototypes
	defaultPrototypeA := NewConcretePrototypeA("default-a", "DefaultPrototypeA")
	defaultPrototypeA.SetMetadata("type", "default")
	registry.prototypes["default-a"] = defaultPrototypeA

	defaultPrototypeB := NewConcretePrototypeB("default-b", []string{"config1", "config2"})
	registry.prototypes["default-b"] = defaultPrototypeB

	return registry
}

// RegisterPrototype registers a new prototype with validation
func (r *ConcretePrototypeRegistry) RegisterPrototype(name string, prototype Prototype) error {
	if name == "" {
		return errors.New("prototype name cannot be empty")
	}
	if prototype == nil {
		return errors.New("prototype cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.prototypes[name]; exists {
		return fmt.Errorf("prototype with name %s already exists", name)
	}

	r.prototypes[name] = prototype
	return nil
}

// GetPrototype retrieves and clones a registered prototype
func (r *ConcretePrototypeRegistry) GetPrototype(name string) (Prototype, error) {
	r.mu.RLock()
	prototype, exists := r.prototypes[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("prototype %s not found", name)
	}

	// Clone the prototype to return a new instance
	clone, err := prototype.Clone()
	if err != nil {
		return nil, fmt.Errorf("failed to clone prototype %s: %w", name, err)
	}

	return clone, nil
}

// UnregisterPrototype removes a prototype from the registry
func (r *ConcretePrototypeRegistry) UnregisterPrototype(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.prototypes[name]; !exists {
		return fmt.Errorf("prototype %s not found", name)
	}

	delete(r.prototypes, name)
	return nil
}

// ListPrototypes returns all registered prototype names
func (r *ConcretePrototypeRegistry) ListPrototypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.prototypes))
	for name := range r.prototypes {
		names = append(names, name)
	}
	return names
}

// TestPrototype demonstrates the usage of the Prototype pattern in production
func TestPrototype() error {
	fmt.Println("=== Prototype Pattern Demo ===\n")

	// Create prototype registry
	registry := NewPrototypeRegistry()

	// Create and register custom prototypes
	customPrototypeA := NewConcretePrototypeA("custom-a", "CustomPrototypeA")
	if err := customPrototypeA.SetMetadata("environment", "production"); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}
	if err := customPrototypeA.SetMetadata("version", "1.0.0"); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	if err := registry.RegisterPrototype("custom-a", customPrototypeA); err != nil {
		return fmt.Errorf("failed to register custom prototype: %w", err)
	}

	// Clone using shallow copy
	fmt.Println("1. Shallow Clone Demo:")
	clone1, err := registry.GetPrototype("custom-a")
	if err != nil {
		return fmt.Errorf("failed to get prototype: %w", err)
	}
	fmt.Printf("   Original ID: %s\n", customPrototypeA.GetID())
	fmt.Printf("   Clone ID: %s\n", clone1.GetID())

	// Deep copy demo
	fmt.Println("\n2. Deep Copy Demo:")
	deepClone, err := customPrototypeA.DeepCopy()
	if err != nil {
		return fmt.Errorf("failed to deep copy: %w", err)
	}
	fmt.Printf("   Original ID: %s\n", customPrototypeA.GetID())
	fmt.Printf("   Deep Clone ID: %s\n", deepClone.GetID())

	// Prototype B demo
	fmt.Println("\n3. ConcretePrototypeB Demo:")
	prototypeB := NewConcretePrototypeB("prototype-b", []string{"setting1", "setting2"})
	if err := registry.RegisterPrototype("prototype-b", prototypeB); err != nil {
		return fmt.Errorf("failed to register prototype B: %w", err)
	}

	cloneB, err := registry.GetPrototype("prototype-b")
	if err != nil {
		return fmt.Errorf("failed to get prototype B: %w", err)
	}
	fmt.Printf("   Prototype B Clone ID: %s\n", cloneB.GetID())

	// List all prototypes
	fmt.Println("\n4. Registered Prototypes:")
	for _, name := range registry.ListPrototypes() {
		fmt.Printf("   - %s\n", name)
	}

	return nil
}
