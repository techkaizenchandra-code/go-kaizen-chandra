// Package structural provides implementations of structural design patterns.
// This file contains a production-grade implementation of the Flyweight pattern
// using a text rendering example where font styles are shared (flyweights)
// and character positions are unique (extrinsic state).
//
// The Flyweight pattern is used to minimize memory usage by sharing as much data
// as possible with similar objects. It's particularly useful when dealing with
// a large number of objects that share common state.
package structural

import (
	"fmt"
	"sync"
)

// FontStyle represents the intrinsic state (shared data) in the Flyweight pattern.
// Multiple characters can share the same FontStyle to reduce memory consumption.
type FontStyle struct {
	family string // Font family name (e.g., "Arial", "Times New Roman")
	size   int    // Font size in points
	weight string // Font weight (e.g., "normal", "bold")
	style  string // Font style (e.g., "normal", "italic")
	color  string // Font color in hex format
}

// FontFactory manages the creation and sharing of FontStyle flyweight objects.
// It implements the Flyweight Factory pattern with thread-safe access.
type FontFactory struct {
	fontStyles map[string]*FontStyle // Cache of shared font styles
	mu         sync.RWMutex          // Mutex for thread-safe operations
}

// Character represents an individual character with its extrinsic state.
// It uses a shared FontStyle (flyweight) and maintains unique position data.
type Character struct {
	char      rune       // The actual character
	x         int        // X position in the document
	y         int        // Y position in the document
	fontStyle *FontStyle // Shared font style (flyweight)
}

// TextDocument represents a document containing multiple characters.
// It demonstrates the practical usage of the Flyweight pattern.
type TextDocument struct {
	name       string       // Document name
	characters []*Character // List of characters in the document
	factory    *FontFactory // Reference to the font factory
}

// NewFontStyle creates a new FontStyle with validation.
// Returns an error if any required parameter is invalid.
func NewFontStyle(family string, size int, weight, style, color string) (*FontStyle, error) {
	if family == "" {
		return nil, fmt.Errorf("font family cannot be empty")
	}
	if size <= 0 {
		return nil, fmt.Errorf("font size must be positive, got %d", size)
	}
	if weight == "" {
		return nil, fmt.Errorf("font weight cannot be empty")
	}
	if style == "" {
		return nil, fmt.Errorf("font style cannot be empty")
	}
	if color == "" {
		return nil, fmt.Errorf("font color cannot be empty")
	}

	return &FontStyle{
		family: family,
		size:   size,
		weight: weight,
		style:  style,
		color:  color,
	}, nil
}

// NewFontFactory creates and initializes a new FontFactory.
func NewFontFactory() *FontFactory {
	return &FontFactory{
		fontStyles: make(map[string]*FontStyle),
	}
}

// GetFontStyle retrieves an existing font style or creates a new one if it doesn't exist.
// This method implements the core flyweight pattern logic with thread-safe lazy initialization.
// The key is generated from the font parameters to ensure uniqueness.
func (ff *FontFactory) GetFontStyle(family string, size int, weight, style, color string) (*FontStyle, error) {
	// Generate a unique key for this font style combination
	key := fmt.Sprintf("%s-%d-%s-%s-%s", family, size, weight, style, color)

	// Check if the font style already exists (read lock)
	ff.mu.RLock()
	if fontStyle, exists := ff.fontStyles[key]; exists {
		ff.mu.RUnlock()
		return fontStyle, nil
	}
	ff.mu.RUnlock()

	// Create new font style (write lock)
	ff.mu.Lock()
	defer ff.mu.Unlock()

	// Double-check in case another goroutine created it
	if fontStyle, exists := ff.fontStyles[key]; exists {
		return fontStyle, nil
	}

	// Create and validate the new font style
	fontStyle, err := NewFontStyle(family, size, weight, style, color)
	if err != nil {
		return nil, fmt.Errorf("failed to create font style: %w", err)
	}

	// Store the new font style
	ff.fontStyles[key] = fontStyle
	return fontStyle, nil
}

// GetFontCount returns the number of unique font styles currently cached.
// Useful for monitoring and metrics.
func (ff *FontFactory) GetFontCount() int {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return len(ff.fontStyles)
}

// Clear removes all cached font styles.
// Useful for cleanup or resetting the factory state.
func (ff *FontFactory) Clear() {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.fontStyles = make(map[string]*FontStyle)
}

// NewCharacter creates a new Character with the specified parameters.
// Returns an error if the font style is nil.
func NewCharacter(char rune, x, y int, fontStyle *FontStyle) (*Character, error) {
	if fontStyle == nil {
		return nil, fmt.Errorf("font style cannot be nil")
	}

	return &Character{
		char:      char,
		x:         x,
		y:         y,
		fontStyle: fontStyle,
	}, nil
}

// Render displays the character with its formatting information.
// In a real application, this would render the character to a screen or document.
func (c *Character) Render() string {
	return fmt.Sprintf("Char '%c' at (%d,%d) - Font: %s, Size: %d, Weight: %s, Style: %s, Color: %s",
		c.char, c.x, c.y,
		c.fontStyle.family,
		c.fontStyle.size,
		c.fontStyle.weight,
		c.fontStyle.style,
		c.fontStyle.color,
	)
}

// NewTextDocument creates a new TextDocument with the specified name and factory.
// Returns an error if the name is empty or factory is nil.
func NewTextDocument(name string, factory *FontFactory) (*TextDocument, error) {
	if name == "" {
		return nil, fmt.Errorf("document name cannot be empty")
	}
	if factory == nil {
		return nil, fmt.Errorf("font factory cannot be nil")
	}

	return &TextDocument{
		name:       name,
		characters: make([]*Character, 0),
		factory:    factory,
	}, nil
}

// AddCharacter adds a character to the document with the specified formatting.
// It uses the factory to get or create the appropriate font style.
func (td *TextDocument) AddCharacter(char rune, x, y int, family string, size int, weight, style, color string) error {
	// Get or create the font style using the factory
	fontStyle, err := td.factory.GetFontStyle(family, size, weight, style, color)
	if err != nil {
		return fmt.Errorf("failed to get font style: %w", err)
	}

	// Create the character
	character, err := NewCharacter(char, x, y, fontStyle)
	if err != nil {
		return fmt.Errorf("failed to create character: %w", err)
	}

	td.characters = append(td.characters, character)
	return nil
}

// Render displays all characters in the document.
func (td *TextDocument) Render() {
	fmt.Printf("\n=== Document: %s ===\n", td.name)
	fmt.Printf("Total characters: %d\n", len(td.characters))
	fmt.Printf("Unique font styles: %d\n\n", td.factory.GetFontCount())

	for i, char := range td.characters {
		fmt.Printf("[%d] %s\n", i+1, char.Render())
	}
}

// GetMemoryStats returns statistics about memory usage.
// This demonstrates the memory savings achieved by the Flyweight pattern.
func (td *TextDocument) GetMemoryStats() map[string]interface{} {
	return map[string]interface{}{
		"total_characters":   len(td.characters),
		"unique_font_styles": td.factory.GetFontCount(),
		"memory_saved":       fmt.Sprintf("%.2f%%", (1.0-float64(td.factory.GetFontCount())/float64(max(len(td.characters), 1)))*100),
	}
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
