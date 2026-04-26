// Package structural provides implementations of structural design patterns.
// This file contains a production-grade implementation of the Composite pattern
// using a file system example with files and directories.
package structural

import (
	"fmt"
	"strings"
)

// Component defines the interface for objects in the composite structure.
// It declares operations that are common to both simple and complex objects
// of the composition.
type Component interface {
	// Name returns the name of the component
	Name() string
	// Size returns the size of the component in bytes
	Size() int64
	// IsComposite returns true if the component can contain children
	IsComposite() bool
	// Add adds a child component (only applicable for composites)
	Add(component Component) error
	// Remove removes a child component by name (only applicable for composites)
	Remove(name string) error
	// GetChildren returns all child components (only applicable for composites)
	GetChildren() []Component
	// Print displays the component structure with the given indentation level
	Print(indent int)
}

// File represents a leaf node in the composite structure.
// It implements the Component interface but cannot contain children.
type File struct {
	name      string
	size      int64
	extension string
}

// Directory represents a composite node in the composite structure.
// It implements the Component interface and can contain other components.
type Directory struct {
	name     string
	children []Component
}

// NewFile creates a new File instance with validation.
// Returns an error if the name is empty or size is negative.
func NewFile(name string, size int64, extension string) (*File, error) {
	if name == "" {
		return nil, fmt.Errorf("file name cannot be empty")
	}
	if size < 0 {
		return nil, fmt.Errorf("file size cannot be negative")
	}
	return &File{
		name:      name,
		size:      size,
		extension: extension,
	}, nil
}

// NewDirectory creates a new Directory instance.
// Returns an error if the name is empty.
func NewDirectory(name string) (*Directory, error) {
	if name == "" {
		return nil, fmt.Errorf("directory name cannot be empty")
	}
	return &Directory{
		name:     name,
		children: make([]Component, 0),
	}, nil
}

// Name returns the name of the file.
func (f *File) Name() string {
	return f.name
}

// Size returns the size of the file in bytes.
func (f *File) Size() int64 {
	return f.size
}

// IsComposite returns false for files as they are leaf nodes.
func (f *File) IsComposite() bool {
	return false
}

// Add returns an error because files cannot contain children.
func (f *File) Add(component Component) error {
	return fmt.Errorf("cannot add children to a file")
}

// Remove returns an error because files cannot contain children.
func (f *File) Remove(name string) error {
	return fmt.Errorf("cannot remove children from a file")
}

// GetChildren returns an empty slice for files.
func (f *File) GetChildren() []Component {
	return []Component{}
}

// Print displays the file information with proper indentation.
func (f *File) Print(indent int) {
	indentation := strings.Repeat("  ", indent)
	extension := ""
	if f.extension != "" {
		extension = fmt.Sprintf(".%s", f.extension)
	}
	fmt.Printf("%s📄 %s%s (%.2f KB)\n", indentation, f.name, extension, float64(f.size)/1024)
}

// Name returns the name of the directory.
func (d *Directory) Name() string {
	return d.name
}

// Size calculates and returns the total size of the directory
// by recursively summing up the sizes of all children.
func (d *Directory) Size() int64 {
	var totalSize int64
	for _, child := range d.children {
		totalSize += child.Size()
	}
	return totalSize
}

// IsComposite returns true for directories as they are composite nodes.
func (d *Directory) IsComposite() bool {
	return true
}

// Add adds a child component to the directory.
// Returns an error if the component is nil or a component with the same name exists.
func (d *Directory) Add(component Component) error {
	if component == nil {
		return fmt.Errorf("cannot add nil component")
	}

	// Check for duplicate names
	for _, child := range d.children {
		if child.Name() == component.Name() {
			return fmt.Errorf("component with name '%s' already exists", component.Name())
		}
	}

	d.children = append(d.children, component)
	return nil
}

// Remove removes a child component from the directory by name.
// Returns an error if the component is not found.
func (d *Directory) Remove(name string) error {
	for i, child := range d.children {
		if child.Name() == name {
			d.children = append(d.children[:i], d.children[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("component with name '%s' not found", name)
}

// GetChildren returns all child components of the directory.
func (d *Directory) GetChildren() []Component {
	// Return a copy to prevent external modifications
	children := make([]Component, len(d.children))
	copy(children, d.children)
	return children
}

// Print displays the directory structure recursively with proper indentation.
func (d *Directory) Print(indent int) {
	indentation := strings.Repeat("  ", indent)
	fmt.Printf("%s📁 %s/ (%.2f KB)\n", indentation, d.name, float64(d.Size())/1024)

	for _, child := range d.children {
		child.Print(indent + 1)
	}
}
