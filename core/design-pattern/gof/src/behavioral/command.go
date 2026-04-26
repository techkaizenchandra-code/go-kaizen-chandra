// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Command pattern
// using a text editor example with undo/redo functionality.
//
// The Command pattern encapsulates a request as an object, allowing you to
// parameterize clients with different requests, queue or log requests,
// and support undoable operations.
package behavioral

import (
	"fmt"
	"strings"
)

// Command defines the interface for all concrete commands.
// Each command encapsulates an action and its undo operation.
type Command interface {
	// Execute performs the command action
	Execute() error
	// Undo reverses the command action
	Undo() error
	// String returns a string representation of the command
	String() string
}

// Receiver defines the interface for objects that commands act upon.
type Receiver interface {
	// Content returns the current state of the receiver
	Content() string
}

// TextEditor represents the receiver in the Command pattern.
// It maintains the state that commands operate on.
type TextEditor struct {
	content string
	cursor  int
}

// NewTextEditor creates a new TextEditor instance.
// Returns an error if initial content is invalid.
func NewTextEditor(initialContent string) (*TextEditor, error) {
	return &TextEditor{
		content: initialContent,
		cursor:  len(initialContent),
	}, nil
}

// Content returns the current content of the text editor.
func (t *TextEditor) Content() string {
	return t.content
}

// Insert adds text at the specified position.
// Returns an error if the position is invalid.
func (t *TextEditor) Insert(text string, position int) error {
	if position < 0 || position > len(t.content) {
		return fmt.Errorf("invalid position: %d (content length: %d)", position, len(t.content))
	}
	t.content = t.content[:position] + text + t.content[position:]
	t.cursor = position + len(text)
	return nil
}

// Delete removes text from the specified position with the given length.
// Returns the deleted text and an error if the operation fails.
func (t *TextEditor) Delete(position, length int) (string, error) {
	if position < 0 || position >= len(t.content) {
		return "", fmt.Errorf("invalid position: %d (content length: %d)", position, len(t.content))
	}
	if position+length > len(t.content) {
		return "", fmt.Errorf("delete length exceeds content bounds")
	}
	deleted := t.content[position : position+length]
	t.content = t.content[:position] + t.content[position+length:]
	t.cursor = position
	return deleted, nil
}

// MoveCursor moves the cursor to the specified position.
func (t *TextEditor) MoveCursor(position int) error {
	if position < 0 || position > len(t.content) {
		return fmt.Errorf("invalid cursor position: %d", position)
	}
	t.cursor = position
	return nil
}

// InsertTextCommand represents a concrete command for inserting text.
type InsertTextCommand struct {
	editor   *TextEditor
	text     string
	position int
}

// NewInsertTextCommand creates a new InsertTextCommand.
// Returns an error if the editor is nil or text is empty.
func NewInsertTextCommand(editor *TextEditor, text string, position int) (*InsertTextCommand, error) {
	if editor == nil {
		return nil, fmt.Errorf("editor cannot be nil")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	return &InsertTextCommand{
		editor:   editor,
		text:     text,
		position: position,
	}, nil
}

// Execute performs the insert operation.
func (c *InsertTextCommand) Execute() error {
	return c.editor.Insert(c.text, c.position)
}

// Undo reverses the insert operation by deleting the inserted text.
func (c *InsertTextCommand) Undo() error {
	_, err := c.editor.Delete(c.position, len(c.text))
	return err
}

// String returns a string representation of the command.
func (c *InsertTextCommand) String() string {
	displayText := c.text
	if len(displayText) > 20 {
		displayText = displayText[:20] + "..."
	}
	return fmt.Sprintf("Insert '%s' at position %d", displayText, c.position)
}

// DeleteTextCommand represents a concrete command for deleting text.
type DeleteTextCommand struct {
	editor      *TextEditor
	position    int
	length      int
	deletedText string
}

// NewDeleteTextCommand creates a new DeleteTextCommand.
// Returns an error if the editor is nil or length is invalid.
func NewDeleteTextCommand(editor *TextEditor, position, length int) (*DeleteTextCommand, error) {
	if editor == nil {
		return nil, fmt.Errorf("editor cannot be nil")
	}
	if length <= 0 {
		return nil, fmt.Errorf("delete length must be positive")
	}
	return &DeleteTextCommand{
		editor:   editor,
		position: position,
		length:   length,
	}, nil
}

// Execute performs the delete operation and stores the deleted text.
func (c *DeleteTextCommand) Execute() error {
	deleted, err := c.editor.Delete(c.position, c.length)
	if err != nil {
		return err
	}
	c.deletedText = deleted
	return nil
}

// Undo reverses the delete operation by re-inserting the deleted text.
func (c *DeleteTextCommand) Undo() error {
	if c.deletedText == "" {
		return fmt.Errorf("no deleted text to restore")
	}
	return c.editor.Insert(c.deletedText, c.position)
}

// String returns a string representation of the command.
func (c *DeleteTextCommand) String() string {
	return fmt.Sprintf("Delete %d characters at position %d", c.length, c.position)
}

// CommandHistory manages the execution and undo/redo of commands.
// It maintains a history of executed commands and tracks the current position.
type CommandHistory struct {
	history []Command
	current int // Points to the next command to redo (-1 if at the end)
}

// NewCommandHistory creates a new CommandHistory instance.
func NewCommandHistory() *CommandHistory {
	return &CommandHistory{
		history: make([]Command, 0),
		current: -1,
	}
}

// ExecuteCommand executes a command and adds it to the history.
// It clears any commands that were undone before executing the new command.
func (h *CommandHistory) ExecuteCommand(cmd Command) error {
	if cmd == nil {
		return fmt.Errorf("command cannot be nil")
	}

	// Execute the command
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	// Clear any commands after the current position (they were undone)
	h.history = h.history[:h.current+1]

	// Add the command to history
	h.history = append(h.history, cmd)
	h.current = len(h.history) - 1

	return nil
}

// Undo undoes the last executed command.
// Returns an error if there are no commands to undo.
func (h *CommandHistory) Undo() error {
	if !h.CanUndo() {
		return fmt.Errorf("no commands to undo")
	}

	cmd := h.history[h.current]
	if err := cmd.Undo(); err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	h.current--
	return nil
}

// Redo re-executes a previously undone command.
// Returns an error if there are no commands to redo.
func (h *CommandHistory) Redo() error {
	if !h.CanRedo() {
		return fmt.Errorf("no commands to redo")
	}

	h.current++
	cmd := h.history[h.current]
	if err := cmd.Execute(); err != nil {
		h.current-- // Rollback on error
		return fmt.Errorf("redo failed: %w", err)
	}

	return nil
}

// CanUndo returns true if there are commands that can be undone.
func (h *CommandHistory) CanUndo() bool {
	return h.current >= 0
}

// CanRedo returns true if there are commands that can be redone.
func (h *CommandHistory) CanRedo() bool {
	return h.current < len(h.history)-1
}

// GetHistory returns a string representation of all commands in history.
func (h *CommandHistory) GetHistory() string {
	if len(h.history) == 0 {
		return "No commands in history"
	}

	var builder strings.Builder
	builder.WriteString("Command History:\n")
	for i, cmd := range h.history {
		marker := "  "
		if i == h.current {
			marker = "→ "
		} else if i > h.current {
			marker = "  [undone] "
		}
		builder.WriteString(fmt.Sprintf("%s%d. %s\n", marker, i+1, cmd.String()))
	}
	return builder.String()
}

// Clear removes all commands from the history.
func (h *CommandHistory) Clear() {
	h.history = make([]Command, 0)
	h.current = -1
}
