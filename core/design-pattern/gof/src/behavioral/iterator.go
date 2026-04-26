// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Iterator pattern
// with support for bidirectional iteration, filtering, and thread-safety considerations.
package behavioral

import (
	"fmt"
	"sync"
)

// Iterator defines the interface for iterating over a collection.
// It provides methods to traverse elements sequentially without exposing
// the underlying collection structure.
type Iterator interface {
	// HasNext returns true if there are more elements to iterate
	HasNext() bool
	// Next returns the next element in the iteration
	// Returns an error if there are no more elements
	Next() (interface{}, error)
	// Reset resets the iterator to the beginning
	Reset()
}

// BidirectionalIterator extends Iterator to support backward traversal.
type BidirectionalIterator interface {
	Iterator
	// HasPrevious returns true if there are previous elements
	HasPrevious() bool
	// Previous returns the previous element in the iteration
	// Returns an error if there are no previous elements
	Previous() (interface{}, error)
}

// FilterableIterator provides iteration with filtering capability.
type FilterableIterator interface {
	Iterator
	// SetFilter sets a predicate function to filter elements
	SetFilter(predicate func(interface{}) bool)
}

// Aggregate defines the interface for collections that can create iterators.
type Aggregate interface {
	// CreateIterator creates a basic forward iterator
	CreateIterator() Iterator
	// CreateBidirectionalIterator creates an iterator that can traverse both directions
	CreateBidirectionalIterator() BidirectionalIterator
	// CreateFilterIterator creates an iterator with filtering capability
	CreateFilterIterator(predicate func(interface{}) bool) Iterator
}

// ConcreteCollection represents a thread-safe collection of items.
// It implements the Aggregate interface and provides various iterator types.
type ConcreteCollection struct {
	items []interface{}
	mu    sync.RWMutex
}

// NewConcreteCollection creates a new ConcreteCollection instance.
// Accepts optional initial capacity for performance optimization.
func NewConcreteCollection(capacity ...int) *ConcreteCollection {
	cap := 0
	if len(capacity) > 0 && capacity[0] > 0 {
		cap = capacity[0]
	}
	return &ConcreteCollection{
		items: make([]interface{}, 0, cap),
	}
}

// Add appends an item to the collection.
// Returns an error if the item is nil.
func (c *ConcreteCollection) Add(item interface{}) error {
	if item == nil {
		return fmt.Errorf("cannot add nil item to collection")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = append(c.items, item)
	return nil
}

// Remove removes an item at the specified index.
// Returns an error if the index is out of bounds.
func (c *ConcreteCollection) Remove(index int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if index < 0 || index >= len(c.items) {
		return fmt.Errorf("index %d out of bounds for collection of size %d", index, len(c.items))
	}
	c.items = append(c.items[:index], c.items[index+1:]...)
	return nil
}

// Get retrieves an item at the specified index.
// Returns an error if the index is out of bounds.
func (c *ConcreteCollection) Get(index int) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if index < 0 || index >= len(c.items) {
		return nil, fmt.Errorf("index %d out of bounds for collection of size %d", index, len(c.items))
	}
	return c.items[index], nil
}

// Size returns the number of items in the collection.
func (c *ConcreteCollection) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the collection.
func (c *ConcreteCollection) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make([]interface{}, 0)
}

// CreateIterator creates a basic forward iterator for the collection.
func (c *ConcreteCollection) CreateIterator() Iterator {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Create a snapshot of items to avoid concurrent modification issues
	itemsCopy := make([]interface{}, len(c.items))
	copy(itemsCopy, c.items)
	return &concreteIterator{
		items:   itemsCopy,
		current: 0,
	}
}

// CreateBidirectionalIterator creates an iterator supporting both forward and backward traversal.
func (c *ConcreteCollection) CreateBidirectionalIterator() BidirectionalIterator {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Create a snapshot of items to avoid concurrent modification issues
	itemsCopy := make([]interface{}, len(c.items))
	copy(itemsCopy, c.items)
	return &bidirectionalIterator{
		items:   itemsCopy,
		current: 0,
	}
}

// CreateFilterIterator creates an iterator that filters elements based on a predicate.
func (c *ConcreteCollection) CreateFilterIterator(predicate func(interface{}) bool) Iterator {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if predicate == nil {
		predicate = func(interface{}) bool { return true }
	}
	// Create a snapshot of items to avoid concurrent modification issues
	itemsCopy := make([]interface{}, len(c.items))
	copy(itemsCopy, c.items)
	return &filterIterator{
		items:     itemsCopy,
		current:   0,
		predicate: predicate,
	}
}

// concreteIterator implements the Iterator interface for forward traversal.
type concreteIterator struct {
	items   []interface{}
	current int
}

// HasNext returns true if there are more elements to iterate.
func (i *concreteIterator) HasNext() bool {
	return i.current < len(i.items)
}

// Next returns the next element in the iteration.
func (i *concreteIterator) Next() (interface{}, error) {
	if !i.HasNext() {
		return nil, fmt.Errorf("no more elements in iterator")
	}
	item := i.items[i.current]
	i.current++
	return item, nil
}

// Reset resets the iterator to the beginning.
func (i *concreteIterator) Reset() {
	i.current = 0
}

// bidirectionalIterator implements the BidirectionalIterator interface.
type bidirectionalIterator struct {
	items   []interface{}
	current int
}

// HasNext returns true if there are more elements to iterate forward.
func (i *bidirectionalIterator) HasNext() bool {
	return i.current < len(i.items)
}

// Next returns the next element in the forward iteration.
func (i *bidirectionalIterator) Next() (interface{}, error) {
	if !i.HasNext() {
		return nil, fmt.Errorf("no more elements in iterator")
	}
	item := i.items[i.current]
	i.current++
	return item, nil
}

// HasPrevious returns true if there are previous elements to iterate backward.
func (i *bidirectionalIterator) HasPrevious() bool {
	return i.current > 0
}

// Previous returns the previous element in the backward iteration.
func (i *bidirectionalIterator) Previous() (interface{}, error) {
	if !i.HasPrevious() {
		return nil, fmt.Errorf("no previous elements in iterator")
	}
	i.current--
	return i.items[i.current], nil
}

// Reset resets the iterator to the beginning.
func (i *bidirectionalIterator) Reset() {
	i.current = 0
}

// filterIterator implements the Iterator interface with filtering capability.
type filterIterator struct {
	items     []interface{}
	current   int
	predicate func(interface{}) bool
}

// HasNext returns true if there are more elements that match the filter.
func (i *filterIterator) HasNext() bool {
	for idx := i.current; idx < len(i.items); idx++ {
		if i.predicate(i.items[idx]) {
			return true
		}
	}
	return false
}

// Next returns the next element that matches the filter predicate.
func (i *filterIterator) Next() (interface{}, error) {
	for i.current < len(i.items) {
		item := i.items[i.current]
		i.current++
		if i.predicate(item) {
			return item, nil
		}
	}
	return nil, fmt.Errorf("no more elements matching the filter")
}

// Reset resets the iterator to the beginning.
func (i *filterIterator) Reset() {
	i.current = 0
}

// SetFilter updates the filter predicate.
func (i *filterIterator) SetFilter(predicate func(interface{}) bool) {
	if predicate == nil {
		i.predicate = func(interface{}) bool { return true }
	} else {
		i.predicate = predicate
	}
	i.Reset()
}
