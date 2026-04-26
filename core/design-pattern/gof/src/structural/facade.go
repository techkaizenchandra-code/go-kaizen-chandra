// Package structural provides implementations of structural design patterns.
// This file contains a production-grade implementation of the Facade pattern
// that provides a simplified interface to the complex composite file system.
//
// The Facade pattern provides a unified interface to a set of interfaces in a subsystem.
// It defines a higher-level interface that makes the subsystem easier to use by wrapping
// a complex subsystem with a simpler interface.
//
// In this implementation:
// - FileSystemFacade acts as the facade that simplifies file system operations
// - The composite pattern (Directory and File) represents the complex subsystem
// - Clients interact with the simple facade methods instead of complex composite operations
package structural

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// FileSystemFacade provides a simplified interface to the complex file system
// built using the composite pattern. It encapsulates the complexity of creating,
// managing, and querying the file system structure.
type FileSystemFacade struct {
	root *Directory
	name string
}

// FileSystemStats holds statistics about the file system.
type FileSystemStats struct {
	TotalFiles       int
	TotalDirectories int
	TotalSize        int64
	LargestFile      string
	LargestFileSize  int64
	DeepestLevel     int
}

// NewFileSystemFacade creates a new FileSystemFacade with a root directory.
// Returns an error if the root name is empty.
func NewFileSystemFacade(rootName string) (*FileSystemFacade, error) {
	if rootName == "" {
		return nil, fmt.Errorf("root directory name cannot be empty")
	}

	root, err := NewDirectory(rootName)
	if err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &FileSystemFacade{
		root: root,
		name: rootName,
	}, nil
}

// CreateFile creates a new file and adds it to the specified directory path.
// The path uses forward slashes as separators (e.g., "dir1/dir2").
// Returns an error if the path is invalid or the file cannot be created.
func (fs *FileSystemFacade) CreateFile(path, fileName string, size int64, extension string) error {
	if fileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}

	file, err := NewFile(fileName, size, extension)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	targetDir, err := fs.getDirectoryByPath(path)
	if err != nil {
		return fmt.Errorf("failed to locate directory '%s': %w", path, err)
	}

	if err := targetDir.Add(file); err != nil {
		return fmt.Errorf("failed to add file to directory: %w", err)
	}

	return nil
}

// CreateDirectory creates a new directory and adds it to the specified parent path.
// The path uses forward slashes as separators (e.g., "dir1/dir2").
// Use empty string or "/" for root level.
// Returns an error if the path is invalid or the directory cannot be created.
func (fs *FileSystemFacade) CreateDirectory(path, dirName string) error {
	if dirName == "" {
		return fmt.Errorf("directory name cannot be empty")
	}

	newDir, err := NewDirectory(dirName)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	targetDir, err := fs.getDirectoryByPath(path)
	if err != nil {
		return fmt.Errorf("failed to locate parent directory '%s': %w", path, err)
	}

	if err := targetDir.Add(newDir); err != nil {
		return fmt.Errorf("failed to add directory to parent: %w", err)
	}

	return nil
}

// DeleteComponent removes a file or directory at the specified path.
// The path should include the component name (e.g., "dir1/dir2/file.txt").
// Returns an error if the path is invalid or the component cannot be found.
func (fs *FileSystemFacade) DeleteComponent(path string) error {
	if path == "" || path == "/" {
		return fmt.Errorf("cannot delete root directory")
	}

	parentPath, componentName := fs.splitPath(path)
	parentDir, err := fs.getDirectoryByPath(parentPath)
	if err != nil {
		return fmt.Errorf("failed to locate parent directory: %w", err)
	}

	if err := parentDir.Remove(componentName); err != nil {
		return fmt.Errorf("failed to remove component: %w", err)
	}

	return nil
}

// GetComponentByPath retrieves a component (file or directory) by its path.
// The path uses forward slashes as separators (e.g., "dir1/dir2/file.txt").
// Returns an error if the component cannot be found.
func (fs *FileSystemFacade) GetComponentByPath(path string) (Component, error) {
	if path == "" || path == "/" {
		return fs.root, nil
	}

	return fs.findComponentInDirectory(fs.root, path)
}

// Search finds all components matching the given name pattern.
// The pattern supports wildcards (* and ?).
// Returns a slice of matching component paths.
func (fs *FileSystemFacade) Search(pattern string) ([]string, error) {
	if pattern == "" {
		return nil, fmt.Errorf("search pattern cannot be empty")
	}

	// Convert wildcard pattern to regex
	regexPattern := "^" + strings.ReplaceAll(strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, ".*"), `\?`, ".") + "$"
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid search pattern: %w", err)
	}

	results := make([]string, 0)
	fs.searchInDirectory(fs.root, "", regex, &results)
	return results, nil
}

// GetTotalSize returns the total size of the entire file system in bytes.
func (fs *FileSystemFacade) GetTotalSize() int64 {
	return fs.root.Size()
}

