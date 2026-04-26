package main

import (
	"fmt"
	"time"
)

// Interface Segregation Principle (ISP):
// Clients should not be forced to depend on interfaces they do not use.
// Split large interfaces into smaller, more specific ones.

// BAD EXAMPLE - Fat Interface (violates ISP)
// type Document interface {
// 	Print()
// 	Scan()
// 	Fax()
// 	Email()
// }

// GOOD EXAMPLE - Segregated Interfaces

// Printable interface for documents that can be printed
type Printable interface {
	Print() error
}

// Scannable interface for documents that can be scanned
type Scannable interface {
	Scan() ([]byte, error)
}

// Faxable interface for documents that can be faxed
type Faxable interface {
	Fax(recipient string) error
}

// Emailable interface for documents that can be emailed
type Emailable interface {
	Email(recipient string) error
}

// Shareable interface for documents that can be shared
type Shareable interface {
	Share(platform string) error
}

// Searchable interface for documents that can be searched
type Searchable interface {
	Search(keyword string) ([]string, error)
}

// Document base struct
type Document struct {
	ID        string
	Title     string
	Content   string
	CreatedAt time.Time
}

// PDFDocument - implements Printable, Emailable, Shareable
type PDFDocument struct {
	Document
	FileSize int64
}

func (p *PDFDocument) Print() error {
	fmt.Printf("[PDF] Printing document: %s (ID: %s)\n", p.Title, p.ID)
	fmt.Printf("       File size: %d bytes\n", p.FileSize)
	return nil
}

func (p *PDFDocument) Email(recipient string) error {
	fmt.Printf("[PDF] Emailing document '%s' to %s\n", p.Title, recipient)
	fmt.Printf("       Attachment size: %d bytes\n", p.FileSize)
	return nil
}

func (p *PDFDocument) Share(platform string) error {
	fmt.Printf("[PDF] Sharing document '%s' on %s\n", p.Title, platform)
	return nil
}

// TextDocument - implements Printable, Emailable, Searchable
type TextDocument struct {
	Document
	WordCount int
}

func (t *TextDocument) Print() error {
	fmt.Printf("[TEXT] Printing document: %s (ID: %s)\n", t.Title, t.ID)
	fmt.Printf("        Word count: %d\n", t.WordCount)
	return nil
}

func (t *TextDocument) Email(recipient string) error {
	fmt.Printf("[TEXT] Emailing document '%s' to %s\n", t.Title, recipient)
	return nil
}

func (t *TextDocument) Search(keyword string) ([]string, error) {
	fmt.Printf("[TEXT] Searching for '%s' in document '%s'\n", keyword, t.Title)
	results := []string{
		fmt.Sprintf("Found '%s' at line 5", keyword),
		fmt.Sprintf("Found '%s' at line 12", keyword),
	}
	return results, nil
}

// ScannedDocument - implements Scannable, Printable, Faxable
type ScannedDocument struct {
	Document
	Resolution int
	Pages      int
}

func (s *ScannedDocument) Scan() ([]byte, error) {
	fmt.Printf("[SCAN] Scanning document: %s at %d DPI\n", s.Title, s.Resolution)
	fmt.Printf("       Total pages: %d\n", s.Pages)
	return []byte("scanned-data"), nil
}

func (s *ScannedDocument) Print() error {
	fmt.Printf("[SCAN] Printing scanned document: %s\n", s.Title)
	fmt.Printf("       Pages: %d, Resolution: %d DPI\n", s.Pages, s.Resolution)
	return nil
}

func (s *ScannedDocument) Fax(recipient string) error {
	fmt.Printf("[SCAN] Faxing document '%s' to %s\n", s.Title, recipient)
	fmt.Printf("       Sending %d pages...\n", s.Pages)
	return nil
}

// ImageDocument - implements Printable, Emailable, Shareable
type ImageDocument struct {
	Document
	Format     string
	Dimensions string
}

func (i *ImageDocument) Print() error {
	fmt.Printf("[IMAGE] Printing image: %s (%s)\n", i.Title, i.Format)
	fmt.Printf("        Dimensions: %s\n", i.Dimensions)
	return nil
}

func (i *ImageDocument) Email(recipient string) error {
	fmt.Printf("[IMAGE] Emailing image '%s' to %s\n", i.Title, recipient)
	return nil
}

func (i *ImageDocument) Share(platform string) error {
	fmt.Printf("[IMAGE] Sharing image '%s' on %s\n", i.Title, platform)
	return nil
}

