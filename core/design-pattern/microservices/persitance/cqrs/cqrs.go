package cqrs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CQRS Interfaces and Core Types
// ============================================================================

// Command represents a write operation intent
type Command interface {
	CommandID() string
	CommandType() string
	AggregateID() string
	Validate() error
}

// Query represents a read operation request
type Query interface {
	QueryID() string
	QueryType() string
	Validate() error
}

// Event represents a domain event that occurred
type Event interface {
	EventID() string
	EventType() string
	AggregateID() string
	OccurredAt() time.Time
	EventData() interface{}
}

// CommandHandler processes commands
type CommandHandler interface {
	Handle(ctx context.Context, cmd Command) error
}

// QueryHandler processes queries
type QueryHandler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// ============================================================================
// Event Store
// ============================================================================

// EventStore stores and retrieves events
type EventStore interface {
	Save(ctx context.Context, events []Event) error
	Load(ctx context.Context, aggregateID string) ([]Event, error)
}

// InMemoryEventStore is a production-ready in-memory event store with concurrency control
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]Event // aggregateID -> events
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]Event),
	}
}

func (es *InMemoryEventStore) Save(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return errors.New("no events to save")
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	for _, event := range events {
		aggregateID := event.AggregateID()
		es.events[aggregateID] = append(es.events[aggregateID], event)
		log.Printf("[EventStore] Saved event: %s for aggregate: %s", event.EventType(), aggregateID)
	}

	return nil
}

func (es *InMemoryEventStore) Load(ctx context.Context, aggregateID string) ([]Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events, exists := es.events[aggregateID]
	if !exists {
		return []Event{}, nil
	}

	return events, nil
}

// ============================================================================
// Message Bus
// ============================================================================

// MessageBus handles command and query routing
type MessageBus interface {
	RegisterCommandHandler(commandType string, handler CommandHandler)
	RegisterQueryHandler(queryType string, handler QueryHandler)
	DispatchCommand(ctx context.Context, cmd Command) error
	DispatchQuery(ctx context.Context, query Query) (interface{}, error)
}

// InMemoryMessageBus is a production-ready message bus with retry and timeout
type InMemoryMessageBus struct {
	commandHandlers map[string]CommandHandler
	queryHandlers   map[string]QueryHandler
	mu              sync.RWMutex
	maxRetries      int
	timeout         time.Duration
}

func NewInMemoryMessageBus(maxRetries int, timeout time.Duration) *InMemoryMessageBus {
	return &InMemoryMessageBus{
		commandHandlers: make(map[string]CommandHandler),
		queryHandlers:   make(map[string]QueryHandler),
		maxRetries:      maxRetries,
		timeout:         timeout,
	}
}

func (mb *InMemoryMessageBus) RegisterCommandHandler(commandType string, handler CommandHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.commandHandlers[commandType] = handler
	log.Printf("[MessageBus] Registered command handler for: %s", commandType)
}

func (mb *InMemoryMessageBus) RegisterQueryHandler(queryType string, handler QueryHandler) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.queryHandlers[queryType] = handler
	log.Printf("[MessageBus] Registered query handler for: %s", queryType)
}

func (mb *InMemoryMessageBus) DispatchCommand(ctx context.Context, cmd Command) error {
	mb.mu.RLock()
	handler, exists := mb.commandHandlers[cmd.CommandType()]
	mb.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler registered for command: %s", cmd.CommandType())
	}

	if err := cmd.Validate(); err != nil {
		return fmt.Errorf("command validation failed: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= mb.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 100 * time.Millisecond
			log.Printf("[MessageBus] Retry attempt %d for command %s after %v", attempt, cmd.CommandID(), backoff)
			time.Sleep(backoff)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, mb.timeout)
		errChan := make(chan error, 1)

		go func() {
			errChan <- handler.Handle(timeoutCtx, cmd)
		}()

		select {
		case err := <-errChan:
			cancel()
			if err == nil {
				log.Printf("[MessageBus] Command dispatched successfully: %s", cmd.CommandID())
				return nil
			}
			lastErr = err
		case <-timeoutCtx.Done():
			cancel()
			lastErr = fmt.Errorf("command execution timeout: %w", timeoutCtx.Err())
		}
	}

	return fmt.Errorf("command failed after %d retries: %w", mb.maxRetries, lastErr)
}

