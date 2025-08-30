package rabbitmq

import "errors"

// Common RabbitMQ errors
var (
	// Connection errors
	ErrConnectionClosed  = errors.New("rabbitmq connection is closed")
	ErrConnectionTimeout = errors.New("rabbitmq connection timeout")
	ErrReconnectFailed   = errors.New("failed to reconnect to rabbitmq")

	// Channel errors
	ErrChannelClosed         = errors.New("rabbitmq channel is closed")
	ErrChannelCreationFailed = errors.New("failed to create rabbitmq channel")

	// Publishing errors
	ErrPublishFailed    = errors.New("failed to publish message")
	ErrExchangeNotFound = errors.New("exchange not found")
	ErrQueueNotFound    = errors.New("queue not found")
	ErrInvalidMessage   = errors.New("invalid message format")

	// Consumer errors
	ErrConsumerClosed         = errors.New("consumer is closed")
	ErrConsumerAlreadyRunning = errors.New("consumer is already running")
	ErrNoHandlerFound         = errors.New("no message handler found")

	// Middleware errors
	ErrPanicRecovered      = errors.New("recovered from panic during message processing")
	ErrProcessingTimeout   = errors.New("message processing timeout")
	ErrValidationFailed    = errors.New("message validation failed")
	ErrDeduplicationFailed = errors.New("message deduplication failed")

	// Configuration errors
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrMissingURL    = errors.New("rabbitmq url is required")
	ErrInvalidURL    = errors.New("invalid rabbitmq url")

	// Queue errors
	ErrQueueDeclarationFailed = errors.New("failed to declare queue")
	ErrQueueBindFailed        = errors.New("failed to bind queue")
	ErrQueuePurgeFailed       = errors.New("failed to purge queue")
	ErrQueueDeleteFailed      = errors.New("failed to delete queue")

	// Exchange errors
	ErrExchangeDeclarationFailed = errors.New("failed to declare exchange")
	ErrExchangeDeleteFailed      = errors.New("failed to delete exchange")
)

// IsConnectionError checks if the error is related to connection issues
func IsConnectionError(err error) bool {
	return err == ErrConnectionClosed ||
		err == ErrConnectionTimeout ||
		err == ErrReconnectFailed
}

// IsChannelError checks if the error is related to channel issues
func IsChannelError(err error) bool {
	return err == ErrChannelClosed ||
		err == ErrChannelCreationFailed
}

// IsPublishError checks if the error is related to publishing
func IsPublishError(err error) bool {
	return err == ErrPublishFailed ||
		err == ErrExchangeNotFound ||
		err == ErrQueueNotFound ||
		err == ErrInvalidMessage
}

// IsConsumerError checks if the error is related to consuming
func IsConsumerError(err error) bool {
	return err == ErrConsumerClosed ||
		err == ErrConsumerAlreadyRunning ||
		err == ErrNoHandlerFound
}

// IsMiddlewareError checks if the error is related to middleware
func IsMiddlewareError(err error) bool {
	return err == ErrPanicRecovered ||
		err == ErrProcessingTimeout ||
		err == ErrValidationFailed ||
		err == ErrDeduplicationFailed
}

// IsRetryableError checks if the error should trigger a retry
func IsRetryableError(err error) bool {
	return IsConnectionError(err) ||
		IsChannelError(err) ||
		err == ErrProcessingTimeout
}

// IsTemporaryError checks if the error is temporary and should be retried
func IsTemporaryError(err error) bool {
	return IsConnectionError(err) ||
		IsChannelError(err) ||
		err == ErrProcessingTimeout ||
		err == ErrPublishFailed
}
