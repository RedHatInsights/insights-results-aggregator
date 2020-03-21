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

type MockBroker struct {
	Address    string
	TopicName  string
	Group      string
	server     *jocko.Server
	connection *jocko.Conn
}

func MustNewMockBroker(t *testing.T) *MockBroker {
	mockBroker, err := NewMockBroker(t, defaultTopicName)
	FailOnError(t, err)
	return mockBroker
}

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

func (mockBroker *MockBroker) MustClose(t *testing.T) {
	FailOnError(t, mockBroker.Close())
}

func (mockBroker *MockBroker) Close() error {
	err := mockBroker.connection.Close()
	if err != nil {
		return err
	}

	return mockBroker.server.Shutdown()
}
