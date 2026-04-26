// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Visitor pattern
// that works with file system components from the Composite pattern.
//
// The Visitor pattern allows you to separate algorithms from the objects on which they operate.
// This makes it easy to add new operations without modifying the existing component classes.
package behavioral

import (
	"fmt"
	"strings"
)

// Visitor defines the interface for visitors that can perform operations
// on different types of components in the file system hierarchy.
// Each Visit method handles a specific component type.
type Visitor interface {
	// VisitFile performs an operation on a file component
	VisitFile(file VisitableFile) error
	// VisitDirectory performs an operation on a directory component
	VisitDirectory(directory VisitableDirectory) error
	// GetResult returns the result of the visitor's operations
	GetResult() interface{}
}

// Visitable defines the interface for components that can accept visitors.
// Components implementing this interface allow visitors to perform operations on them.
type Visitable interface {
	// Accept accepts a visitor and allows it to perform operations
	Accept(visitor Visitor) error
}

// VisitableFile represents a file component that can be visited.
// It provides access to file-specific properties needed by visitors.
type VisitableFile interface {
	Visitable
	GetName() string
	GetSize() int64
	GetExtension() string
}

// VisitableDirectory represents a directory component that can be visited.
// It provides access to directory-specific properties and children.
type VisitableDirectory interface {
	Visitable
	GetName() string
	GetChildren() []Visitable
}

// FileNode represents a file in the visitor pattern context.
// It implements VisitableFile interface.
type FileNode struct {
	name      string
	size      int64
	extension string
}

// DirectoryNode represents a directory in the visitor pattern context.
// It implements VisitableDirectory interface.
type DirectoryNode struct {
	name     string
	children []Visitable
}

// NewFileNode creates a new FileNode with validation.
func NewFileNode(name string, size int64, extension string) (*FileNode, error) {
	if name == "" {
		return nil, fmt.Errorf("file name cannot be empty")
	}
	if size < 0 {
		return nil, fmt.Errorf("file size cannot be negative")
	}
	return &FileNode{
		name:      name,
		size:      size,
		extension: extension,
	}, nil
}

// NewDirectoryNode creates a new DirectoryNode with validation.
func NewDirectoryNode(name string) (*DirectoryNode, error) {
	if name == "" {
		return nil, fmt.Errorf("directory name cannot be empty")
	}
	return &DirectoryNode{
		name:     name,
		children: make([]Visitable, 0),
	}, nil
}

// GetName returns the file name.
func (f *FileNode) GetName() string {
	return f.name
}

// GetSize returns the file size in bytes.
func (f *FileNode) GetSize() int64 {
	return f.size
}

// GetExtension returns the file extension.
func (f *FileNode) GetExtension() string {
	return f.extension
}

// Accept implements the Visitable interface for FileNode.
func (f *FileNode) Accept(visitor Visitor) error {
	if visitor == nil {
		return fmt.Errorf("visitor cannot be nil")
	}
	return visitor.VisitFile(f)
}

// GetName returns the directory name.
func (d *DirectoryNode) GetName() string {
	return d.name
}

// GetChildren returns the directory's children.
func (d *DirectoryNode) GetChildren() []Visitable {
	children := make([]Visitable, len(d.children))
	copy(children, d.children)
	return children
}

// AddChild adds a child to the directory.
func (d *DirectoryNode) AddChild(child Visitable) error {
	if child == nil {
		return fmt.Errorf("cannot add nil child")
	}
	d.children = append(d.children, child)
	return nil
}

// Accept implements the Visitable interface for DirectoryNode.
// It visits the directory itself and then recursively visits all children.
func (d *DirectoryNode) Accept(visitor Visitor) error {
	if visitor == nil {
		return fmt.Errorf("visitor cannot be nil")
	}

	// Visit the directory itself
	if err := visitor.VisitDirectory(d); err != nil {
		return err
	}

	// Recursively visit all children
	for _, child := range d.children {
		if err := child.Accept(visitor); err != nil {
			return err
		}
	}

	return nil
}

