// Package structural provides implementations of structural design patterns.
// This file contains a production-grade implementation of the Decorator pattern
// using a data processing pipeline example with various processing decorators.
package structural

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

// DataProcessor defines the interface for processing data.
// It declares the core operation that both concrete components
// and decorators must implement.
type DataProcessor interface {
	// Process processes the input data and returns the processed result
	Process(data []byte) ([]byte, error)
	// GetDescription returns a description of what this processor does
	GetDescription() string
}

// BaseDataProcessor represents the concrete component in the decorator pattern.
// It provides the basic functionality that can be enhanced by decorators.
type BaseDataProcessor struct {
	name string
}

// NewBaseDataProcessor creates a new BaseDataProcessor instance.
// Returns an error if the name is empty.
func NewBaseDataProcessor(name string) (*BaseDataProcessor, error) {
	if name == "" {
		return nil, fmt.Errorf("processor name cannot be empty")
	}
	return &BaseDataProcessor{
		name: name,
	}, nil
}

// Process performs basic data processing (pass-through).
func (b *BaseDataProcessor) Process(data []byte) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("input data cannot be nil")
	}
	// Basic processor just returns a copy of the data
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// GetDescription returns the description of the base processor.
func (b *BaseDataProcessor) GetDescription() string {
	return fmt.Sprintf("Base Processor: %s", b.name)
}

// CompressionDecorator adds compression functionality to a data processor.
type CompressionDecorator struct {
	processor DataProcessor
	level     int
}

// NewCompressionDecorator creates a new CompressionDecorator.
// level should be between gzip.BestSpeed and gzip.BestCompression.
// Returns an error if the processor is nil or level is invalid.
func NewCompressionDecorator(processor DataProcessor, level int) (*CompressionDecorator, error) {
	if processor == nil {
		return nil, fmt.Errorf("processor cannot be nil")
	}
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		return nil, fmt.Errorf("invalid compression level: must be between %d and %d", gzip.BestSpeed, gzip.BestCompression)
	}
	return &CompressionDecorator{
		processor: processor,
		level:     level,
	}, nil
}

