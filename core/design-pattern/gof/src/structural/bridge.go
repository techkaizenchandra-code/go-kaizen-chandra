package structural

import (
	"context"
	"fmt"
	"time"
)

// MessageSender is the Implementor interface that defines the low-level operations
// for sending messages through different channels.
type MessageSender interface {
	SendMessage(ctx context.Context, recipient, message string) error
	GetChannelName() string
}

// EmailSender is a concrete implementor that sends messages via email.
type EmailSender struct {
	smtpHost string
	smtpPort int
}

// NewEmailSender creates a new EmailSender instance.
func NewEmailSender(host string, port int) *EmailSender {
	return &EmailSender{
		smtpHost: host,
		smtpPort: port,
	}
}

// SendMessage sends an email message to the recipient.
func (e *EmailSender) SendMessage(ctx context.Context, recipient, message string) error {
	// Simulate email sending with context cancellation support
	select {
	case <-ctx.Done():
		return fmt.Errorf("email sending cancelled: %w", ctx.Err())
	case <-time.After(100 * time.Millisecond):
		// In production, this would integrate with actual SMTP server
		fmt.Printf("[EMAIL] Sending to %s via %s:%d\nMessage: %s\n", recipient, e.smtpHost, e.smtpPort, message)
		return nil
	}
}

// GetChannelName returns the channel identifier.
func (e *EmailSender) GetChannelName() string {
	return "Email"
}

// SMSSender is a concrete implementor that sends messages via SMS.
type SMSSender struct {
	apiKey     string
	gatewayURL string
	maxRetries int
}

// NewSMSSender creates a new SMSSender instance.
func NewSMSSender(apiKey, gatewayURL string, maxRetries int) *SMSSender {
	return &SMSSender{
		apiKey:     apiKey,
		gatewayURL: gatewayURL,
		maxRetries: maxRetries,
	}
}

// SendMessage sends an SMS message to the recipient.
func (s *SMSSender) SendMessage(ctx context.Context, recipient, message string) error {
	// Simulate SMS sending with retry logic
	for i := 0; i < s.maxRetries; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("sms sending cancelled: %w", ctx.Err())
		case <-time.After(50 * time.Millisecond):
			// In production, this would call actual SMS gateway API
			fmt.Printf("[SMS] Sending to %s via %s (attempt %d/%d)\nMessage: %s\n",
				recipient, s.gatewayURL, i+1, s.maxRetries, message)
			return nil
		}
	}
	return fmt.Errorf("failed to send SMS after %d retries", s.maxRetries)
}

// GetChannelName returns the channel identifier.
func (s *SMSSender) GetChannelName() string {
	return "SMS"
}

// PushNotificationSender is a concrete implementor that sends push notifications.
type PushNotificationSender struct {
	appID       string
	serverKey   string
	environment string // production or staging
}

// NewPushNotificationSender creates a new PushNotificationSender instance.
func NewPushNotificationSender(appID, serverKey, environment string) *PushNotificationSender {
	return &PushNotificationSender{
		appID:       appID,
		serverKey:   serverKey,
		environment: environment,
	}
}

// SendMessage sends a push notification to the recipient.
func (p *PushNotificationSender) SendMessage(ctx context.Context, recipient, message string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("push notification cancelled: %w", ctx.Err())
	case <-time.After(75 * time.Millisecond):
		// In production, this would integrate with FCM/APNS
		fmt.Printf("[PUSH] Sending to %s via %s (%s)\nMessage: %s\n",
			recipient, p.appID, p.environment, message)
		return nil
	}
}

// GetChannelName returns the channel identifier.
func (p *PushNotificationSender) GetChannelName() string {
	return "Push Notification"
}

// Notification is the Abstraction that defines high-level notification operations.
type Notification struct {
	sender    MessageSender
	metadata  map[string]string
	timestamp time.Time
}

// NewNotification creates a new Notification instance with the given sender.
func NewNotification(sender MessageSender) *Notification {
	return &Notification{
		sender:    sender,
		metadata:  make(map[string]string),
		timestamp: time.Now(),
	}
}

// Send sends a notification to the recipient.
func (n *Notification) Send(ctx context.Context, recipient, message string) error {
	if recipient == "" {
		return fmt.Errorf("recipient cannot be empty")
	}
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	fmt.Printf("\n--- Sending %s notification at %s ---\n",
		n.sender.GetChannelName(), n.timestamp.Format(time.RFC3339))

	return n.sender.SendMessage(ctx, recipient, message)
}

// SetMetadata adds metadata to the notification.
func (n *Notification) SetMetadata(key, value string) {
	n.metadata[key] = value
}

// GetMetadata retrieves metadata from the notification.
func (n *Notification) GetMetadata(key string) (string, bool) {
	value, exists := n.metadata[key]
	return value, exists
}

// UrgentNotification is a refined abstraction that adds urgent message handling.
type UrgentNotification struct {
	*Notification
	priority int
}

// NewUrgentNotification creates a new UrgentNotification instance.
func NewUrgentNotification(sender MessageSender, priority int) *UrgentNotification {
	return &UrgentNotification{
		Notification: NewNotification(sender),
		priority:     priority,
	}
}

// Send sends an urgent notification with priority prefix.
func (u *UrgentNotification) Send(ctx context.Context, recipient, message string) error {
	urgentMessage := fmt.Sprintf("[URGENT - Priority %d] %s", u.priority, message)
	u.SetMetadata("priority", fmt.Sprintf("%d", u.priority))
	u.SetMetadata("type", "urgent")

	fmt.Printf("\n⚠️  URGENT NOTIFICATION ⚠️\n")
	return u.Notification.Send(ctx, recipient, urgentMessage)
}