// SizeCalculatorVisitor calculates the total size of files in the hierarchy.
// This is a concrete visitor implementation that demonstrates aggregation operations.
type SizeCalculatorVisitor struct {
	totalSize      int64
	fileCount      int
	directoryCount int
}

// NewSizeCalculatorVisitor creates a new SizeCalculatorVisitor.
func NewSizeCalculatorVisitor() *SizeCalculatorVisitor {
	return &SizeCalculatorVisitor{
		totalSize:      0,
		fileCount:      0,
		directoryCount: 0,
	}
}

// VisitFile adds the file's size to the total.
func (v *SizeCalculatorVisitor) VisitFile(file VisitableFile) error {
	v.totalSize += file.GetSize()
	v.fileCount++
	return nil
}

// VisitDirectory increments the directory count.
func (v *SizeCalculatorVisitor) VisitDirectory(directory VisitableDirectory) error {
	v.directoryCount++
	return nil
}

// GetResult returns a map with the calculated statistics.
func (v *SizeCalculatorVisitor) GetResult() interface{} {
	return map[string]interface{}{
		"totalSize":      v.totalSize,
		"totalSizeKB":    float64(v.totalSize) / 1024,
		"totalSizeMB":    float64(v.totalSize) / (1024 * 1024),
		"fileCount":      v.fileCount,
		"directoryCount": v.directoryCount,
	}
}

// GetTotalSize returns the total size in bytes.
func (v *SizeCalculatorVisitor) GetTotalSize() int64 {
	return v.totalSize
}

// GetFileCount returns the number of files visited.
func (v *SizeCalculatorVisitor) GetFileCount() int {
	return v.fileCount
}

// GetDirectoryCount returns the number of directories visited.
func (v *SizeCalculatorVisitor) GetDirectoryCount() int {
	return v.directoryCount
}

// SearchVisitor searches for components matching specific criteria.
// This demonstrates filtering operations using the visitor pattern.
type SearchVisitor struct {
	searchName      string
	searchExtension string
	caseSensitive   bool
	matchedFiles    []string
	matchedDirs     []string
}

// SearchOptions configures the search visitor behavior.
type SearchOptions struct {
	Name          string
	Extension     string
	CaseSensitive bool
}

// NewSearchVisitor creates a new SearchVisitor with the given options.
func NewSearchVisitor(options SearchOptions) *SearchVisitor {
	return &SearchVisitor{
		searchName:      options.Name,
		searchExtension: options.Extension,
		caseSensitive:   options.CaseSensitive,
		matchedFiles:    make([]string, 0),
		matchedDirs:     make([]string, 0),
	}
}

// VisitFile checks if the file matches the search criteria.
func (v *SearchVisitor) VisitFile(file VisitableFile) error {
	name := file.GetName()
	extension := file.GetExtension()

	nameMatches := v.searchName == ""
	extensionMatches := v.searchExtension == ""

	if v.searchName != "" {
		if v.caseSensitive {
			nameMatches = strings.Contains(name, v.searchName)
		} else {
			nameMatches = strings.Contains(strings.ToLower(name), strings.ToLower(v.searchName))
		}
	}

	if v.searchExtension != "" {
		if v.caseSensitive {
			extensionMatches = extension == v.searchExtension
		} else {
			extensionMatches = strings.EqualFold(extension, v.searchExtension)
		}
	}

	if nameMatches && extensionMatches {
		fullName := name
		if extension != "" {
			fullName = fmt.Sprintf("%s.%s", name, extension)
		}
		v.matchedFiles = append(v.matchedFiles, fullName)
	}

	return nil
}

// VisitDirectory checks if the directory matches the search criteria.
func (v *SearchVisitor) VisitDirectory(directory VisitableDirectory) error {
	if v.searchName == "" {
		return nil
	}

	name := directory.GetName()
	matches := false

	if v.caseSensitive {
		matches = strings.Contains(name, v.searchName)
	} else {
		matches = strings.Contains(strings.ToLower(name), strings.ToLower(v.searchName))
	}

	if matches {
		v.matchedDirs = append(v.matchedDirs, name)
	}

	return nil
}

