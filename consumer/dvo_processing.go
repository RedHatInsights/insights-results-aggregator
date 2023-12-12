// Copyright 2023 Red Hat, Inc
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

package consumer

import (
	"encoding/json"
	"errors"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

// DVORulesProcessor satisfies MessageProcessor interface
type DVORulesProcessor struct {
}

func (DVORulesProcessor) deserializeMessage(messageValue []byte) (incomingMessage, error) {
	var deserialized incomingMessage

	err := json.Unmarshal(messageValue, &deserialized)
	if err != nil {
		return deserialized, err
	}
	if deserialized.Organization == nil {
		return deserialized, errors.New("missing required attribute 'OrgID'")
	}
	if deserialized.ClusterName == nil {
		return deserialized, errors.New("missing required attribute 'ClusterName'")
	}
	if deserialized.DvoMetrics == nil {
		return deserialized, errors.New("missing required attribute 'Metrics'")
	}
	_, err = uuid.Parse(string(*deserialized.ClusterName))
	if err != nil {
		return deserialized, errors.New("cluster name is not a UUID")
	}
	return deserialized, nil
}

func (DVORulesProcessor) parseMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (incomingMessage, error) {
	//TODO implement me
	panic("implement me")
}

func (DVORulesProcessor) processMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error) {
	//TODO implement me
	panic("implement me")
}

func (DVORulesProcessor) shouldProcess(consumer *KafkaConsumer, consumed *sarama.ConsumerMessage, parsed *incomingMessage) error {
	//TODO implement me
	panic("implement me")
}
