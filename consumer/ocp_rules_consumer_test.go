/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

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
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/saramahelpers"
	"github.com/RedHatInsights/insights-results-aggregator/storage"

	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	zerolog_log "github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/IBM/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

var (
	ocpConsumer = consumer.KafkaConsumer{
		MessageProcessor: consumer.OCPRulesProcessor{},
	}
)

func createOCPConsumer(brokerCfg broker.Configuration, mockStorage storage.OCPRecommendationsStorage) *consumer.KafkaConsumer {
	return &consumer.KafkaConsumer{
		Configuration:    brokerCfg,
		Storage:          mockStorage,
		MessageProcessor: consumer.OCPRulesProcessor{},
	}
}

func dummyOCPConsumer(s storage.OCPRecommendationsStorage, allowlist bool) consumer.Consumer {
	brokerCfg := broker.Configuration{
		Addresses: "localhost:1234",
		Topic:     "topic",
		Group:     "group",
	}
	if allowlist {
		brokerCfg.OrgAllowlist = mapset.NewSetWith(types.OrgID(1))
		brokerCfg.OrgAllowlistEnabled = true
	} else {
		brokerCfg.OrgAllowlistEnabled = false
	}
	return createOCPConsumer(brokerCfg, s)
}

func TestOCPRulesConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
		defer closer()

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewOCPRulesConsumer(broker.Configuration{
			Addresses: mockBroker.Addr(),
			Topic:     testTopicName,
			Enabled:   true,
		}, mockStorage)
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}

func TestDeserializeEmptyMessage(t *testing.T) {
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(""))
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestDeserializeMessageWithWrongContent(t *testing.T) {
	const message = `{"this":"is", "not":"expected content"}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required attribute")
}

func TestDeserializeMessageWithImproperJSON(t *testing.T) {
	const message = `"this_is_not_json_dude"`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(
		t,
		err,
		"json: cannot unmarshal string into Go value of type consumer.incomingMessage",
	)
}

func TestDeserializeMessageWithImproperReport(t *testing.T) {
	consumerMessage := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Report": {
			"system": {
				"metadata": {},
				"hostname": null
			},
			"reports": "blablablabla",
			"fingerprints": [],
			"skips": [],
			"info": []
	}
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	message, err := consumer.DeserializeMessage(&c, []byte(consumerMessage))
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)
}

func TestDeserializeProperMessage(t *testing.T) {
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	message, err := consumer.DeserializeMessage(&c, []byte(testdata.ConsumerMessage))
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)
}

func TestDeserializeMessageWrongClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "this is not a UUID",
		"Report": ` + testdata.ConsumerReport + `
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "cluster name is not a UUID")
}

func TestDeserializeMessageWithoutOrgID(t *testing.T) {
	message := `{
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": ` + testdata.ConsumerReport + `
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestDeserializeMessageWithoutClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"Report": ` + testdata.ConsumerReport + `
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'ClusterName'")
}

func TestDeserializeMessageWithoutReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func TestDeserializeMessageWithEmptyReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": {}
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.Nil(t, err, "deserializeMessage should not return error for empty report")
}

func TestDeserializeMessageNullReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": null
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.OCPRulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func TestIsReportWithEmptyAttributesAllEmpty(t *testing.T) {
	r := consumer.Report{
		"system":       unmarshall(`{"metadata": {}, "hostname": null}`),
		"reports":      unmarshall("[]"),
		"fingerprints": unmarshall("[]"),
		"skips":        unmarshall("[]"),
		"info":         unmarshall("[]"),
	}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.True(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = true for this report")
}

func TestIsReportWithEmptyAttributesEmptyReport(t *testing.T) {
	r := consumer.Report{}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.True(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = true for this report")
}

func TestIsReportWithEmptyAttributesSystemDataIsPresent(t *testing.T) {
	r := consumer.Report{
		"system":       unmarshall(`{"metadata": {}, "hostname": "a_hostname_that_can_be_unmarshalled"}`),
		"reports":      unmarshall("[]"),
		"fingerprints": unmarshall("[]"),
		"skips":        unmarshall("[]"),
		"info":         unmarshall("[]"),
	}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.False(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = false for this report")
}

// Additional existence check have been added to avoid the possibility of
// panicking due to a nil pointer exception
func TestIsReportWithEmptyAttributesLessAttributes(t *testing.T) {
	r := consumer.Report{
		"system":       unmarshall(`{"metadata": {}, "hostname": "a_hostname_that_can_be_unmarshalled"}`),
		"reports":      unmarshall("[]"),
		"fingerprints": unmarshall("[]"),
	}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.False(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = false for this report")
}

func TestIsReportWithEmptyAttributesInfoIsNotPresent(t *testing.T) {
	r := consumer.Report{
		"system":       unmarshall(`{"metadata": {}, "hostname": null}`),
		"reports":      unmarshall("[]"),
		"fingerprints": unmarshall("[]"),
		"skips":        unmarshall("[]"),
	}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.True(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = true for this report")
}

func TestIsReportWithEmptyAttributesReportsIsPresent(t *testing.T) {
	reportItems := `[
		{
			"component": "test.rule",
			"key": "test.key",
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		}
	]`
	r := consumer.Report{
		"system":       unmarshall(`{"metadata": {}, "hostname": null}`),
		"reports":      unmarshall(reportItems),
		"fingerprints": unmarshall("[]"),
		"skips":        unmarshall("[]"),
	}
	isEmpty := consumer.IsReportWithEmptyAttributes(r)
	assert.False(t, isEmpty, "IsReportWithEmptyAttributes should return isEmpty = false for this report")
}

func TestCheckReportStructureEmptyReport(t *testing.T) {
	report := consumer.Report{}

	err := consumer.CheckReportStructure(report)
	assert.EqualError(t, err, "empty report found in deserialized message")
}

func TestCheckReportStructureReportWithAllAttributesPresentAndEmpty(t *testing.T) {
	report := consumer.Report{
		"fingerprints": unmarshall("[]"),
		"info":         unmarshall("[]"),
		"reports":      unmarshall("[]"),
		"skips":        unmarshall("[]"),
		"system":       unmarshall("{}"),
	}
	err := consumer.CheckReportStructure(report)
	helpers.FailOnError(t, err, "empty report with all expected attributes present should be processed")
}

func TestCheckReportStructureReportWithAnalysisMetadata(t *testing.T) {
	report := consumer.Report{
		"system":            unmarshall(`{"metadata": {}, "hostname": null}`),
		"fingerprints":      unmarshall("[]"),
		"analysis_metadata": unmarshall(`{"start": "2023-09-11T18:33:14.527845+00:00", "finish": "2023-09-11T18:33:15.632777+00:00"}`),
	}
	err := consumer.CheckReportStructure(report)
	assert.EqualError(t, err, "empty report found in deserialized message")
}

// If some attributes are missing, but all the present attributes are empty, we just
// skip the processing of the message.
func TestCheckReportStructureReportWithEmptyAndMissingAttributes(t *testing.T) {
	report := consumer.Report{
		"fingerprints": unmarshall("[]"),
		"reports":      unmarshall("[]"),
		"skips":        unmarshall("[]"),
	}
	err := consumer.CheckReportStructure(report)
	assert.EqualError(t, err, "empty report found in deserialized message")
}

func TestCheckReportStructureReportWithItems(t *testing.T) {
	report := consumer.Report{
		"fingerprints": unmarshall("[]"),
		"reports":      unmarshall(string(testdata.Report2Rules)),
		"skips":        unmarshall("[]"),
		"system":       unmarshall(`{"metadata": {},"hostname": null}`),
	}
	err := consumer.CheckReportStructure(report)
	assert.Nil(t, err, "checkReportStructure should return err = nil for empty reports")
}

func TestParseReportContentEmptyReportsAttribute(t *testing.T) {
	deserialized, err := consumer.DeserializeMessage(&ocpConsumer, []byte(testdata.ConsumerMessage))
	assert.Nil(t, err, "deserializeMessage should not return error for this message")

	err = consumer.ParseReportContent(&deserialized)
	assert.Nil(t, err, "ParseReportContent should not return nil for empty report")
}

func TestParseReportContentValidReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + string(testdata.Report2Rules) + `
	}`

	deserialized, err := consumer.DeserializeMessage(&ocpConsumer, []byte(message))
	assert.Nil(t, err, "deserializeMessage should not return error for this message")

	err = consumer.ParseReportContent(&deserialized)
	assert.Nil(t, err, "ParseReportContent should return nil for reports with proper structure")
}

func TestParseEmptyMessage(t *testing.T) {
	message := sarama.ConsumerMessage{}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestParseMessageWithWrongContent(t *testing.T) {
	message := sarama.ConsumerMessage{Value: []byte(`{"this":"is", "not":"expected content"}`)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestParseProperMessageWrongClusterName(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "this is not a UUID",
		"Report": ` + testdata.ConsumerReport + `
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "cluster name is not a UUID")
}

func TestParseMessageWithoutOrgID(t *testing.T) {
	data := `{
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": ` + testdata.ConsumerReport + `
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestParseMessageWithoutClusterName(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"Report": ` + testdata.ConsumerReport + `
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'ClusterName'")
}

func TestParseMessageWithoutReport(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func TestParseMessageEmptyReport(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": {}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "empty report found in deserialized message")
}
func TestParseMessageNullReport(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": null
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func TestParseMessageWithImproperJSON(t *testing.T) {
	message := sarama.ConsumerMessage{Value: []byte(`"this_is_not_json_dude"`)}
	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "json: cannot unmarshal string into Go value of type consumer.incomingMessage")
}

func TestParseMessageWithImproperReport(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Report": {
			"system": {
				"metadata": {},
				"hostname": null
			},
			"reports": "blablablabla",
			"fingerprints": [],
			"skips": [],
			"info": []
		}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}

	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	assert.EqualError(t, err, "json: cannot unmarshal string into Go value of type []types.ReportItem")
}

func TestParseProperMessageReportWithEmptyAttributes(t *testing.T) {
	message := sarama.ConsumerMessage{Value: []byte(testdata.ConsumerMessage)}
	parsed, err := consumer.ParseMessage(&ocpConsumer, &message)
	helpers.FailOnError(t, err, "empty report with all expected attributes present should be processed")

	assert.Equal(t, types.OrgID(1), *parsed.Organization)
	assert.Equal(t, testdata.ClusterName, *parsed.ClusterName)

	var expectedReport consumer.Report
	err = json.Unmarshal([]byte(testdata.ConsumerReport), &expectedReport)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedReport, *parsed.Report)
	assert.EqualValues(t, []types.ReportItem{}, parsed.ParsedHits)
}

func TestParseProperMessageWithInfoReport(t *testing.T) {
	// this report contains just one rule hit: from special INFO rule used
	// just in external data pipeline
	consumerReport := `{
		"fingerprints": [],
		"reports": [],
		"skips": [],
		"system": {},
		"info": [
			{
				"info_id": "version_info|CLUSTER_VERSION_INFO",
				"component": "ccx_rules_processing.version_info.report",
				"type": "info",
				"key": "CLUSTER_VERSION_INFO",
				"details": {
				  "version": "4.9",
				  "type": "info",
				  "info_key": "CLUSTER_VERSION_INFO"
				},
				"tags": [],
				"links": {}
			  }
		]
	}`
	message := sarama.ConsumerMessage{Value: []byte(createConsumerMessage(consumerReport))}
	parsed, err := consumer.ParseMessage(&ocpConsumer, &message)
	helpers.FailOnError(t, err, "this message is valid and should be processed")
	assert.Equal(t, types.OrgID(1), *parsed.Organization)
	assert.Equal(t, testdata.ClusterName, *parsed.ClusterName)

	var expectedReport consumer.Report
	err = json.Unmarshal([]byte(consumerReport), &expectedReport)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedReport, *parsed.Report)
	assert.EqualValues(t, []types.ReportItem{}, parsed.ParsedHits)

	expectedInfoReport := []types.InfoItem{
		{
			InfoID:  "version_info|CLUSTER_VERSION_INFO",
			InfoKey: "CLUSTER_VERSION_INFO",
			Details: map[string]string{
				"version":  "4.9",
				"type":     "info",
				"info_key": "CLUSTER_VERSION_INFO",
			},
		},
	}
	assert.EqualValues(t, expectedInfoReport, parsed.ParsedInfo)
}

func TestParseProperMessageNoInfoAttribute(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Report": {
			"system": {
				"metadata": {},
				"hostname": null
			},
			"reports": [
				{
					"component": "` + string(testdata.Rule2ID) + `",
					"key": "` + testdata.ErrorKey2 + `",
					"user_vote": 0,
					"disabled": ` + fmt.Sprint(testdata.Rule2Disabled) + `,
					"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
				}
			],
			"fingerprints": [],
			"skips": []
		}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}

	_, err := consumer.ParseMessage(&ocpConsumer, &message)
	helpers.FailOnError(t, err)
}

func TestProcessEmptyMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	c := dummyOCPConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	// message is empty -> nothing should be written into storage
	err := c.HandleMessage(&message)
	assert.EqualError(t, err, "unexpected end of JSON input")

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	// no record should be written into database
	assert.Equal(
		t,
		0,
		count,
		"process message shouldn't write anything into the DB",
	)
}

func TestProcessCorrectMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)

	defer closer()

	c := dummyOCPConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageReportWithRuleHits)
	// message is correct -> one record should be written into storage
	err := c.HandleMessage(&message)
	helpers.FailOnError(t, err)

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	// exactly one record should be written into database
	assert.Equal(t, 1, count, "process message should write one record into DB")
}

func TestProcessingEmptyReportMissingAttributesWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)

	mockConsumer := dummyOCPConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
	helpers.FailOnError(t, err, "empty report should not be considered an error at HandleMessage level")
}

func TestProcessingValidMessageEmptyReportWithRequiredAttributesWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)

	mockConsumer := dummyOCPConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestProcessingCorrectMessageWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)

	mockConsumer := dummyOCPConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestProcessingMessageWithWrongDateFormatAndEmptyReport(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := dummyOCPConsumer(mockStorage, true)

	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
	assert.Nil(t, err, "Message with empty report should not be processed")
}

func TestProcessingMessageWithWrongDateFormatReportNotEmpty(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := dummyOCPConsumer(mockStorage, true)

	messageValue := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "2020.01.23 16:15:59"
	}`
	err := consumerProcessMessage(mockConsumer, messageValue)
	if _, ok := err.(*time.ParseError); err == nil || !ok {
		t.Fatal(fmt.Errorf(
			"expected time.ParseError error because date format is wrong. Got %+v", err,
		))
	}
}

func TestKafkaConsumerMockOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockOCPRulesConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgAllowlist,
			[]string{messageReportWithRuleHits},
		)

		go mockConsumer.Serve()

		// wait for message processing
		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockBadMessage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockOCPRulesConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgAllowlist,
			[]string{"bad message"},
		)

		go mockConsumer.Serve()

		// wait for message processing
		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockWritingMsgWithEmptyReportToClosedStorage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockOCPRulesConsumerWithExpectedMessages(
			t, testTopicName, testOrgAllowlist, []string{messageNoReportsNoInfo},
		)

		err := mockConsumer.KafkaConsumer.Storage.Close()
		helpers.FailOnError(t, err)

		go mockConsumer.Serve()

		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		// Since the report is present but empty, we stop processing this message without errors
		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockWritingMsgWithReportToClosedStorage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockOCPRulesConsumerWithExpectedMessages(
			t, testTopicName, testOrgAllowlist, []string{messageReportWithRuleHits},
		)

		err := mockConsumer.KafkaConsumer.Storage.Close()
		helpers.FailOnError(t, err)

		go mockConsumer.Serve()

		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		// Since the report is present and not empty, it is processed, and we reach the closed DB error
		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumer_ProcessMessage_OrganizationAllowlistDisabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := dummyOCPConsumer(mockStorage, false)

	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
	helpers.FailOnError(t, err)
}

func TestKafkaConsumer_ProcessMessageWithEmptyReport_OrganizationIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Addresses:           "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        mapset.NewSetWith(types.OrgID(123)), // in testdata, OrgID = 1
		OrgAllowlistEnabled: true,
	}
	mockConsumer := createOCPConsumer(brokerCfg, mockStorage)

	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
	helpers.FailOnError(t, err, "message have empty report and should not be processed")
}

func TestKafkaConsumer_ProcessMessage_OrganizationIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Addresses:           "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        mapset.NewSetWith(types.OrgID(123)), // in testdata, OrgID = 1
		OrgAllowlistEnabled: true,
	}
	mockConsumer := createOCPConsumer(brokerCfg, mockStorage)

	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
	assert.EqualError(t, err, organizationIDNotInAllowList)
}