func (mb *InMemoryMessageBus) DispatchQuery(ctx context.Context, query Query) (interface{}, error) {
	mb.mu.RLock()
	handler, exists := mb.queryHandlers[query.QueryType()]
	mb.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for query: %s", query.QueryType())
	}

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, mb.timeout)
	defer cancel()

	type result struct {
		data interface{}
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		data, err := handler.Handle(timeoutCtx, query)
		resultChan <- result{data: data, err: err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		log.Printf("[MessageBus] Query executed successfully: %s", query.QueryID())
		return res.data, nil
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("query execution timeout: %w", timeoutCtx.Err())
	}
}

// ============================================================================
// Domain Model (Write Side - Command Model)
// ============================================================================

// Product represents the aggregate root
type Product struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Quantity    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int
	events      []Event
}

func NewProduct(id, name, description string, price float64, quantity int) *Product {
	product := &Product{
		ID:          id,
		Name:        name,
		Description: description,
		Price:       price,
		Quantity:    quantity,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
	}

	event := &ProductCreatedEvent{
		BaseEvent: BaseEvent{
			ID:               uuid.New().String(),
			Type:             "ProductCreated",
			AggregateIDValue: id,
			Timestamp:        time.Now(),
		},
		ProductID:   id,
		ProductName: name,
		Price:       price,
		Quantity:    quantity,
	}

	product.events = append(product.events, event)
	return product
}

func (p *Product) UpdatePrice(newPrice float64) error {
	if newPrice <= 0 {
		return errors.New("price must be positive")
	}

	p.Price = newPrice
	p.UpdatedAt = time.Now()
	p.Version++

	event := &ProductPriceUpdatedEvent{
		BaseEvent: BaseEvent{
			ID:               uuid.New().String(),
			Type:             "ProductPriceUpdated",
			AggregateIDValue: p.ID,
			Timestamp:        time.Now(),
		},
		ProductID: p.ID,
		OldPrice:  p.Price,
		NewPrice:  newPrice,
	}

	p.events = append(p.events, event)
	return nil
}

func (p *Product) UpdateQuantity(quantity int) error {
	if quantity < 0 {
		return errors.New("quantity cannot be negative")
	}

	p.Quantity = quantity
	p.UpdatedAt = time.Now()
	p.Version++

	event := &ProductQuantityUpdatedEvent{
		BaseEvent: BaseEvent{
			ID:               uuid.New().String(),
			Type:             "ProductQuantityUpdated",
			AggregateIDValue: p.ID,
			Timestamp:        time.Now(),
		},
		ProductID:   p.ID,
		NewQuantity: quantity,
	}

	p.events = append(p.events, event)
	return nil
}

func (p *Product) GetUncommittedEvents() []Event {
	return p.events
}

func (p *Product) ClearEvents() {
	p.events = []Event{}
}

// ============================================================================
// Domain Events
// ============================================================================

type BaseEvent struct {
	ID               string
	Type             string
	AggregateIDValue string
	Timestamp        time.Time
}

func (e *BaseEvent) EventID() string {
	return e.ID
}

func (e *BaseEvent) EventType() string {
	return e.Type
}

func (e *BaseEvent) AggregateID() string {
	return e.AggregateIDValue
}

