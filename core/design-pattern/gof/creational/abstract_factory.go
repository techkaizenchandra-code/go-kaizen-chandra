// Package creational implements creational design patterns including
// the Abstract Factory pattern for production-grade applications.
// This implementation provides thread-safe creation of families of related
// objects with comprehensive error handling and platform abstraction.
package creational

import (
	"errors"
	"fmt"
	"sync"
)

// Button defines the interface for button products across different platforms
type Button interface {
	// Render displays the button on the screen
	Render() error
	// OnClick handles the click event
	OnClick() error
	// GetPlatform returns the platform type
	GetPlatform() PlatformType
}

// Checkbox defines the interface for checkbox products across different platforms
type Checkbox interface {
	// Render displays the checkbox on the screen
	Render() error
	// OnCheck handles the check/uncheck event
	OnCheck(checked bool) error
	// GetPlatform returns the platform type
	GetPlatform() PlatformType
}

// PlatformType represents the type of platform/OS
type PlatformType string

const (
	// PlatformWindows represents Windows platform
	PlatformWindows PlatformType = "Windows"
	// PlatformMac represents Mac platform
	PlatformMac PlatformType = "Mac"
)

// WindowsButton is a concrete implementation of Button for Windows platform
type WindowsButton struct {
	label       string
	initialized bool
}

// NewWindowsButton creates a new WindowsButton instance
func NewWindowsButton(label string) *WindowsButton {
	return &WindowsButton{
		label:       label,
		initialized: true,
	}
}

// Render displays the Windows button
func (b *WindowsButton) Render() error {
	if !b.initialized {
		return errors.New("button not initialized")
	}
	fmt.Printf("Rendering Windows button: %s\n", b.label)
	return nil
}

// OnClick handles the Windows button click event
func (b *WindowsButton) OnClick() error {
	if !b.initialized {
		return errors.New("button not initialized")
	}
	fmt.Printf("Windows button '%s' clicked\n", b.label)
	return nil
}

// GetPlatform returns the Windows platform type
func (b *WindowsButton) GetPlatform() PlatformType {
	return PlatformWindows
}

// WindowsCheckbox is a concrete implementation of Checkbox for Windows platform
type WindowsCheckbox struct {
	label       string
	checked     bool
	initialized bool
}

// NewWindowsCheckbox creates a new WindowsCheckbox instance
func NewWindowsCheckbox(label string) *WindowsCheckbox {
	return &WindowsCheckbox{
		label:       label,
		initialized: true,
	}
}

// Render displays the Windows checkbox
func (c *WindowsCheckbox) Render() error {
	if !c.initialized {
		return errors.New("checkbox not initialized")
	}
	fmt.Printf("Rendering Windows checkbox: %s [%v]\n", c.label, c.checked)
	return nil
}

// OnCheck handles the Windows checkbox check/uncheck event
func (c *WindowsCheckbox) OnCheck(checked bool) error {
	if !c.initialized {
		return errors.New("checkbox not initialized")
	}
	c.checked = checked
	fmt.Printf("Windows checkbox '%s' set to: %v\n", c.label, checked)
	return nil
}

// GetPlatform returns the Windows platform type
func (c *WindowsCheckbox) GetPlatform() PlatformType {
	return PlatformWindows
}

// MacButton is a concrete implementation of Button for Mac platform
type MacButton struct {
	label       string
	initialized bool
}

// NewMacButton creates a new MacButton instance
func NewMacButton(label string) *MacButton {
	return &MacButton{
		label:       label,
		initialized: true,
	}
}

// Render displays the Mac button
func (b *MacButton) Render() error {
	if !b.initialized {
		return errors.New("button not initialized")
	}
	fmt.Printf("Rendering Mac button: %s\n", b.label)
	return nil
}

// OnClick handles the Mac button click event
func (b *MacButton) OnClick() error {
	if !b.initialized {
		return errors.New("button not initialized")
	}
	fmt.Printf("Mac button '%s' clicked\n", b.label)
	return nil
}

// GetPlatform returns the Mac platform type
func (b *MacButton) GetPlatform() PlatformType {
	return PlatformMac
}

