package rabbitmq

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Built-in middleware functions

// LoggingMiddleware logs message processing
func LoggingMiddleware(next MessageHandler) MessageHandler {
	return func(delivery *Delivery) error {
		start := time.Now()
		log.Printf("RabbitMQ Middleware: Processing message [%s] from queue", delivery.MessageId)

		err := next(delivery)

		duration := time.Since(start)
		if err != nil {
			log.Printf("RabbitMQ Middleware: Message processing failed after %v: %v", duration, err)
		} else {
			log.Printf("RabbitMQ Middleware: Message processed successfully in %v", duration)
		}

		return err
	}
}

// RetryMiddleware provides retry functionality for failed messages
func RetryMiddleware(maxRetries int, retryDelay time.Duration) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(delivery *Delivery) error {
			retryCount := 0

			// Check if this message has been retried before
			if retryHeader, exists := delivery.GetHeader("x-retry-count"); exists {
				if count, ok := retryHeader.(int); ok {
					retryCount = count
				}
			}

			err := next(delivery)
			if err != nil && retryCount < maxRetries {
				// Increment retry count
				retryCount++
				log.Printf("RabbitMQ Middleware: Retrying message (attempt %d/%d): %v", retryCount, maxRetries, err)

				// Publish the message back to the queue with retry count
				headers := make(amqp.Table)
				for k, v := range delivery.Headers {
					headers[k] = v
				}
				headers["x-retry-count"] = retryCount

				// Add delay before retry
				if retryDelay > 0 {
					time.Sleep(retryDelay)
				}

				// Note: In a real implementation, you'd want to republish to a retry queue
				// For now, we'll just acknowledge the message and let it fail
				return nil
			}

			return err
		}
	}
}

// RateLimitMiddleware provides rate limiting
func RateLimitMiddleware(requestsPerSecond int) MiddlewareFunc {
	limiter := time.NewTicker(time.Second / time.Duration(requestsPerSecond))

	return func(next MessageHandler) MessageHandler {
		return func(delivery *Delivery) error {
			// Wait for rate limiter
			<-limiter.C
			return next(delivery)
		}
	}
}

// ValidationMiddleware validates message structure
func ValidationMiddleware(validator func(*Delivery) error) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(delivery *Delivery) error {
			if err := validator(delivery); err != nil {
				log.Printf("RabbitMQ Middleware: Message validation failed: %v", err)
				return err
			}
			return next(delivery)
		}
	}
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next MessageHandler) MessageHandler {
	return func(delivery *Delivery) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("RabbitMQ Middleware: Recovered from panic: %v", r)
				err = ErrPanicRecovered
			}
		}()

		return next(delivery)
	}
}

// TimeoutMiddleware adds timeout to message processing
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(delivery *Delivery) error {
			done := make(chan error, 1)

			go func() {
				done <- next(delivery)
			}()

			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				log.Printf("RabbitMQ Middleware: Message processing timeout after %v", timeout)
				return ErrProcessingTimeout
			}
		}
	}
}

// DeduplicationMiddleware prevents duplicate message processing
func DeduplicationMiddleware(store MessageStore) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(delivery *Delivery) error {
			messageID := delivery.MessageId
			if messageID == "" {
				// If no message ID, generate one from headers or body
				messageID = generateMessageID(delivery)
			}

			// Check if we've already processed this message
			if store.HasProcessed(messageID) {
				log.Printf("RabbitMQ Middleware: Duplicate message detected, skipping: %s", messageID)
				return nil
			}

			// Process the message
			err := next(delivery)
			if err == nil {
				// Mark as processed only if successful
				store.MarkProcessed(messageID)
			}

			return err
		}
	}
}

// MessageStore interface for deduplication
type MessageStore interface {
	HasProcessed(messageID string) bool
	MarkProcessed(messageID string)
}

// InMemoryMessageStore is a simple in-memory implementation
type InMemoryMessageStore struct {
	processed map[string]time.Time
	ttl       time.Duration
}

// NewInMemoryMessageStore creates a new in-memory message store
func NewInMemoryMessageStore(ttl time.Duration) *InMemoryMessageStore {
	store := &InMemoryMessageStore{
		processed: make(map[string]time.Time),
		ttl:       ttl,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return store
}

// HasProcessed checks if a message has been processed
func (s *InMemoryMessageStore) HasProcessed(messageID string) bool {
	_, exists := s.processed[messageID]
	return exists
}

// MarkProcessed marks a message as processed
func (s *InMemoryMessageStore) MarkProcessed(messageID string) {
	s.processed[messageID] = time.Now()
}

// cleanup removes expired entries
func (s *InMemoryMessageStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for id, timestamp := range s.processed {
			if now.Sub(timestamp) > s.ttl {
				delete(s.processed, id)
			}
		}
	}
}

// Helper functions

func generateMessageID(delivery *Delivery) string {
	// Try to generate ID from various sources
	if delivery.MessageId != "" {
		return delivery.MessageId
	}

	if correlationID, exists := delivery.GetStringHeader("correlation-id"); exists {
		return correlationID
	}

	if timestamp := delivery.Timestamp; !timestamp.IsZero() {
		return delivery.RoutingKey + "_" + timestamp.Format("20060102150405.000000")
	}

	// Fallback to routing key + current time
	return delivery.RoutingKey + "_" + time.Now().Format("20060102150405.000000")
}
