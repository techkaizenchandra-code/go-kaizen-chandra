// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Template Method pattern
// using a data processing pipeline example with different file format processors.
package behavioral

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// DataProcessor defines the interface for data processing operations.
// It declares the template method and primitive operations that subclasses must implement.
type DataProcessor interface {
	// Process is the template method that defines the algorithm skeleton
	Process() error
	// ReadData reads data from the source
	ReadData() ([]byte, error)
	// ValidateData validates the read data
	ValidateData(data []byte) error
	// ProcessData processes the validated data
	ProcessData(data []byte) (interface{}, error)
	// SaveData saves the processed data
	SaveData(processedData interface{}) error
	// PreProcess is a hook method called before processing (optional)
	PreProcess() error
	// PostProcess is a hook method called after processing (optional)
	PostProcess() error
}

// baseDataProcessor provides a skeletal implementation of the template method algorithm.
// Concrete processors embed this struct and override specific steps.
type baseDataProcessor struct {
	sourcePath      string
	destinationPath string
	startTime       time.Time
	endTime         time.Time
	processor       DataProcessor // Reference to the concrete implementation
}

// ProcessingResult holds the result of a data processing operation.
type ProcessingResult struct {
	RecordsProcessed int
	Duration         time.Duration
	Status           string
	Error            error
}

// Process implements the template method that defines the algorithm skeleton.
// This method cannot be overridden by subclasses and ensures the correct order of operations.
func (b *baseDataProcessor) Process() error {
	b.startTime = time.Now()

	// Hook: Pre-processing
	if err := b.processor.PreProcess(); err != nil {
		return fmt.Errorf("pre-processing failed: %w", err)
	}

	// Step 1: Read data
	data, err := b.processor.ReadData()
	if err != nil {
		return fmt.Errorf("read data failed: %w", err)
	}

	// Step 2: Validate data
	if err := b.processor.ValidateData(data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Step 3: Process data
	processedData, err := b.processor.ProcessData(data)
	if err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	// Step 4: Save data
	if err := b.processor.SaveData(processedData); err != nil {
		return fmt.Errorf("save data failed: %w", err)
	}

	// Hook: Post-processing
	if err := b.processor.PostProcess(); err != nil {
		return fmt.Errorf("post-processing failed: %w", err)
	}

	b.endTime = time.Now()
	return nil
}

// PreProcess is a hook method with a default empty implementation.
// Subclasses can override this to perform pre-processing tasks.
func (b *baseDataProcessor) PreProcess() error {
	// Default implementation does nothing
	return nil
}

// PostProcess is a hook method with a default empty implementation.
// Subclasses can override this to perform post-processing tasks.
func (b *baseDataProcessor) PostProcess() error {
	// Default implementation does nothing
	return nil
}

// GetProcessingDuration returns the duration of the processing operation.
func (b *baseDataProcessor) GetProcessingDuration() time.Duration {
	if b.endTime.IsZero() {
		return 0
	}
	return b.endTime.Sub(b.startTime)
}

// CSVDataProcessor is a concrete implementation for processing CSV files.
type CSVDataProcessor struct {
	baseDataProcessor
	delimiter     rune
	hasHeader     bool
	rawData       []byte
	parsedRecords [][]string
}

// NewCSVDataProcessor creates a new CSV data processor with validation.
func NewCSVDataProcessor(sourcePath, destinationPath string, delimiter rune, hasHeader bool) (*CSVDataProcessor, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}
	if destinationPath == "" {
		return nil, fmt.Errorf("destination path cannot be empty")
	}
	if delimiter == 0 {
		delimiter = ',' // Default delimiter
	}

	processor := &CSVDataProcessor{
		baseDataProcessor: baseDataProcessor{
			sourcePath:      sourcePath,
			destinationPath: destinationPath,
		},
		delimiter: delimiter,
		hasHeader: hasHeader,
	}
	processor.baseDataProcessor.processor = processor
	return processor, nil
}

// ReadData reads CSV data from the source.
func (c *CSVDataProcessor) ReadData() ([]byte, error) {
	// In production, this would read from an actual file
	// Simulated CSV data for demonstration
	csvData := fmt.Sprintf("id%cname%cage%cemail\n1%cJohn Doe%c30%cjohn@example.com\n2%cJane Smith%c25%cjane@example.com",
		c.delimiter, c.delimiter, c.delimiter,
		c.delimiter, c.delimiter, c.delimiter, c.delimiter,
		c.delimiter, c.delimiter, c.delimiter, c.delimiter)
	c.rawData = []byte(csvData)
	return c.rawData, nil
}