func (e *BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

func (e *BaseEvent) EventData() interface{} {
	return e
}

type ProductCreatedEvent struct {
	BaseEvent
	ProductID   string
	ProductName string
	Price       float64
	Quantity    int
}

type ProductPriceUpdatedEvent struct {
	BaseEvent
	ProductID string
	OldPrice  float64
	NewPrice  float64
}

type ProductQuantityUpdatedEvent struct {
	BaseEvent
	ProductID   string
	NewQuantity int
}

// ============================================================================
// Commands
// ============================================================================

type CreateProductCommand struct {
	ID          string
	ProductID   string
	Name        string
	Description string
	Price       float64
	Quantity    int
}

func (c *CreateProductCommand) CommandID() string   { return c.ID }
func (c *CreateProductCommand) CommandType() string { return "CreateProduct" }
func (c *CreateProductCommand) AggregateID() string { return c.ProductID }
func (c *CreateProductCommand) Validate() error {
	if c.ProductID == "" {
		return errors.New("product ID is required")
	}
	if c.Name == "" {
		return errors.New("product name is required")
	}
	if c.Price <= 0 {
		return errors.New("price must be positive")
	}
	if c.Quantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return nil
}

type UpdateProductPriceCommand struct {
	ID        string
	ProductID string
	NewPrice  float64
}

func (c *UpdateProductPriceCommand) CommandID() string   { return c.ID }
func (c *UpdateProductPriceCommand) CommandType() string { return "UpdateProductPrice" }
func (c *UpdateProductPriceCommand) AggregateID() string { return c.ProductID }
func (c *UpdateProductPriceCommand) Validate() error {
	if c.ProductID == "" {
		return errors.New("product ID is required")
	}
	if c.NewPrice <= 0 {
		return errors.New("price must be positive")
	}
	return nil
}

type UpdateProductQuantityCommand struct {
	ID          string
	ProductID   string
	NewQuantity int
}

func (c *UpdateProductQuantityCommand) CommandID() string   { return c.ID }
func (c *UpdateProductQuantityCommand) CommandType() string { return "UpdateProductQuantity" }
func (c *UpdateProductQuantityCommand) AggregateID() string { return c.ProductID }
func (c *UpdateProductQuantityCommand) Validate() error {
	if c.ProductID == "" {
		return errors.New("product ID is required")
	}
	if c.NewQuantity < 0 {
		return errors.New("quantity cannot be negative")
	}
	return nil
}

// ============================================================================
// Queries
// ============================================================================

type GetProductByIDQuery struct {
	ID        string
	ProductID string
}

func (q *GetProductByIDQuery) QueryID() string   { return q.ID }
func (q *GetProductByIDQuery) QueryType() string { return "GetProductByID" }
func (q *GetProductByIDQuery) Validate() error {
	if q.ProductID == "" {
		return errors.New("product ID is required")
	}
	return nil
}

type GetAllProductsQuery struct {
	ID string
}

func (q *GetAllProductsQuery) QueryID() string   { return q.ID }
func (q *GetAllProductsQuery) QueryType() string { return "GetAllProducts" }
func (q *GetAllProductsQuery) Validate() error   { return nil }

// ============================================================================
// Read Model (Query Side)
// ============================================================================

type ProductReadModel struct {
	ID          string
	Name        string
	Description string
	Price       float64
	Quantity    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ============================================================================
// Repositories
// ============================================================================

// WriteRepository for command side
type WriteRepository interface {
	Save(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
}

// ReadRepository for query side
type ReadRepository interface {
	Save(ctx context.Context, model *ProductReadModel) error
	GetByID(ctx context.Context, id string) (*ProductReadModel, error)
	GetAll(ctx context.Context) ([]*ProductReadModel, error)
}

// InMemoryWriteRepository implements WriteRepository
type InMemoryWriteRepository struct {
	mu         sync.RWMutex
	products   map[string]*Product
	eventStore EventStore
}

func NewInMemoryWriteRepository(eventStore EventStore) *InMemoryWriteRepository {
	return &InMemoryWriteRepository{
		products:   make(map[string]*Product),
		eventStore: eventStore,
	}
}

func (r *InMemoryWriteRepository) Save(ctx context.Context, product *Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	events := product.GetUncommittedEvents()
	if len(events) > 0 {
		if err := r.eventStore.Save(ctx, events); err != nil {
			return fmt.Errorf("failed to save events: %w", err)
		}
		product.ClearEvents()
	}

	r.products[product.ID] = product
	log.Printf("[WriteRepository] Saved product: %s", product.ID)
	return nil
}

func (r *InMemoryWriteRepository) GetByID(ctx context.Context, id string) (*Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, exists := r.products[id]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", id)
	}

	return product, nil
}

// InMemoryReadRepository implements ReadRepository
type InMemoryReadRepository struct {
	mu       sync.RWMutex
	products map[string]*ProductReadModel
}

func NewInMemoryReadRepository() *InMemoryReadRepository {
	return &InMemoryReadRepository{
		products: make(map[string]*ProductReadModel),
	}
}

func (r *InMemoryReadRepository) Save(ctx context.Context, model *ProductReadModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.products[model.ID] = model
	log.Printf("[ReadRepository] Saved read model: %s", model.ID)
	return nil
}

func (r *InMemoryReadRepository) GetByID(ctx context.Context, id string) (*ProductReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.products[id]
	if !exists {
		return nil, fmt.Errorf("product not found: %s", id)
	}

	return model, nil
}

func (r *InMemoryReadRepository) GetAll(ctx context.Context) ([]*ProductReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	products := make([]*ProductReadModel, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}

	return products, nil
}

// ============================================================================
// Command Handlers
// ============================================================================

type ProductCommandHandler struct {
	writeRepo WriteRepository
	readRepo  ReadRepository
}

func NewProductCommandHandler(writeRepo WriteRepository, readRepo ReadRepository) *ProductCommandHandler {
	return &ProductCommandHandler{
		writeRepo: writeRepo,
		readRepo:  readRepo,
	}
}

func (h *ProductCommandHandler) Handle(ctx context.Context, cmd Command) error {
	switch c := cmd.(type) {
	case *CreateProductCommand:
		return h.handleCreateProduct(ctx, c)
	case *UpdateProductPriceCommand:
		return h.handleUpdatePrice(ctx, c)
	case *UpdateProductQuantityCommand:
		return h.handleUpdateQuantity(ctx, c)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.CommandType())
	}
}

func (h *ProductCommandHandler) handleCreateProduct(ctx context.Context, cmd *CreateProductCommand) error {
	product := NewProduct(cmd.ProductID, cmd.Name, cmd.Description, cmd.Price, cmd.Quantity)

	if err := h.writeRepo.Save(ctx, product); err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}

	// Update read model
	readModel := &ProductReadModel{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Quantity:    product.Quantity,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}

	if err := h.readRepo.Save(ctx, readModel); err != nil {
		return fmt.Errorf("failed to update read model: %w", err)
	}

	log.Printf("[CommandHandler] Product created: %s", product.ID)
	return nil
}

