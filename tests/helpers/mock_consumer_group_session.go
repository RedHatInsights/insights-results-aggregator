package helpers

import (
	"context"

	"github.com/Shopify/sarama"
)

// MockConsumerGroupSession MockConsumerGroupSession
type MockConsumerGroupSession struct{}

// Claims returns information about the claimed partitions by topic.
func (cgs *MockConsumerGroupSession) Claims() map[string][]int32 {
	return nil
}

// MemberID returns the cluster member ID.
func (cgs *MockConsumerGroupSession) MemberID() string {
	return ""
}

// GenerationID returns the current generation ID.
func (cgs *MockConsumerGroupSession) GenerationID() int32 {
	return 0
}

// MarkOffset marks the provided offset, alongside a metadata string
func (cgs *MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

// ResetOffset resets to the provided offset, alongside a metadata string that
func (cgs *MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

// MarkMessage marks a message as consumed.
func (cgs *MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {}

// Context returns the session context.
func (cgs *MockConsumerGroupSession) Context() context.Context {
	return context.TODO()
}