func TestKafkaConsumer_ProcessMessageWithEmptyReport_OrganizationBadConfigIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Addresses:           "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        nil,
		OrgAllowlistEnabled: true,
	}
	mockConsumer := createOCPConsumer(brokerCfg, mockStorage)

	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
	helpers.FailOnError(t, err, "message have empty report and should not be processed")
}

func TestKafkaConsumer_ProcessMessage_OrganizationBadConfigIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Addresses:           "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        nil,
		OrgAllowlistEnabled: true,
	}
	mockConsumer := createOCPConsumer(brokerCfg, mockStorage)

	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
	assert.EqualError(t, err, organizationIDNotInAllowList)
}

func TestKafkaConsumer_ProcessMessage_MessageFromTheFuture(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := createOCPConsumer(wrongBrokerCfg, mockStorage)

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339) + `"
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)
	assert.Contains(t, buf.String(), "got a message from the future")
}

func TestKafkaConsumer_ProcessMessage_MoreRecentReportAlreadyExists(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := createOCPConsumer(wrongBrokerCfg, mockStorage)

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Format(time.RFC3339) + `"
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	message = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `"
	}`

	err = consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "Skipping because a more recent report already exists for this cluster")
}

func TestKafkaConsumer_ProcessMessage_MessageWithNoSchemaVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := createOCPConsumer(wrongBrokerCfg, mockStorage)

	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
	helpers.FailOnError(t, err)
	assert.Contains(t, buf.String(), "\"level\":\"warn\"")
	assert.Contains(t, buf.String(), "Received data with unexpected version")
}