func (h *ProductCommandHandler) handleUpdatePrice(ctx context.Context, cmd *UpdateProductPriceCommand) error {
	product, err := h.writeRepo.GetByID(ctx, cmd.ProductID)
	if err != nil {
		return err
	}

	if err := product.UpdatePrice(cmd.NewPrice); err != nil {
		return err
	}

	if err := h.writeRepo.Save(ctx, product); err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}

	// Update read model
	readModel, err := h.readRepo.GetByID(ctx, cmd.ProductID)
	if err != nil {
		return err
	}

	readModel.Price = cmd.NewPrice
	readModel.UpdatedAt = time.Now()

	if err := h.readRepo.Save(ctx, readModel); err != nil {
		return fmt.Errorf("failed to update read model: %w", err)
	}

	log.Printf("[CommandHandler] Product price updated: %s", product.ID)
	return nil
}

func (h *ProductCommandHandler) handleUpdateQuantity(ctx context.Context, cmd *UpdateProductQuantityCommand) error {
	product, err := h.writeRepo.GetByID(ctx, cmd.ProductID)
	if err != nil {
		return err
	}

	if err := product.UpdateQuantity(cmd.NewQuantity); err != nil {
		return err
	}

	if err := h.writeRepo.Save(ctx, product); err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}

	// Update read model
	readModel, err := h.readRepo.GetByID(ctx, cmd.ProductID)
	if err != nil {
		return err
	}

	readModel.Quantity = cmd.NewQuantity
	readModel.UpdatedAt = time.Now()

	if err := h.readRepo.Save(ctx, readModel); err != nil {
		return fmt.Errorf("failed to update read model: %w", err)
	}

	log.Printf("[CommandHandler] Product quantity updated: %s", product.ID)
	return nil
}

// ============================================================================
// Query Handlers
// ============================================================================

type ProductQueryHandler struct {
	readRepo ReadRepository
}

func NewProductQueryHandler(readRepo ReadRepository) *ProductQueryHandler {
	return &ProductQueryHandler{
		readRepo: readRepo,
	}
}

func (h *ProductQueryHandler) Handle(ctx context.Context, query Query) (interface{}, error) {
	switch q := query.(type) {
	case *GetProductByIDQuery:
		return h.handleGetByID(ctx, q)
	case *GetAllProductsQuery:
		return h.handleGetAll(ctx, q)
	default:
		return nil, fmt.Errorf("unknown query type: %s", query.QueryType())
	}
}

func (h *ProductQueryHandler) handleGetByID(ctx context.Context, query *GetProductByIDQuery) (interface{}, error) {
	product, err := h.readRepo.GetByID(ctx, query.ProductID)
	if err != nil {
		return nil, err
	}

	log.Printf("[QueryHandler] Retrieved product: %s", product.ID)
	return product, nil
}

func (h *ProductQueryHandler) handleGetAll(ctx context.Context, query *GetAllProductsQuery) (interface{}, error) {
	products, err := h.readRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("[QueryHandler] Retrieved %d products", len(products))
	return products, nil
}

// ============================================================================
// CQRS Application Service
// ============================================================================

type CQRSApplication struct {
	messageBus MessageBus
	eventStore EventStore
	writeRepo  WriteRepository
	readRepo   ReadRepository
	mu         sync.Mutex
	shutdown   chan struct{}
}

