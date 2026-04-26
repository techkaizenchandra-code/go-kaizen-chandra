// Package behavioral provides implementations of behavioral design patterns.
// This file contains a production-grade implementation of the Mediator pattern
// using a chat room system where users (colleagues) communicate through a central
// mediator, promoting loose coupling and centralized control of interactions.
package behavioral

import (
	"fmt"
	"sync"
	"time"
)

// Mediator defines the interface for communication between colleagues.
// It encapsulates how a set of objects interact and promotes loose coupling
// by keeping objects from referring to each other explicitly.
type Mediator interface {
	// RegisterUser adds a user to the chat room
	RegisterUser(user Colleague) error
	// UnregisterUser removes a user from the chat room
	UnregisterUser(username string) error
	// SendMessage broadcasts a message from one user to all other users
	SendMessage(message string, from Colleague) error
	// SendPrivateMessage sends a message from one user to a specific user
	SendPrivateMessage(message string, from Colleague, toUsername string) error
	// GetOnlineUsers returns a list of currently online users
	GetOnlineUsers() []string
}

// Colleague defines the interface for users in the chat system.
// Colleagues communicate with each other through the mediator.
type Colleague interface {
	// SetMediator sets the mediator for this colleague
	SetMediator(mediator Mediator)
	// Send sends a message through the mediator
	Send(message string) error
	// SendPrivate sends a private message to a specific user
	SendPrivate(message string, toUsername string) error
	// Receive handles receiving a message
	Receive(message string, from string)
	// GetName returns the name of the colleague
	GetName() string
	// IsOnline returns whether the user is currently online
	IsOnline() bool
	// SetOnline sets the online status of the user
	SetOnline(status bool)
}

// ChatRoom is a concrete mediator that manages communication between users.
// It maintains a registry of users and routes messages between them.
type ChatRoom struct {
	name           string
	users          map[string]Colleague
	messageHistory []ChatMessage
	mu             sync.RWMutex
	maxHistory     int
}

// ChatMessage represents a message in the chat room with metadata.
type ChatMessage struct {
	From      string
	To        string // Empty for broadcast messages
	Content   string
	Timestamp time.Time
	IsPrivate bool
}

// User is a concrete colleague that represents a chat room participant.
// Users send and receive messages through the mediator.
type User struct {
	name     string
	mediator Mediator
	online   bool
	mu       sync.RWMutex
}

// NewChatRoom creates a new ChatRoom instance with validation.
// Returns an error if the name is empty.
func NewChatRoom(name string, maxHistory int) (*ChatRoom, error) {
	if name == "" {
		return nil, fmt.Errorf("chat room name cannot be empty")
	}
	if maxHistory < 0 {
		maxHistory = 100 // Default history size
	}
	return &ChatRoom{
		name:           name,
		users:          make(map[string]Colleague),
		messageHistory: make([]ChatMessage, 0),
		maxHistory:     maxHistory,
	}, nil
}

// NewUser creates a new User instance with validation.
// Returns an error if the username is empty.
func NewUser(name string) (*User, error) {
	if name == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	return &User{
		name:   name,
		online: true,
	}, nil
}