func TestKafkaConsumer_ProcessMessage_MessageWithUnexpectedSchemaVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := createOCPConsumer(wrongBrokerCfg, mockStorage)

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(3)) + `
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)
	assert.Contains(t, buf.String(), "\"level\":\"warn\"")
	assert.Contains(t, buf.String(), "Received data with unexpected version")
}

func TestKafkaConsumer_ProcessMessage_MessageWithExpectedSchemaVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockConsumer := createOCPConsumer(wrongBrokerCfg, mockStorage)

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(1)) + `
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	message = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(2)) + `
	}`

	err = consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	assert.NotContains(t, buf.String(), "\"level\":\"warn\"")
	assert.NotContains(t, buf.String(), "Received data with unexpected version")
}

func TestKafkaConsumer_ConsumeClaim(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	kafkaConsumer := createOCPConsumer(broker.Configuration{}, mockStorage)

	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim(nil)

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)
}

func TestKafkaConsumer_ConsumeClaim_DBError(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	closer()

	kafkaConsumer := createOCPConsumer(broker.Configuration{}, mockStorage)

	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim(nil)

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "starting messages loop")
}

func TestKafkaConsumer_ConsumeClaim_OKMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	kafkaConsumer := createOCPConsumer(broker.Configuration{}, mockStorage)

	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim([]*sarama.ConsumerMessage{
		saramahelpers.StringToSaramaConsumerMessage(testdata.ConsumerMessage),
	})

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)
}
