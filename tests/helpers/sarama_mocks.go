package helpers

import (
	"errors"

	"github.com/Shopify/sarama"
)

// func NewConsumerFromClient(client Client) (Consumer, error)
func NewMockConsumerFromClient(client sarama.Client) (sarama.Consumer, error) {
	return nil, errors.New("Monkey patched!!")
}

func NewMockClient(addrs []string, conf *sarama.Config) (sarama.Client, error) {
	return nil, errors.New("Money patched!!")
}
