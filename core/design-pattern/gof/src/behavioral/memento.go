// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Memento pattern
// using a text editor example with undo/redo functionality and state management.
package behavioral

import (
	"fmt"
	"time"
)

// Memento interface defines the contract for memento objects that store state.
// It provides methods to retrieve the stored state without exposing the internal structure.
type Memento interface {
	// GetState returns the stored state as a string representation
	GetState() string
	// GetTimestamp returns when this memento was created
	GetTimestamp() time.Time
}

// TextMemento is a concrete implementation of Memento that stores text editor state.
// It encapsulates the state of a TextEditor including content and cursor position.
type TextMemento struct {
	content        string
	cursorPosition int
	timestamp      time.Time
}

// NewTextMemento creates a new TextMemento instance with validation.
// Returns an error if cursor position is negative.
func NewTextMemento(content string, cursorPosition int) (*TextMemento, error) {
	if cursorPosition < 0 {
		return nil, fmt.Errorf("cursor position cannot be negative")
	}
	if cursorPosition > len(content) {
		return nil, fmt.Errorf("cursor position cannot exceed content length")
	}

	return &TextMemento{
		content:        content,
		cursorPosition: cursorPosition,
		timestamp:      time.Now(),
	}, nil
}

// GetState returns the stored state as a formatted string.
func (m *TextMemento) GetState() string {
	return fmt.Sprintf("Content: %s, Cursor: %d", m.content, m.cursorPosition)
}

// GetTimestamp returns when this memento was created.
func (m *TextMemento) GetTimestamp() time.Time {
	return m.timestamp
}

// Originator interface defines the contract for objects that can create and restore from mementos.
type Originator interface {
	// CreateMemento creates a memento containing the current state
	CreateMemento() (Memento, error)
	// RestoreFromMemento restores the state from the given memento
	RestoreFromMemento(memento Memento) error
}

// TextEditor is a concrete implementation of Originator.
// It represents a simple text editor with content and cursor management.
type TextEditor2 struct {
	content        string
	cursorPosition int
	fileName       string
	lastModified   time.Time
}

// NewTextEditor creates a new TextEditor instance.
// Returns an error if the file name is empty.
func NewTextEditor2(fileName string) (*TextEditor2, error) {
	if fileName == "" {
		return nil, fmt.Errorf("file name cannot be empty")
	}

	return &TextEditor2{
		content:        "",
		cursorPosition: 0,
		fileName:       fileName,
		lastModified:   time.Now(),
	}, nil
}

// SetContent updates the editor content and cursor position.
// Returns an error if cursor position is invalid.
func (e *TextEditor2) SetContent(content string, cursorPosition int) error {
	if cursorPosition < 0 {
		return fmt.Errorf("cursor position cannot be negative")
	}
	if cursorPosition > len(content) {
		return fmt.Errorf("cursor position cannot exceed content length")
	}

	e.content = content
	e.cursorPosition = cursorPosition
	e.lastModified = time.Now()
	return nil
}

// GetContent returns the current content of the editor.
func (e *TextEditor2) GetContent() string {
	return e.content
}

// CreateMemento creates and returns a memento containing the current editor state.
func (e *TextEditor2) CreateMemento() (Memento, error) {
	return NewTextMemento(e.content, e.cursorPosition)
}

// RestoreFromMemento restores the editor state from the given memento.
// Returns an error if the memento is not a TextMemento or if restoration fails.
func (e *TextEditor2) RestoreFromMemento(memento Memento) error {
	if memento == nil {
		return fmt.Errorf("memento cannot be nil")
	}

	textMemento, ok := memento.(*TextMemento)
	if !ok {
		return fmt.Errorf("invalid memento type")
	}

	e.content = textMemento.content
	e.cursorPosition = textMemento.cursorPosition
	e.lastModified = time.Now()
	return nil
}

// GetMetadata returns metadata about the current editor state.
func (e *TextEditor2) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"fileName":       e.fileName,
		"contentLength":  len(e.content),
		"cursorPosition": e.cursorPosition,
		"lastModified":   e.lastModified,
	}
}

// Caretaker interface defines the contract for managing memento history.
// It provides undo/redo functionality and history management.
type Caretaker interface {
	// Save stores a memento in the history
	Save(memento Memento) error
	// Undo retrieves the previous memento from history
	Undo() (Memento, error)
	// Redo retrieves the next memento from history
	Redo() (Memento, error)
	// CanUndo returns true if undo operation is available
	CanUndo() bool
	// CanRedo returns true if redo operation is available
	CanRedo() bool
	// GetHistorySize returns the current number of mementos in history
	GetHistorySize() int
	// Clear removes all mementos from history
	Clear()
}

// History is a concrete implementation of Caretaker.
// It manages a collection of mementos with undo/redo support and size limits.
type History struct {
	mementos     []Memento
	currentIndex int
	maxSize      int
}

// NewHistory creates a new History instance with the specified maximum size.
// Returns an error if maxSize is less than 1.
func NewHistory(maxSize int) (*History, error) {
	if maxSize < 1 {
		return nil, fmt.Errorf("history max size must be at least 1")
	}

	return &History{
		mementos:     make([]Memento, 0, maxSize),
		currentIndex: -1,
		maxSize:      maxSize,
	}, nil
}

// Save stores a memento in the history.
// If at an intermediate position, it clears redo history.
// Implements circular buffer behavior when history size limit is reached.
func (h *History) Save(memento Memento) error {
	if memento == nil {
		return fmt.Errorf("cannot save nil memento")
	}

	// Clear any redo history if we're not at the end
	if h.currentIndex < len(h.mementos)-1 {
		h.mementos = h.mementos[:h.currentIndex+1]
	}

	// Add the new memento
	h.mementos = append(h.mementos, memento)
	h.currentIndex++

	// Remove oldest memento if we exceed max size
	if len(h.mementos) > h.maxSize {
		h.mementos = h.mementos[1:]
		h.currentIndex--
	}

	return nil
}

// Undo retrieves the previous memento from history.
// Returns an error if undo is not available.
func (h *History) Undo() (Memento, error) {
	if !h.CanUndo() {
		return nil, fmt.Errorf("no more undo operations available")
	}

	h.currentIndex--
	return h.mementos[h.currentIndex], nil
}

// Redo retrieves the next memento from history.
// Returns an error if redo is not available.
func (h *History) Redo() (Memento, error) {
	if !h.CanRedo() {
		return nil, fmt.Errorf("no more redo operations available")
	}

	h.currentIndex++
	return h.mementos[h.currentIndex], nil
}

// CanUndo returns true if undo operation is available.
func (h *History) CanUndo() bool {
	return h.currentIndex > 0
}

// CanRedo returns true if redo operation is available.
func (h *History) CanRedo() bool {
	return h.currentIndex < len(h.mementos)-1
}

// GetHistorySize returns the current number of mementos in history.
func (h *History) GetHistorySize() int {
	return len(h.mementos)
}

// Clear removes all mementos from history and resets the index.
func (h *History) Clear() {
	h.mementos = make([]Memento, 0, h.maxSize)
	h.currentIndex = -1
}