// RegisterUser adds a user to the chat room.
// Returns an error if the user is nil or already registered.
func (c *ChatRoom) RegisterUser(user Colleague) error {
	if user == nil {
		return fmt.Errorf("cannot register nil user")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	username := user.GetName()
	if _, exists := c.users[username]; exists {
		return fmt.Errorf("user '%s' is already registered in the chat room", username)
	}

	c.users[username] = user
	user.SetMediator(c)

	// Notify other users
	joinMessage := fmt.Sprintf("📢 %s has joined the chat room", username)
	for _, u := range c.users {
		if u.GetName() != username && u.IsOnline() {
			u.Receive(joinMessage, "System")
		}
	}

	return nil
}

// UnregisterUser removes a user from the chat room.
// Returns an error if the user is not found.
func (c *ChatRoom) UnregisterUser(username string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.users[username]; !exists {
		return fmt.Errorf("user '%s' not found in the chat room", username)
	}

	delete(c.users, username)

	// Notify other users
	leaveMessage := fmt.Sprintf("📢 %s has left the chat room", username)
	for _, u := range c.users {
		if u.IsOnline() {
			u.Receive(leaveMessage, "System")
		}
	}

	return nil
}

// SendMessage broadcasts a message from one user to all other online users.
// Returns an error if the sender is not registered or offline.
func (c *ChatRoom) SendMessage(message string, from Colleague) error {
	if from == nil {
		return fmt.Errorf("sender cannot be nil")
	}

	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	fromUsername := from.GetName()
	if _, exists := c.users[fromUsername]; !exists {
		return fmt.Errorf("sender '%s' is not registered in the chat room", fromUsername)
	}

	if !from.IsOnline() {
		return fmt.Errorf("sender '%s' is offline", fromUsername)
	}

	// Store message in history
	chatMsg := ChatMessage{
		From:      fromUsername,
		To:        "",
		Content:   message,
		Timestamp: time.Now(),
		IsPrivate: false,
	}
	c.addToHistory(chatMsg)

	// Broadcast to all other online users
	for username, user := range c.users {
		if username != fromUsername && user.IsOnline() {
			user.Receive(message, fromUsername)
		}
	}

	return nil
}

// SendPrivateMessage sends a message from one user to a specific user.
// Returns an error if sender or recipient is not found or offline.
func (c *ChatRoom) SendPrivateMessage(message string, from Colleague, toUsername string) error {
	if from == nil {
		return fmt.Errorf("sender cannot be nil")
	}

	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	if toUsername == "" {
		return fmt.Errorf("recipient username cannot be empty")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	fromUsername := from.GetName()
	if _, exists := c.users[fromUsername]; !exists {
		return fmt.Errorf("sender '%s' is not registered in the chat room", fromUsername)
	}

	if !from.IsOnline() {
		return fmt.Errorf("sender '%s' is offline", fromUsername)
	}

	recipient, exists := c.users[toUsername]
	if !exists {
		return fmt.Errorf("recipient '%s' is not registered in the chat room", toUsername)
	}

	if !recipient.IsOnline() {
		return fmt.Errorf("recipient '%s' is offline", toUsername)
	}

	// Store message in history
	chatMsg := ChatMessage{
		From:      fromUsername,
		To:        toUsername,
		Content:   message,
		Timestamp: time.Now(),
		IsPrivate: true,
	}
	c.addToHistory(chatMsg)

	// Send private message
	privateMsg := fmt.Sprintf("[Private] %s", message)
	recipient.Receive(privateMsg, fromUsername)

	return nil
}

// GetOnlineUsers returns a list of currently online users.
func (c *ChatRoom) GetOnlineUsers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	onlineUsers := make([]string, 0)
	for username, user := range c.users {
		if user.IsOnline() {
			onlineUsers = append(onlineUsers, username)
		}
	}
	return onlineUsers
}

// addToHistory adds a message to the chat history with size limit management.
func (c *ChatRoom) addToHistory(msg ChatMessage) {
	if len(c.messageHistory) >= c.maxHistory {
		// Remove oldest message
		c.messageHistory = c.messageHistory[1:]
	}
	c.messageHistory = append(c.messageHistory, msg)
}

// SetMediator sets the mediator for this user.
func (u *User) SetMediator(mediator Mediator) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.mediator = mediator
}

// Send sends a message through the mediator to all users.
func (u *User) Send(message string) error {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.mediator == nil {
		return fmt.Errorf("mediator not set for user '%s'", u.name)
	}

	return u.mediator.SendMessage(message, u)
}

// SendPrivate sends a private message to a specific user through the mediator.
func (u *User) SendPrivate(message string, toUsername string) error {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.mediator == nil {
		return fmt.Errorf("mediator not set for user '%s'", u.name)
	}

	return u.mediator.SendPrivateMessage(message, u, toUsername)
}

// Receive handles receiving a message from another user.
func (u *User) Receive(message string, from string) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s -> %s: %s\n", timestamp, from, u.name, message)
}

// GetName returns the username.
func (u *User) GetName() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.name
}

// IsOnline returns whether the user is currently online.
func (u *User) IsOnline() bool {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.online
}

// SetOnline sets the online status of the user.
func (u *User) SetOnline(status bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.online = status
}