// ValidateData validates the CSV data structure.
func (c *CSVDataProcessor) ValidateData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("CSV data is empty")
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("CSV has no lines")
	}

	if c.hasHeader && len(lines) < 2 {
		return fmt.Errorf("CSV with header must have at least 2 lines")
	}

	return nil
}

// ProcessData processes the CSV data into structured records.
func (c *CSVDataProcessor) ProcessData(data []byte) (interface{}, error) {
	lines := strings.Split(string(data), "\n")
	c.parsedRecords = make([][]string, 0, len(lines))

	startIndex := 0
	if c.hasHeader {
		startIndex = 1
	}

	for i := startIndex; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		fields := strings.Split(line, string(c.delimiter))
		c.parsedRecords = append(c.parsedRecords, fields)
	}

	return c.parsedRecords, nil
}

// SaveData saves the processed CSV records.
func (c *CSVDataProcessor) SaveData(processedData interface{}) error {
	records, ok := processedData.([][]string)
	if !ok {
		return fmt.Errorf("invalid data type for CSV save")
	}

	// In production, this would write to an actual file
	fmt.Printf("Saving %d CSV records to %s\n", len(records), c.destinationPath)
	return nil
}

// PreProcess performs CSV-specific pre-processing.
func (c *CSVDataProcessor) PreProcess() error {
	fmt.Printf("Starting CSV processing from %s\n", c.sourcePath)
	return nil
}

// PostProcess performs CSV-specific post-processing.
func (c *CSVDataProcessor) PostProcess() error {
	fmt.Printf("Completed CSV processing. Records processed: %d\n", len(c.parsedRecords))
	return nil
}

// JSONDataProcessor is a concrete implementation for processing JSON files.
type JSONDataProcessor struct {
	baseDataProcessor
	prettyPrint   bool
	rawData       []byte
	parsedRecords []map[string]interface{}
}

// NewJSONDataProcessor creates a new JSON data processor with validation.
func NewJSONDataProcessor(sourcePath, destinationPath string, prettyPrint bool) (*JSONDataProcessor, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}
	if destinationPath == "" {
		return nil, fmt.Errorf("destination path cannot be empty")
	}

	processor := &JSONDataProcessor{
		baseDataProcessor: baseDataProcessor{
			sourcePath:      sourcePath,
			destinationPath: destinationPath,
		},
		prettyPrint: prettyPrint,
	}
	processor.baseDataProcessor.processor = processor
	return processor, nil
}

// ReadData reads JSON data from the source.
func (j *JSONDataProcessor) ReadData() ([]byte, error) {
	// In production, this would read from an actual file
	// Simulated JSON data for demonstration
	jsonData := `[
		{"id": 1, "name": "John Doe", "age": 30, "email": "john@example.com"},
		{"id": 2, "name": "Jane Smith", "age": 25, "email": "jane@example.com"}
	]`
	j.rawData = []byte(jsonData)
	return j.rawData, nil
}

// ValidateData validates the JSON data structure.
func (j *JSONDataProcessor) ValidateData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("JSON data is empty")
	}

	// Validate that it's valid JSON
	var test interface{}
	if err := json.Unmarshal(data, &test); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