// GetResult returns the search results.
func (v *SearchVisitor) GetResult() interface{} {
	return map[string]interface{}{
		"matchedFiles":       v.matchedFiles,
		"matchedDirectories": v.matchedDirs,
		"totalMatches":       len(v.matchedFiles) + len(v.matchedDirs),
	}
}

// GetMatchedFiles returns the list of matched files.
func (v *SearchVisitor) GetMatchedFiles() []string {
	result := make([]string, len(v.matchedFiles))
	copy(result, v.matchedFiles)
	return result
}

// GetMatchedDirectories returns the list of matched directories.
func (v *SearchVisitor) GetMatchedDirectories() []string {
	result := make([]string, len(v.matchedDirs))
	copy(result, v.matchedDirs)
	return result
}

// DetailReportVisitor generates a detailed report of the file system structure.
// This demonstrates reporting and formatting operations using the visitor pattern.
type DetailReportVisitor struct {
	report           strings.Builder
	currentIndent    int
	showSize         bool
	showExtension    bool
	totalSize        int64
	filesByExtension map[string]int
}

// ReportOptions configures the detail report visitor behavior.
type ReportOptions struct {
	ShowSize      bool
	ShowExtension bool
}

// NewDetailReportVisitor creates a new DetailReportVisitor with the given options.
func NewDetailReportVisitor(options ReportOptions) *DetailReportVisitor {
	return &DetailReportVisitor{
		report:           strings.Builder{},
		currentIndent:    0,
		showSize:         options.ShowSize,
		showExtension:    options.ShowExtension,
		totalSize:        0,
		filesByExtension: make(map[string]int),
	}
}

// VisitFile adds the file information to the report.
func (v *DetailReportVisitor) VisitFile(file VisitableFile) error {
	indent := strings.Repeat("  ", v.currentIndent)
	name := file.GetName()
	extension := file.GetExtension()
	size := file.GetSize()

	v.totalSize += size

	// Track files by extension
	ext := extension
	if ext == "" {
		ext = "no-extension"
	}
	v.filesByExtension[ext]++

	// Build file representation
	fileInfo := fmt.Sprintf("%s📄 %s", indent, name)

	if v.showExtension && extension != "" {
		fileInfo += fmt.Sprintf(".%s", extension)
	}

	if v.showSize {
		fileInfo += fmt.Sprintf(" (%.2f KB)", float64(size)/1024)
	}

	v.report.WriteString(fileInfo + "\n")
	return nil
}

// VisitDirectory adds the directory information to the report and manages indentation.
func (v *DetailReportVisitor) VisitDirectory(directory VisitableDirectory) error {
	indent := strings.Repeat("  ", v.currentIndent)
	name := directory.GetName()

	v.report.WriteString(fmt.Sprintf("%s📁 %s/\n", indent, name))
	v.currentIndent++

	return nil
}

// GetResult returns the formatted report with statistics.
func (v *DetailReportVisitor) GetResult() interface{} {
	// Add summary statistics
	summary := strings.Builder{}
	summary.WriteString("\n=== Summary ===\n")
	summary.WriteString(fmt.Sprintf("Total Size: %.2f MB\n", float64(v.totalSize)/(1024*1024)))
	summary.WriteString("\nFiles by Extension:\n")

	for ext, count := range v.filesByExtension {
		summary.WriteString(fmt.Sprintf("  .%s: %d file(s)\n", ext, count))
	}

	return v.report.String() + summary.String()
}

// GetReport returns the formatted report as a string.
func (v *DetailReportVisitor) GetReport() string {
	result := v.GetResult()
	if str, ok := result.(string); ok {
		return str
	}
	return ""
}

// GetFilesByExtension returns the file count grouped by extension.
func (v *DetailReportVisitor) GetFilesByExtension() map[string]int {
	result := make(map[string]int)
	for k, v := range v.filesByExtension {
		result[k] = v
	}
	return result
}
