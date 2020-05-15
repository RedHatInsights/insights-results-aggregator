// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helpers

import (
	"time"

	"github.com/Shopify/sarama"
)

// MockConsumerGroupClaim MockConsumerGroupClaim
type MockConsumerGroupClaim struct {
	channel chan *sarama.ConsumerMessage
}

// NewMockConsumerGroupClaim creates MockConsumerGroupClaim with provided messages
func NewMockConsumerGroupClaim(messages []*sarama.ConsumerMessage) *MockConsumerGroupClaim {
	channel := make(chan *sarama.ConsumerMessage, len(messages)+1)
	for _, message := range messages {
		channel <- message
	}
	close(channel)

	return &MockConsumerGroupClaim{
		channel: channel,
	}
}

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
	return cgc.channel
}

var offset int64 = 0

// StringToSaramaConsumerMessage converts string to sarama consumer message
func StringToSaramaConsumerMessage(str string) *sarama.ConsumerMessage {
	message := &sarama.ConsumerMessage{
		Headers:        nil,
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Value:          []byte(str),
		Topic:          "topic",
		Partition:      0,
		Offset:         offset,
	}
	offset++

	return message
}