// ProcessData processes the JSON data into structured records.
func (j *JSONDataProcessor) ProcessData(data []byte) (interface{}, error) {
	if err := json.Unmarshal(data, &j.parsedRecords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Apply business logic transformations here
	for i := range j.parsedRecords {
		// Add processing timestamp
		j.parsedRecords[i]["processed_at"] = time.Now().Format(time.RFC3339)
	}

	return j.parsedRecords, nil
}

// SaveData saves the processed JSON records.
func (j *JSONDataProcessor) SaveData(processedData interface{}) error {
	records, ok := processedData.([]map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data type for JSON save")
	}

	var output []byte
	var err error

	if j.prettyPrint {
		output, err = json.MarshalIndent(records, "", "  ")
	} else {
		output, err = json.Marshal(records)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// In production, this would write to an actual file
	fmt.Printf("Saving %d JSON records to %s (%d bytes)\n", len(records), j.destinationPath, len(output))
	return nil
}

// PreProcess performs JSON-specific pre-processing.
func (j *JSONDataProcessor) PreProcess() error {
	fmt.Printf("Starting JSON processing from %s\n", j.sourcePath)
	return nil
}

// PostProcess performs JSON-specific post-processing.
func (j *JSONDataProcessor) PostProcess() error {
	fmt.Printf("Completed JSON processing. Records processed: %d\n", len(j.parsedRecords))
	return nil
}

// XMLDataProcessor is a concrete implementation for processing XML files.
type XMLDataProcessor struct {
	baseDataProcessor
	indent        string
	rawData       []byte
	parsedRecords XMLRecords
}

// XMLRecords represents the root element for XML data.
type XMLRecords struct {
	XMLName xml.Name    `xml:"records"`
	Records []XMLRecord `xml:"record"`
}

// XMLRecord represents a single record in XML format.
type XMLRecord struct {
	ID    int    `xml:"id"`
	Name  string `xml:"name"`
	Age   int    `xml:"age"`
	Email string `xml:"email"`
}

// NewXMLDataProcessor creates a new XML data processor with validation.
func NewXMLDataProcessor(sourcePath, destinationPath string, indent string) (*XMLDataProcessor, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}
	if destinationPath == "" {
		return nil, fmt.Errorf("destination path cannot be empty")
	}

	processor := &XMLDataProcessor{
		baseDataProcessor: baseDataProcessor{
			sourcePath:      sourcePath,
			destinationPath: destinationPath,
		},
		indent: indent,
	}
	processor.baseDataProcessor.processor = processor
	return processor, nil
}

// ReadData reads XML data from the source.
func (x *XMLDataProcessor) ReadData() ([]byte, error) {
	// In production, this would read from an actual file
	// Simulated XML data for demonstration
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<records>
	<record>
		<id>1</id>
		<name>John Doe</name>
		<age>30</age>
		<email>john@example.com</email>
	</record>
	<record>
		<id>2</id>
		<name>Jane Smith</name>
		<age>25</age>
		<email>jane@example.com</email>
	</record>
</records>`
	x.rawData = []byte(xmlData)
	return x.rawData, nil
}

// ValidateData validates the XML data structure.
func (x *XMLDataProcessor) ValidateData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("XML data is empty")
	}

	// Validate that it's valid XML
	var test XMLRecords
	if err := xml.Unmarshal(data, &test); err != nil {
		return fmt.Errorf("invalid XML format: %w", err)
	}

	if len(test.Records) == 0 {
		return fmt.Errorf("XML contains no records")
	}

	return nil
}

// ProcessData processes the XML data into structured records.
func (x *XMLDataProcessor) ProcessData(data []byte) (interface{}, error) {
	if err := xml.Unmarshal(data, &x.parsedRecords); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// Apply business logic transformations here
	// For example, normalize email addresses to lowercase
	for i := range x.parsedRecords.Records {
		x.parsedRecords.Records[i].Email = strings.ToLower(x.parsedRecords.Records[i].Email)
	}

	return &x.parsedRecords, nil
}

// SaveData saves the processed XML records.
func (x *XMLDataProcessor) SaveData(processedData interface{}) error {
	records, ok := processedData.(*XMLRecords)
	if !ok {
		return fmt.Errorf("invalid data type for XML save")
	}

	var output []byte
	var err error

	if x.indent != "" {
		output, err = xml.MarshalIndent(records, "", x.indent)
	} else {
		output, err = xml.Marshal(records)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}

	// In production, this would write to an actual file
	fmt.Printf("Saving %d XML records to %s (%d bytes)\n", len(records.Records), x.destinationPath, len(output))
	return nil
}

// PreProcess performs XML-specific pre-processing.
func (x *XMLDataProcessor) PreProcess() error {
	fmt.Printf("Starting XML processing from %s\n", x.sourcePath)
	return nil
}

// PostProcess performs XML-specific post-processing.
func (x *XMLDataProcessor) PostProcess() error {
	fmt.Printf("Completed XML processing. Records processed: %d\n", len(x.parsedRecords.Records))
	return nil
}