// Process compresses the data after processing by the wrapped processor.
func (c *CompressionDecorator) Process(data []byte) ([]byte, error) {
	// First, process with the wrapped processor
	processed, err := c.processor.Process(data)
	if err != nil {
		return nil, fmt.Errorf("compression decorator: wrapped processor failed: %w", err)
	}

	// Then compress the result
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, c.level)
	if err != nil {
		return nil, fmt.Errorf("compression decorator: failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(processed); err != nil {
		writer.Close()
		return nil, fmt.Errorf("compression decorator: failed to compress data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("compression decorator: failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// GetDescription returns the description including compression details.
func (c *CompressionDecorator) GetDescription() string {
	return fmt.Sprintf("%s -> Compression (level: %d)", c.processor.GetDescription(), c.level)
}

// EncryptionDecorator adds encryption functionality to a data processor.
type EncryptionDecorator struct {
	processor DataProcessor
	key       []byte
}

// NewEncryptionDecorator creates a new EncryptionDecorator.
// The key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256.
// Returns an error if the processor is nil or key length is invalid.
func NewEncryptionDecorator(processor DataProcessor, key []byte) (*EncryptionDecorator, error) {
	if processor == nil {
		return nil, fmt.Errorf("processor cannot be nil")
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}

	// Create a copy of the key to prevent external modifications
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	return &EncryptionDecorator{
		processor: processor,
		key:       keyCopy,
	}, nil
}

// Process encrypts the data after processing by the wrapped processor.
func (e *EncryptionDecorator) Process(data []byte) ([]byte, error) {
	// First, process with the wrapped processor
	processed, err := e.processor.Process(data)
	if err != nil {
		return nil, fmt.Errorf("encryption decorator: wrapped processor failed: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("encryption decorator: failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption decorator: failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encryption decorator: failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, processed, nil)
	return ciphertext, nil
}

// GetDescription returns the description including encryption details.
func (e *EncryptionDecorator) GetDescription() string {
	return fmt.Sprintf("%s -> Encryption (AES-%d)", e.processor.GetDescription(), len(e.key)*8)
}

// LoggingDecorator adds logging functionality to a data processor.
type LoggingDecorator struct {
	processor DataProcessor
	logger    Logger
}

// Logger defines the interface for logging operations.
type Logger interface {
	Log(message string)
}

// ConsoleLogger implements Logger interface for console output.
type ConsoleLogger struct {
	prefix string
}

// NewConsoleLogger creates a new ConsoleLogger with the given prefix.
func NewConsoleLogger(prefix string) *ConsoleLogger {
	return &ConsoleLogger{
		prefix: prefix,
	}
}

// Log writes a log message to the console with timestamp.
func (c *ConsoleLogger) Log(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if c.prefix != "" {
		fmt.Printf("[%s] [%s] %s\n", timestamp, c.prefix, message)
	} else {
		fmt.Printf("[%s] %s\n", timestamp, message)
	}
}

// NewLoggingDecorator creates a new LoggingDecorator.
// Returns an error if the processor or logger is nil.
func NewLoggingDecorator(processor DataProcessor, logger Logger) (*LoggingDecorator, error) {
	if processor == nil {
		return nil, fmt.Errorf("processor cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return &LoggingDecorator{
		processor: processor,
		logger:    logger,
	}, nil
}

// Process logs the processing operation and delegates to the wrapped processor.
func (l *LoggingDecorator) Process(data []byte) ([]byte, error) {
	l.logger.Log(fmt.Sprintf("Processing started - Input size: %d bytes", len(data)))
	startTime := time.Now()

	// Process with the wrapped processor
	result, err := l.processor.Process(data)

	duration := time.Since(startTime)

	if err != nil {
		l.logger.Log(fmt.Sprintf("Processing failed after %v: %v", duration, err))
		return nil, fmt.Errorf("logging decorator: wrapped processor failed: %w", err)
	}

	l.logger.Log(fmt.Sprintf("Processing completed in %v - Output size: %d bytes", duration, len(result)))
	return result, nil
}

// GetDescription returns the description including logging details.
func (l *LoggingDecorator) GetDescription() string {
	return fmt.Sprintf("%s -> Logging", l.processor.GetDescription())
}

// ValidationDecorator adds input validation to a data processor.
type ValidationDecorator struct {
	processor  DataProcessor
	minSize    int
	maxSize    int
	allowEmpty bool
}

// ValidationConfig holds configuration for the validation decorator.
type ValidationConfig struct {
	MinSize    int
	MaxSize    int
	AllowEmpty bool
}

// NewValidationDecorator creates a new ValidationDecorator with the given configuration.
// Returns an error if the processor is nil or configuration is invalid.
func NewValidationDecorator(processor DataProcessor, config ValidationConfig) (*ValidationDecorator, error) {
	if processor == nil {
		return nil, fmt.Errorf("processor cannot be nil")
	}
	if config.MinSize < 0 {
		return nil, fmt.Errorf("minimum size cannot be negative")
	}
	if config.MaxSize > 0 && config.MaxSize < config.MinSize {
		return nil, fmt.Errorf("maximum size cannot be less than minimum size")
	}

	return &ValidationDecorator{
		processor:  processor,
		minSize:    config.MinSize,
		maxSize:    config.MaxSize,
		allowEmpty: config.AllowEmpty,
	}, nil
}

// Process validates the input data before delegating to the wrapped processor.
func (v *ValidationDecorator) Process(data []byte) ([]byte, error) {
	// Validate input
	if data == nil {
		return nil, fmt.Errorf("validation decorator: input data cannot be nil")
	}

	if len(data) == 0 && !v.allowEmpty {
		return nil, fmt.Errorf("validation decorator: empty data not allowed")
	}

	if len(data) < v.minSize {
		return nil, fmt.Errorf("validation decorator: data size %d is below minimum %d", len(data), v.minSize)
	}

	if v.maxSize > 0 && len(data) > v.maxSize {
		return nil, fmt.Errorf("validation decorator: data size %d exceeds maximum %d", len(data), v.maxSize)
	}

	// Delegate to wrapped processor
	result, err := v.processor.Process(data)
	if err != nil {
		return nil, fmt.Errorf("validation decorator: wrapped processor failed: %w", err)
	}

	return result, nil
}

// GetDescription returns the description including validation details.
func (v *ValidationDecorator) GetDescription() string {
	return fmt.Sprintf("%s -> Validation (min: %d, max: %d, allowEmpty: %t)",
		v.processor.GetDescription(), v.minSize, v.maxSize, v.allowEmpty)
}