// ScheduledNotification is a refined abstraction that adds scheduling capabilities.
type ScheduledNotification struct {
	*Notification
	scheduledTime time.Time
}

// NewScheduledNotification creates a new ScheduledNotification instance.
func NewScheduledNotification(sender MessageSender, scheduledTime time.Time) *ScheduledNotification {
	return &ScheduledNotification{
		Notification:  NewNotification(sender),
		scheduledTime: scheduledTime,
	}
}

// Send sends a scheduled notification at the specified time.
func (s *ScheduledNotification) Send(ctx context.Context, recipient, message string) error {
	s.SetMetadata("scheduled_time", s.scheduledTime.Format(time.RFC3339))
	s.SetMetadata("type", "scheduled")

	// Wait until scheduled time or context cancellation
	waitDuration := time.Until(s.scheduledTime)
	if waitDuration > 0 {
		fmt.Printf("\n⏰ Waiting %v until scheduled time...\n", waitDuration)
		select {
		case <-ctx.Done():
			return fmt.Errorf("scheduled notification cancelled: %w", ctx.Err())
		case <-time.After(waitDuration):
			// Continue to send
		}
	}

	scheduledMessage := fmt.Sprintf("[Scheduled for %s] %s",
		s.scheduledTime.Format("2006-01-02 15:04:05"), message)
	return s.Notification.Send(ctx, recipient, scheduledMessage)
}

// BatchNotification is a refined abstraction that sends notifications to multiple recipients.
type BatchNotification struct {
	*Notification
	batchSize int
}

// NewBatchNotification creates a new BatchNotification instance.
func NewBatchNotification(sender MessageSender, batchSize int) *BatchNotification {
	return &BatchNotification{
		Notification: NewNotification(sender),
		batchSize:    batchSize,
	}
}

// SendBatch sends notifications to multiple recipients in batches.
func (b *BatchNotification) SendBatch(ctx context.Context, recipients []string, message string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("recipients list cannot be empty")
	}

	b.SetMetadata("batch_size", fmt.Sprintf("%d", b.batchSize))
	b.SetMetadata("total_recipients", fmt.Sprintf("%d", len(recipients)))
	b.SetMetadata("type", "batch")

	fmt.Printf("\n📧 Sending batch notification to %d recipients (batch size: %d)\n",
		len(recipients), b.batchSize)

	errorCount := 0
	for i := 0; i < len(recipients); i += b.batchSize {
		end := i + b.batchSize
		if end > len(recipients) {
			end = len(recipients)
		}

		batch := recipients[i:end]
		fmt.Printf("Processing batch %d-%d...\n", i+1, end)

		for _, recipient := range batch {
			if err := b.Notification.Send(ctx, recipient, message); err != nil {
				fmt.Printf("Error sending to %s: %v\n", recipient, err)
				errorCount++
			}
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("failed to send %d out of %d notifications", errorCount, len(recipients))
	}

	return nil
}

// BridgePatternDemo demonstrates the Bridge pattern usage in a production scenario.
func BridgePatternDemo() {
	fmt.Println("=== Bridge Pattern Demo: Multi-Channel Notification System ===\n")

	// Create different implementors (message senders)
	emailSender := NewEmailSender("smtp.example.com", 587)
	smsSender := NewSMSSender("sync-key-123", "https://sms.gateway.com", 3)
	pushSender := NewPushNotificationSender("app-id-456", "server-key-789", "production")

	ctx := context.Background()

	// Example 1: Regular notifications with different senders
	fmt.Println("\n--- Example 1: Regular Notifications ---")

	emailNotification := NewNotification(emailSender)
	emailNotification.SetMetadata("campaign", "welcome")
	_ = emailNotification.Send(ctx, "user@example.com", "Welcome to our service!")

	smsNotification := NewNotification(smsSender)
	smsNotification.SetMetadata("campaign", "verification")
	_ = smsNotification.Send(ctx, "+1234567890", "Your verification code is: 123456")

	// Example 2: Urgent notifications
	fmt.Println("\n--- Example 2: Urgent Notifications ---")

	urgentEmail := NewUrgentNotification(emailSender, 1)
	_ = urgentEmail.Send(ctx, "admin@example.com", "Server CPU usage exceeded 90%")

	urgentPush := NewUrgentNotification(pushSender, 2)
	_ = urgentPush.Send(ctx, "device-token-xyz", "Security alert: New login detected")

	// Example 3: Scheduled notifications
	fmt.Println("\n--- Example 3: Scheduled Notifications ---")

	scheduledTime := time.Now().Add(2 * time.Second)
	scheduledSMS := NewScheduledNotification(smsSender, scheduledTime)
	_ = scheduledSMS.Send(ctx, "+1234567890", "Reminder: Your appointment is tomorrow")

	// Example 4: Batch notifications
	fmt.Println("\n--- Example 4: Batch Notifications ---")

	batchEmail := NewBatchNotification(emailSender, 2)
	recipients := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
		"user4@example.com",
		"user5@example.com",
	}
	_ = batchEmail.SendBatch(ctx, recipients, "Monthly newsletter: Check out our latest updates!")

	// Example 5: Switching implementation at runtime
	fmt.Println("\n--- Example 5: Runtime Implementation Switching ---")

	notification := NewNotification(emailSender)
	_ = notification.Send(ctx, "user@example.com", "Sending via email")

	// Switch to SMS sender
	notification.sender = smsSender
	_ = notification.Send(ctx, "+1234567890", "Now sending via SMS")

	// Switch to push notification
	notification.sender = pushSender
	_ = notification.Send(ctx, "device-token-abc", "Now sending via push notification")

	fmt.Println("\n=== Bridge Pattern Demo Complete ===")
}