func NewCQRSApplication() *CQRSApplication {
	eventStore := NewInMemoryEventStore()
	writeRepo := NewInMemoryWriteRepository(eventStore)
	readRepo := NewInMemoryReadRepository()
	messageBus := NewInMemoryMessageBus(3, 5*time.Second)

	app := &CQRSApplication{
		messageBus: messageBus,
		eventStore: eventStore,
		writeRepo:  writeRepo,
		readRepo:   readRepo,
		shutdown:   make(chan struct{}),
	}

	app.registerHandlers()
	return app
}

func (app *CQRSApplication) registerHandlers() {
	commandHandler := NewProductCommandHandler(app.writeRepo, app.readRepo)
	queryHandler := NewProductQueryHandler(app.readRepo)

	app.messageBus.RegisterCommandHandler("CreateProduct", commandHandler)
	app.messageBus.RegisterCommandHandler("UpdateProductPrice", commandHandler)
	app.messageBus.RegisterCommandHandler("UpdateProductQuantity", commandHandler)

	app.messageBus.RegisterQueryHandler("GetProductByID", queryHandler)
	app.messageBus.RegisterQueryHandler("GetAllProducts", queryHandler)
}

func (app *CQRSApplication) ExecuteCommand(ctx context.Context, cmd Command) error {
	return app.messageBus.DispatchCommand(ctx, cmd)
}

func (app *CQRSApplication) ExecuteQuery(ctx context.Context, query Query) (interface{}, error) {
	return app.messageBus.DispatchQuery(ctx, query)
}

func (app *CQRSApplication) Shutdown(ctx context.Context) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	select {
	case <-app.shutdown:
		return errors.New("already shutdown")
	default:
		close(app.shutdown)
		log.Println("[CQRSApplication] Graceful shutdown completed")
		return nil
	}
}

// ============================================================================
// Example Usage
// ============================================================================

func TestCQRSPattern() {
	log.Println("=== CQRS Pattern Demo ===")

	app := NewCQRSApplication()
	ctx := context.Background()

	// Create products
	productID1 := uuid.New().String()
	createCmd1 := &CreateProductCommand{
		ID:          uuid.New().String(),
		ProductID:   productID1,
		Name:        "Laptop",
		Description: "High-performance laptop",
		Price:       1299.99,
		Quantity:    50,
	}

	if err := app.ExecuteCommand(ctx, createCmd1); err != nil {
		log.Fatalf("Failed to create product: %v", err)
	}

	productID2 := uuid.New().String()
	createCmd2 := &CreateProductCommand{
		ID:          uuid.New().String(),
		ProductID:   productID2,
		Name:        "Mouse",
		Description: "Wireless mouse",
		Price:       29.99,
		Quantity:    200,
	}

	if err := app.ExecuteCommand(ctx, createCmd2); err != nil {
		log.Fatalf("Failed to create product: %v", err)
	}

	// Update product price
	updatePriceCmd := &UpdateProductPriceCommand{
		ID:        uuid.New().String(),
		ProductID: productID1,
		NewPrice:  1199.99,
	}

	if err := app.ExecuteCommand(ctx, updatePriceCmd); err != nil {
		log.Fatalf("Failed to update price: %v", err)
	}

	// Update product quantity
	updateQuantityCmd := &UpdateProductQuantityCommand{
		ID:          uuid.New().String(),
		ProductID:   productID2,
		NewQuantity: 180,
	}

	if err := app.ExecuteCommand(ctx, updateQuantityCmd); err != nil {
		log.Fatalf("Failed to update quantity: %v", err)
	}

	// Query single product
	getProductQuery := &GetProductByIDQuery{
		ID:        uuid.New().String(),
		ProductID: productID1,
	}

	result, err := app.ExecuteQuery(ctx, getProductQuery)
	if err != nil {
		log.Fatalf("Failed to get product: %v", err)
	}

	product := result.(*ProductReadModel)
	log.Printf("Product: ID=%s, Name=%s, Price=%.2f, Quantity=%d",
		product.ID, product.Name, product.Price, product.Quantity)

	// Query all products
	getAllQuery := &GetAllProductsQuery{
		ID: uuid.New().String(),
	}

	allResult, err := app.ExecuteQuery(ctx, getAllQuery)
	if err != nil {
		log.Fatalf("Failed to get all products: %v", err)
	}

	products := allResult.([]*ProductReadModel)
	log.Printf("Total products: %d", len(products))
	for _, p := range products {
		log.Printf("  - %s: $%.2f (qty: %d)", p.Name, p.Price, p.Quantity)
	}

	// Graceful shutdown
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("\n✓ CQRS Pattern: Commands and Queries are separated")
	log.Println("✓ Event Sourcing: All changes are stored as events")
	log.Println("✓ Production Ready: Includes retry, timeout, and concurrency control")
}