// DocumentProcessor - processes documents based on their capabilities
type DocumentProcessor struct {
	Name string
}

func (dp *DocumentProcessor) ProcessPrintable(doc Printable) {
	fmt.Printf("\n[Processor: %s] Processing printable document...\n", dp.Name)
	doc.Print()
}

func (dp *DocumentProcessor) ProcessEmailable(doc Emailable, recipient string) {
	fmt.Printf("\n[Processor: %s] Processing emailable document...\n", dp.Name)
	doc.Email(recipient)
}

func (dp *DocumentProcessor) ProcessScannable(doc Scannable) {
	fmt.Printf("\n[Processor: %s] Processing scannable document...\n", dp.Name)
	doc.Scan()
}

func (dp *DocumentProcessor) ProcessShareable(doc Shareable, platform string) {
	fmt.Printf("\n[Processor: %s] Processing shareable document...\n", dp.Name)
	doc.Share(platform)
}

func (dp *DocumentProcessor) ProcessSearchable(doc Searchable, keyword string) {
	fmt.Printf("\n[Processor: %s] Processing searchable document...\n", dp.Name)
	results, _ := doc.Search(keyword)
	for _, result := range results {
		fmt.Printf("        - %s\n", result)
	}
}

// TestInterfaceSegregation demonstrates the Interface Segregation Principle
func TestInterfaceSegregation() {
	fmt.Println("=== Interface Segregation Principle Demo ===")
	fmt.Println("ISP: Clients should not be forced to depend on interfaces they don't use\n")

	// Create various document types
	pdfDoc := &PDFDocument{
		Document: Document{
			ID:        "PDF-001",
			Title:     "Annual Report 2026",
			Content:   "Financial data and analysis...",
			CreatedAt: time.Now(),
		},
		FileSize: 2048576,
	}

	textDoc := &TextDocument{
		Document: Document{
			ID:        "TXT-001",
			Title:     "Meeting Notes",
			Content:   "Discussion points and action items...",
			CreatedAt: time.Now(),
		},
		WordCount: 450,
	}

	scannedDoc := &ScannedDocument{
		Document: Document{
			ID:        "SCAN-001",
			Title:     "Contract Agreement",
			Content:   "Legal document scanned from paper...",
			CreatedAt: time.Now(),
		},
		Resolution: 300,
		Pages:      15,
	}

	imageDoc := &ImageDocument{
		Document: Document{
			ID:        "IMG-001",
			Title:     "Product Photo",
			Content:   "High-resolution product image...",
			CreatedAt: time.Now(),
		},
		Format:     "PNG",
		Dimensions: "1920x1080",
	}

	processor := &DocumentProcessor{Name: "MainProcessor"}

	// Process PDF document - only uses interfaces it implements
	processor.ProcessPrintable(pdfDoc)
	processor.ProcessEmailable(pdfDoc, "finance@company.com")
	processor.ProcessShareable(pdfDoc, "SharePoint")

	// Process Text document - only uses interfaces it implements
	processor.ProcessPrintable(textDoc)
	processor.ProcessEmailable(textDoc, "team@company.com")
	processor.ProcessSearchable(textDoc, "action items")

	// Process Scanned document - only uses interfaces it implements
	processor.ProcessScannable(scannedDoc)
	processor.ProcessPrintable(scannedDoc)
	// Note: Can't call processor.ProcessEmailable(scannedDoc, ...) - doesn't implement Emailable

	// Process Image document
	processor.ProcessPrintable(imageDoc)
	processor.ProcessShareable(imageDoc, "Instagram")

	// Demonstrate batch operations
	fmt.Println("\n--- Batch Operations ---")
	printableDocuments := []Printable{pdfDoc, textDoc, scannedDoc, imageDoc}
	fmt.Println("\nPrinting all printable documents:")
	for _, doc := range printableDocuments {
		doc.Print()
		fmt.Println()
	}

	emailableDocuments := []Emailable{pdfDoc, textDoc, imageDoc}
	fmt.Println("\nEmailing all emailable documents:")
	for _, doc := range emailableDocuments {
		doc.Email("distribution@company.com")
	}

	fmt.Println("\n✓ Interface Segregation Principle Benefits:")
	fmt.Println("  • Each document type implements only the interfaces it needs")
	fmt.Println("  • No forced implementation of unused methods")
	fmt.Println("  • Better separation of concerns and flexibility")
	fmt.Println("  • Easier to maintain and extend")
	fmt.Println("  • Clients depend only on the methods they actually use")
}
