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

package consumer_test

import (
	"fmt"
	"github.com/Shopify/sarama"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

func TestConsumerConstructorNoKafka(t *testing.T) {
	storageCfg := storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	}
	storage, err := storage.New(storageCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	brokerCfg := broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
	consumer, err := consumer.New(brokerCfg, storage)
	if err == nil {
		t.Fatal("Error should be reported")
	}
	if consumer != nil {
		t.Fatal("consumer.New should return nil instead of Consumer implementation")
	}
}

func TestParseEmptyMessage(t *testing.T) {
	const message = ``
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "unexpected end of JSON input") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithWrongContent(t *testing.T) {
	const message = `{"this":"is", "not":"expected content"}`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for message that has improper content")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "Missing required attribute 'OrgID'") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithImproperJSON(t *testing.T) {
	const message = `"this_is_not_json_dude"`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for message that does not contain valid JSON")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "json: cannot unmarshal") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseProperMessage(t *testing.T) {
	const messageStr = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}"}
`
	message, err := consumer.ParseMessage([]byte(messageStr))
	if err != nil {
		t.Fatal(err)
	}
	if int(*message.Organization) != 1 {
		t.Fatal("OrgID is different", message.Organization)
	}
	if *message.ClusterName != "aaaaaaaa-bbbb-cccc-dddd-000000000000" {
		t.Fatal("Cluster name is different", *message.ClusterName)
	}
	if *message.Report != "{}" {
		t.Fatal("Report name is different", *message.Report)
	}
}

func TestParseProperMessageWrongClusterName(t *testing.T) {
	const message = `
{"OrgID":1,
 "ClusterName":"this is not a UUID",
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for a wrong ClusterName format")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "Cluster name is not a UUID") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithoutOrgID(t *testing.T) {
	const message = `
{"ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func TestParseMessageWithoutClusterName(t *testing.T) {
	const message = `
{"OrgID":1,
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func TestParseMessageWithoutReport(t *testing.T) {
	const message = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func memoryStorage() (storage.Storage, error) {
	storageCfg := storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	}
	storage, err := storage.New(storageCfg)
	if err != nil {
		return nil, err
	}
	err = storage.Init()
	if err != nil {
		return nil, err
	}
	return storage, nil
}

func dummyConsumer(s storage.Storage) consumer.Consumer {
	brokerCfg := broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
	return consumer.KafkaConsumer{
		Configuration:     brokerCfg,
		Consumer:          nil,
		PartitionConsumer: nil,
		Storage:           s,
	}
}

func TestProcessEmptyMessage(t *testing.T) {
	storage, err := memoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	c := dummyConsumer(storage)

	message := sarama.ConsumerMessage{}
	// messsage is empty -> nothing should be written into storage
	c.ProcessMessage(&message)
	cnt, err := storage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	if cnt != 0 {
		t.Fatal("ProcessMessage wrote anything into DB", cnt)
	}
}

func TestProcessCorrectMessage(t *testing.T) {
	storage, err := memoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	c := dummyConsumer(storage)

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020-01-23T16:15:59.478901889Z"}
`
	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	// messsage is empty -> nothing should be written into storage
	err = c.ProcessMessage(&message)
	if err != nil {
		t.Fatal(err)
	}
	cnt, err := storage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	if cnt == 0 {
		t.Fatal("ProcessMessage does not wrote anything into storage")
	}
	if cnt != 1 {
		t.Fatal("ProcessMessage does more writes than expected")
	}
}

func TestProcessingMessageWithClosedStorage(t *testing.T) {
	storage, err := memoryStorage()
	if err != nil {
		t.Fatal(err)
	}

	c := dummyConsumer(storage)

	storage.Close()

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020-01-23T16:15:59.478901889Z"}
`

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	err = c.ProcessMessage(&message)
	if err == nil {
		t.Fatal(fmt.Errorf("Expected error because database was closed"))
	}
}

func TestProcessingMessageWithWrongDateFormat(t *testing.T) {
	storage, err := memoryStorage()
	if err != nil {
		t.Fatal(err)
	}

	c := dummyConsumer(storage)

	defer storage.Close()

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020.01.23 16:15:59"}
`

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	err = c.ProcessMessage(&message)
	if _, ok := err.(*time.ParseError); err == nil || !ok {
		t.Fatal(fmt.Errorf("Expected time.ParseError error because date format is wrong"))
	}
}
