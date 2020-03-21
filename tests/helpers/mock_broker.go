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
	"testing"

	"github.com/nash-io/jocko/protocol"

	"github.com/nash-io/jocko/jocko"
	"github.com/nash-io/jocko/jocko/config"
)

const (
	defaultTopicName  = "topic"
	numPartitions     = 1
	replicationFactor = 1
)

// MockBroker is a mock broker based on "github.com/nash-io/jocko" library
// prefer using MustGetMockKafkaConsumerWithExpectedMessages where it's possible
type MockBroker struct {
	Address    string
	TopicName  string
	Group      string
	server     *jocko.Server
	connection *jocko.Conn
}

// MustNewMockBroker creates a new mock broker based on "github.com/nash-io/jocko" library
// raises an error if creation wasn't successful
// prefer using MustGetMockKafkaConsumerWithExpectedMessages where it's possible
func MustNewMockBroker(t *testing.T) *MockBroker {
	mockBroker, err := NewMockBroker(t, defaultTopicName)
	FailOnError(t, err)
	return mockBroker
}

// NewMockBroker creates a new mock broker based on "github.com/nash-io/jocko" library
// prefer using MustGetMockKafkaConsumerWithExpectedMessages where it's possible
func NewMockBroker(t *testing.T, topicName string) (*MockBroker, error) {
	server, _ := jocko.NewTestServer(t, func(cfg *config.Config) {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 1
		cfg.StartAsLeader = true
	}, nil)
	err := server.Start(context.Background())
	if err != nil {
		return nil, err
	}

	address := server.Addr().String()

	connection, err := jocko.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	mockBroker := &MockBroker{
		Address:    address,
		TopicName:  topicName,
		server:     server,
		connection: connection,
	}

	if len(topicName) != 0 {
		err := mockBroker.createTopic(topicName)
		if err != nil {
			return nil, err
		}
	}

	return mockBroker, nil
}

func (mockBroker *MockBroker) createTopic(topicName string) error {
	resp, err := mockBroker.connection.CreateTopics(&protocol.CreateTopicRequests{
		Requests: []*protocol.CreateTopicRequest{{
			Topic:             topicName,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		}},
	})
	if err != nil {
		return err
	}

	for _, topicErrCode := range resp.TopicErrorCodes {
		if topicErrCode.ErrorCode != protocol.ErrNone.Code() &&
			topicErrCode.ErrorCode != protocol.ErrTopicAlreadyExists.Code() {
			return protocol.Errs[topicErrCode.ErrorCode]
		}
	}

	return nil
}

// MustClose cleans up connection and mock server
// raises an error if it wasn't successful
func (mockBroker *MockBroker) MustClose(t *testing.T) {
	FailOnError(t, mockBroker.Close())
}

// Close cleans up connection and mock server
func (mockBroker *MockBroker) Close() error {
	err := mockBroker.connection.Close()
	if err != nil {
		return err
	}

	return mockBroker.server.Shutdown()
}