// MacCheckbox is a concrete implementation of Checkbox for Mac platform
type MacCheckbox struct {
	label       string
	checked     bool
	initialized bool
}

// NewMacCheckbox creates a new MacCheckbox instance
func NewMacCheckbox(label string) *MacCheckbox {
	return &MacCheckbox{
		label:       label,
		initialized: true,
	}
}

// Render displays the Mac checkbox
func (c *MacCheckbox) Render() error {
	if !c.initialized {
		return errors.New("checkbox not initialized")
	}
	fmt.Printf("Rendering Mac checkbox: %s [%v]\n", c.label, c.checked)
	return nil
}

// OnCheck handles the Mac checkbox check/uncheck event
func (c *MacCheckbox) OnCheck(checked bool) error {
	if !c.initialized {
		return errors.New("checkbox not initialized")
	}
	c.checked = checked
	fmt.Printf("Mac checkbox '%s' set to: %v\n", c.label, checked)
	return nil
}

// GetPlatform returns the Mac platform type
func (c *MacCheckbox) GetPlatform() PlatformType {
	return PlatformMac
}

// GUIFactory defines the abstract factory interface for creating families of related GUI products
type GUIFactory interface {
	// CreateButton creates a button for the specific platform
	CreateButton(label string) (Button, error)
	// CreateCheckbox creates a checkbox for the specific platform
	CreateCheckbox(label string) (Checkbox, error)
	// GetPlatform returns the platform type this factory creates products for
	GetPlatform() PlatformType
}

// WindowsFactory is a concrete factory for creating Windows GUI products
type WindowsFactory struct{}

// NewWindowsFactory creates a new WindowsFactory instance
func NewWindowsFactory() *WindowsFactory {
	return &WindowsFactory{}
}

// CreateButton creates a Windows button
func (f *WindowsFactory) CreateButton(label string) (Button, error) {
	if label == "" {
		return nil, errors.New("button label cannot be empty")
	}
	return NewWindowsButton(label), nil
}

// CreateCheckbox creates a Windows checkbox
func (f *WindowsFactory) CreateCheckbox(label string) (Checkbox, error) {
	if label == "" {
		return nil, errors.New("checkbox label cannot be empty")
	}
	return NewWindowsCheckbox(label), nil
}

// GetPlatform returns the Windows platform type
func (f *WindowsFactory) GetPlatform() PlatformType {
	return PlatformWindows
}

// MacFactory is a concrete factory for creating Mac GUI products
type MacFactory struct{}

// NewMacFactory creates a new MacFactory instance
func NewMacFactory() *MacFactory {
	return &MacFactory{}
}

// CreateButton creates a Mac button
func (f *MacFactory) CreateButton(label string) (Button, error) {
	if label == "" {
		return nil, errors.New("button label cannot be empty")
	}
	return NewMacButton(label), nil
}

// CreateCheckbox creates a Mac checkbox
func (f *MacFactory) CreateCheckbox(label string) (Checkbox, error) {
	if label == "" {
		return nil, errors.New("checkbox label cannot be empty")
	}
	return NewMacCheckbox(label), nil
}

// GetPlatform returns the Mac platform type
func (f *MacFactory) GetPlatform() PlatformType {
	return PlatformMac
}

// FactoryCreator is a function type that creates a GUIFactory instance
type FactoryCreator func() (GUIFactory, error)

// FactoryProvider manages thread-safe registration and retrieval of GUI factories
type FactoryProvider struct {
	mu        sync.RWMutex
	factories map[PlatformType]FactoryCreator
}

// NewFactoryProvider creates a new FactoryProvider with default factories registered
func NewFactoryProvider() *FactoryProvider {
	provider := &FactoryProvider{
		factories: make(map[PlatformType]FactoryCreator),
	}

	// Register default factories
	provider.factories[PlatformWindows] = func() (GUIFactory, error) {
		return NewWindowsFactory(), nil
	}

	provider.factories[PlatformMac] = func() (GUIFactory, error) {
		return NewMacFactory(), nil
	}

	return provider
}

// GetFactory retrieves a factory for the specified platform with thread-safety
func (p *FactoryProvider) GetFactory(platform PlatformType) (GUIFactory, error) {
	p.mu.RLock()
	creator, exists := p.factories[platform]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	factory, err := creator()
	if err != nil {
		return nil, fmt.Errorf("failed to create factory for platform %s: %w", platform, err)
	}

	return factory, nil
}