// GetStatistics computes and returns statistics about the file system.
func (fs *FileSystemFacade) GetStatistics() *FileSystemStats {
	stats := &FileSystemStats{
		TotalFiles:       0,
		TotalDirectories: 0,
		TotalSize:        0,
		LargestFile:      "",
		LargestFileSize:  0,
		DeepestLevel:     0,
	}

	fs.collectStats(fs.root, "", 0, stats)
	stats.TotalSize = fs.root.Size()

	return stats
}

// PrintStructure displays the entire file system structure.
func (fs *FileSystemFacade) PrintStructure() {
	fmt.Printf("File System: %s\n", fs.name)
	fmt.Printf("Total Size: %.2f MB\n\n", float64(fs.GetTotalSize())/(1024*1024))
	fs.root.Print(0)
}

// CopyComponent creates a deep copy of a component and adds it to the target directory.
// Returns an error if the source or target paths are invalid.
func (fs *FileSystemFacade) CopyComponent(sourcePath, targetPath, newName string) error {
	if sourcePath == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	sourceComponent, err := fs.GetComponentByPath(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to find source component: %w", err)
	}

	targetDir, err := fs.getDirectoryByPath(targetPath)
	if err != nil {
		return fmt.Errorf("failed to locate target directory: %w", err)
	}

	copiedComponent, err := fs.copyComponentRecursive(sourceComponent, newName)
	if err != nil {
		return fmt.Errorf("failed to copy component: %w", err)
	}

	if err := targetDir.Add(copiedComponent); err != nil {
		return fmt.Errorf("failed to add copied component to target: %w", err)
	}

	return nil
}

// getDirectoryByPath navigates to a directory using the given path.
// Returns the root if path is empty or "/".
func (fs *FileSystemFacade) getDirectoryByPath(path string) (*Directory, error) {
	if path == "" || path == "/" {
		return fs.root, nil
	}

	component, err := fs.findComponentInDirectory(fs.root, path)
	if err != nil {
		return nil, err
	}

	dir, ok := component.(*Directory)
	if !ok {
		return nil, fmt.Errorf("path '%s' is not a directory", path)
	}

	return dir, nil
}

// splitPath splits a path into parent path and component name.
func (fs *FileSystemFacade) splitPath(path string) (string, string) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 {
		return "/", parts[0]
	}

	parentPath := strings.Join(parts[:len(parts)-1], "/")
	componentName := parts[len(parts)-1]

	return parentPath, componentName
}

// findComponentInDirectory recursively searches for a component by path.
func (fs *FileSystemFacade) findComponentInDirectory(dir *Directory, path string) (Component, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return dir, nil
	}

	parts := strings.Split(path, "/")
	currentName := parts[0]
	remainingPath := strings.Join(parts[1:], "/")

	for _, child := range dir.GetChildren() {
		if child.Name() == currentName {
			if remainingPath == "" {
				return child, nil
			}

			if childDir, ok := child.(*Directory); ok {
				return fs.findComponentInDirectory(childDir, remainingPath)
			}

			return nil, fmt.Errorf("path component '%s' is not a directory", currentName)
		}
	}

	return nil, fmt.Errorf("component '%s' not found in path '%s'", currentName, path)
}

// searchInDirectory recursively searches for components matching the pattern.
func (fs *FileSystemFacade) searchInDirectory(dir *Directory, currentPath string, pattern *regexp.Regexp, results *[]string) {
	for _, child := range dir.GetChildren() {
		childPath := filepath.Join(currentPath, child.Name())

		if pattern.MatchString(child.Name()) {
			*results = append(*results, childPath)
		}

		if childDir, ok := child.(*Directory); ok {
			fs.searchInDirectory(childDir, childPath, pattern, results)
		}
	}
}

// collectStats recursively collects statistics about the file system.
func (fs *FileSystemFacade) collectStats(component Component, currentPath string, level int, stats *FileSystemStats) {
	if level > stats.DeepestLevel {
		stats.DeepestLevel = level
	}

	if component.IsComposite() {
		stats.TotalDirectories++
		dir := component.(*Directory)
		for _, child := range dir.GetChildren() {
			childPath := filepath.Join(currentPath, child.Name())
			fs.collectStats(child, childPath, level+1, stats)
		}
	} else {
		stats.TotalFiles++
		if component.Size() > stats.LargestFileSize {
			stats.LargestFileSize = component.Size()
			stats.LargestFile = filepath.Join(currentPath, component.Name())
		}
	}
}

// copyComponentRecursive creates a deep copy of a component.
func (fs *FileSystemFacade) copyComponentRecursive(component Component, newName string) (Component, error) {
	name := newName
	if name == "" {
		name = component.Name()
	}

	if component.IsComposite() {
		newDir, err := NewDirectory(name)
		if err != nil {
			return nil, err
		}

		dir := component.(*Directory)
		for _, child := range dir.GetChildren() {
			copiedChild, err := fs.copyComponentRecursive(child, "")
			if err != nil {
				return nil, err
			}
			if err := newDir.Add(copiedChild); err != nil {
				return nil, err
			}
		}

		return newDir, nil
	}

	// Copy file
	file := component.(*File)
	newFile, err := NewFile(name, file.size, file.extension)
	if err != nil {
		return nil, err
	}

	return newFile, nil
}
