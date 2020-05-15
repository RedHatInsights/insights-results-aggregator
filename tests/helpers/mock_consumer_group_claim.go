package helpers

import "github.com/Shopify/sarama"

// TODO: make it produce messages

// MockConsumerGroupClaim MockConsumerGroupClaim
type MockConsumerGroupClaim struct{}

// Topic returns the consumed topic name.
func (cgc *MockConsumerGroupClaim) Topic() string {
	return ""
}

// Partition returns the consumed partition.
func (cgc *MockConsumerGroupClaim) Partition() int32 {
	return 0
}

// InitialOffset returns the initial offset that was used as a starting point for this claim.
func (cgc *MockConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

// HighWaterMarkOffset returns the high water mark offset of the partition,
func (cgc *MockConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return 0
}

// Messages returns the read channel for the messages that are returned by
func (cgc *MockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	ch := make(chan *sarama.ConsumerMessage)
	close(ch)
	return ch
}