// RegisterFactory allows dynamic registration of new factory types at runtime
func (p *FactoryProvider) RegisterFactory(platform PlatformType, creator FactoryCreator) error {
	if platform == "" {
		return errors.New("platform type cannot be empty")
	}
	if creator == nil {
		return errors.New("factory creator cannot be nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.factories[platform]; exists {
		return fmt.Errorf("factory for platform %s is already registered", platform)
	}

	p.factories[platform] = creator
	return nil
}

// GetSupportedPlatforms returns a list of all registered platform types
func (p *FactoryProvider) GetSupportedPlatforms() []PlatformType {
	p.mu.RLock()
	defer p.mu.RUnlock()

	platforms := make([]PlatformType, 0, len(p.factories))
	for platform := range p.factories {
		platforms = append(platforms, platform)
	}
	return platforms
}

// Application demonstrates using the abstract factory pattern
type Application struct {
	factory  GUIFactory
	button   Button
	checkbox Checkbox
}

// NewApplication creates a new Application with the specified factory
func NewApplication(factory GUIFactory) *Application {
	return &Application{
		factory: factory,
	}
}

// CreateUI creates the user interface components using the factory
func (a *Application) CreateUI() error {
	button, err := a.factory.CreateButton("Submit")
	if err != nil {
		return fmt.Errorf("failed to create button: %w", err)
	}
	a.button = button

	checkbox, err := a.factory.CreateCheckbox("Accept Terms")
	if err != nil {
		return fmt.Errorf("failed to create checkbox: %w", err)
	}
	a.checkbox = checkbox

	return nil
}

// Render renders all UI components
func (a *Application) Render() error {
	if a.button == nil || a.checkbox == nil {
		return errors.New("UI components not created")
	}

	if err := a.button.Render(); err != nil {
		return fmt.Errorf("failed to render button: %w", err)
	}

	if err := a.checkbox.Render(); err != nil {
		return fmt.Errorf("failed to render checkbox: %w", err)
	}

	return nil
}

// TestAbstractFactory demonstrates how to use the abstract factory pattern in production
func TestAbstractFactory() error {
	// Create factory provider
	provider := NewFactoryProvider()

	// List supported platforms
	fmt.Println("Supported platforms:", provider.GetSupportedPlatforms())

	// Create Windows application
	fmt.Println("\n=== Creating Windows Application ===")
	windowsFactory, err := provider.GetFactory(PlatformWindows)
	if err != nil {
		return fmt.Errorf("failed to get Windows factory: %w", err)
	}

	windowsApp := NewApplication(windowsFactory)
	if err := windowsApp.CreateUI(); err != nil {
		return fmt.Errorf("failed to create Windows UI: %w", err)
	}

	if err := windowsApp.Render(); err != nil {
		return fmt.Errorf("failed to render Windows UI: %w", err)
	}

	// Interact with Windows components
	if err := windowsApp.button.OnClick(); err != nil {
		return fmt.Errorf("failed to click Windows button: %w", err)
	}

	if err := windowsApp.checkbox.OnCheck(true); err != nil {
		return fmt.Errorf("failed to check Windows checkbox: %w", err)
	}

	// Create Mac application
	fmt.Println("\n=== Creating Mac Application ===")
	macFactory, err := provider.GetFactory(PlatformMac)
	if err != nil {
		return fmt.Errorf("failed to get Mac factory: %w", err)
	}

	macApp := NewApplication(macFactory)
	if err := macApp.CreateUI(); err != nil {
		return fmt.Errorf("failed to create Mac UI: %w", err)
	}

	if err := macApp.Render(); err != nil {
		return fmt.Errorf("failed to render Mac UI: %w", err)
	}

	// Interact with Mac components
	if err := macApp.button.OnClick(); err != nil {
		return fmt.Errorf("failed to click Mac button: %w", err)
	}

	if err := macApp.checkbox.OnCheck(true); err != nil {
		return fmt.Errorf("failed to check Mac checkbox: %w", err)
	}

	return nil
}
