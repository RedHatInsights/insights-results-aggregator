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
	"context"

	"github.com/Shopify/sarama"
)

// MockConsumerGroupSession MockConsumerGroupSession
type MockConsumerGroupSession struct{}

// Claims returns information about the claimed partitions by topic.
func (*MockConsumerGroupSession) Claims() map[string][]int32 {
	return nil
}

// MemberID returns the cluster member ID.
func (*MockConsumerGroupSession) MemberID() string {
	return ""
}

// GenerationID returns the current generation ID.
func (*MockConsumerGroupSession) GenerationID() int32 {
	return 0
}

// MarkOffset marks the provided offset, alongside a metadata string
func (*MockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

// ResetOffset resets to the provided offset, alongside a metadata string that
func (*MockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

// MarkMessage marks a message as consumed.
func (*MockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {}

// Context returns the session context.
func (*MockConsumerGroupSession) Context() context.Context {
	return context.TODO()
}

// Commit commits
func (*MockConsumerGroupSession) Commit() {}
