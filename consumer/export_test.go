/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package consumer

import "github.com/Shopify/sarama"

// Export for testing
//
// This source file contains name aliases of all package-private functions
// that need to be called from unit tests. Aliases should start with uppercase
// letter because unit tests belong to different package.
//
// Please look into the following blogpost:
// https://medium.com/@robiplus/golang-trick-export-for-test-aa16cbd7b8cd
// to see why this trick is needed.
var (
	CheckReportStructure         = checkReportStructure
	IsReportWithEmptyAttributes  = isReportWithEmptyAttributes
	NumberOfExpectedKeysInReport = numberOfExpectedKeysInReport
	ExpectedKeysInReport         = expectedKeysInReport
)

// Inc type is a trick to get golint to work for the ParseMessage defined below...
type Inc struct {
	incomingMessage
}

// DeserializeMessage returns the result of the private MessageProcessor.DeserializeMessage method
func DeserializeMessage(consumer *KafkaConsumer, msg []byte) (Inc, error) {
	incomingMessage, err := consumer.MessageProcessor.deserializeMessage(msg)
	return Inc{incomingMessage}, err
}

// ParseMessage returns the result of the private MessageProcessor.parseMessage method
func ParseMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (Inc, error) {
	incomingMessage, err := consumer.MessageProcessor.parseMessage(consumer, msg)
	return Inc{incomingMessage}, err
}

// ParseMessage returns the result of the private parseReportContent function
func ParseReportContent(message *Inc) error {
	return parseReportContent(&message.incomingMessage)
}
