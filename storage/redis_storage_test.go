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

package storage_test

import (
	"fmt"
	"time"

	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// default Redis configuration
var configuration = storage.Configuration{
	RedisConfiguration: storage.RedisConfiguration{
		RedisEndpoint:       "localhost:12345",
		RedisDatabase:       0,
		RedisTimeoutSeconds: 1,
		RedisPassword:       "",
	},
}

// getMockRedis is used to get a mocked Redis client to expect and
// respond to queries
func getMockRedis(t *testing.T) (
	mockClient storage.RedisStorage, mockServer redismock.ClientMock,
) {
	client, mockServer := redismock.NewClientMock()
	mockClient = storage.RedisStorage{
		Client: redis.Client{Connection: client},
	}
	err := mockClient.Init()
	if err != nil {
		t.Fatal(err)
	}
	return
}

// assertRedisExpectationsMet helper function used to ensure mock expectations were met
func assertRedisExpectationsMet(t *testing.T, mock redismock.ClientMock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func constructExpectedKey(orgID types.OrgID, clusterName types.ClusterName, requestID types.RequestID) string {
	return fmt.Sprintf("organization:%d:cluster:%s:request:%s",
		int(testdata.OrgID),
		string(testdata.ClusterName),
		string(testdata.RequestID1))
}

// TestNewRedisClient checks if it is possible to construct Redis client
func TestNewRedisClient(t *testing.T) {
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration)

	// check results
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

// TestNewDummyRedisClient checks if it is possible to construct Redis
// client structure useful for testing
func TestNewDummyRedisClient(t *testing.T) {
	// configuration where Redis endpoint is set to empty string
	configuration := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "",
			RedisDatabase:       0,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration)

	// check results
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

// TestNewRedisClientDBIndexOutOfRange checks if Redis client
// constructor checks for incorrect database index
func TestNewRedisClientDBIndexOutOfRange(t *testing.T) {
	configuration1 := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "localhost:12345",
			RedisDatabase:       -1,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration1)

	// check results
	assert.Nil(t, client)
	assert.Error(t, err)

	configuration2 := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "localhost:12345",
			RedisDatabase:       16,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err = storage.NewRedisStorage(configuration2)

	// check results
	assert.Nil(t, client)
	assert.Error(t, err)
}

// TestRedisWriteReportForCluster checks the method WriteReportForCluster
func TestRedisWriteReportForCluster(t *testing.T) {
	client, server := getMockRedis(t)

	// Redis client needs to be initialized
	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	// it is expected that key will be set with given expiration period
	expectedKey := constructExpectedKey(testdata.OrgID, testdata.ClusterName, testdata.RequestID1)
	server.ExpectSet(expectedKey, "", client.Expiration).SetVal("OK")

	timestamp := time.Now()

	expectedData := ctypes.SimplifiedReport{
		OrgID:              int(testdata.OrgID),
		RequestID:          string(testdata.RequestID1),
		ClusterID:          string(testdata.ClusterName),
		ReceivedTimestamp:  testdata.LastCheckedAt,
		ProcessedTimestamp: timestamp,
		RuleHitsCSV:        "ccx_rules_ocp.external.rules.node_installer_degraded|ek1,test.rule2|ek2,test.rule3|ek3",
	}

	expectedReportKey := expectedKey + ":reports"
	server.ExpectHSet(expectedReportKey, expectedData).SetVal(1)
	server.ExpectExpire(expectedReportKey, client.Expiration).SetVal(true)

	err = client.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules, testdata.Report3RulesParsed,
		testdata.LastCheckedAt, testdata.LastCheckedAt, timestamp,
		testdata.KafkaOffset, testdata.RequestID1)

	assert.NoError(t, err)
	assertRedisExpectationsMet(t, server)
}

// TestWriteEmptyReport checks the method WriteReportForCluster for empty rule hits
func TestWriteEmptyReport(t *testing.T) {
	client, server := getMockRedis(t)

	// Redis client needs to be initialized
	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	// it is expected that key will be set with given expiration period
	expectedKey := constructExpectedKey(testdata.OrgID, testdata.ClusterName, testdata.RequestID1)
	server.ExpectSet(expectedKey, "", client.Expiration).SetVal("OK")

	timestamp := time.Now()

	expectedData := ctypes.SimplifiedReport{
		OrgID:              int(testdata.OrgID),
		RequestID:          string(testdata.RequestID1),
		ClusterID:          string(testdata.ClusterName),
		ReceivedTimestamp:  testdata.LastCheckedAt,
		ProcessedTimestamp: timestamp,
		RuleHitsCSV:        "",
	}

	expectedReportKey := expectedKey + ":reports"
	server.ExpectHSet(expectedReportKey, expectedData).SetVal(1)
	server.ExpectExpire(expectedReportKey, client.Expiration).SetVal(true)

	err = client.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules, []types.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, timestamp,
		testdata.KafkaOffset, testdata.RequestID1)

	assert.NoError(t, err)
	assertRedisExpectationsMet(t, server)
}

// TestRedisWriteReportForClusterErrorHandling1 checks how the method
// WriteReportForCluster handles errors
func TestRedisWriteReportForClusterErrorHandling1(t *testing.T) {
	errorMessage := "key set error!"

	client, server := getMockRedis(t)

	// Redis client needs to be initialized
	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	// it is expected that key will be set with given expiration period
	expectedKey := constructExpectedKey(testdata.OrgID, testdata.ClusterName, testdata.RequestID1)
	server.ExpectSet(expectedKey, "", client.Expiration).SetErr(errors.New(errorMessage))

	timestamp := time.Now()

	err = client.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules, []types.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, timestamp,
		testdata.KafkaOffset, testdata.RequestID1)

	assert.Error(t, err)
	assert.EqualError(t, err, errorMessage)
	assertRedisExpectationsMet(t, server)
}

// TestRedisWriteReportForClusterErrorHandling2 checks how the method
// WriteReportForCluster handles errors
func TestRedisWriteReportForClusterErrorHandling2(t *testing.T) {
	errorMessage := "hash set error!"

	client, server := getMockRedis(t)

	// Redis client needs to be initialized
	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	// it is expected that key will be set with given expiration period
	expectedKey := constructExpectedKey(testdata.OrgID, testdata.ClusterName, testdata.RequestID1)
	server.ExpectSet(expectedKey, "", client.Expiration).SetVal("OK")

	timestamp := time.Now()

	expectedData := ctypes.SimplifiedReport{
		OrgID:              int(testdata.OrgID),
		RequestID:          string(testdata.RequestID1),
		ClusterID:          string(testdata.ClusterName),
		ReceivedTimestamp:  testdata.LastCheckedAt,
		ProcessedTimestamp: timestamp,
		RuleHitsCSV:        "",
	}

	expectedReportKey := expectedKey + ":reports"
	server.ExpectHSet(expectedReportKey, expectedData).SetErr(errors.New(errorMessage))

	err = client.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules, []types.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, timestamp,
		testdata.KafkaOffset, testdata.RequestID1)

	assert.Error(t, err)
	assert.EqualError(t, err, errorMessage)
	assertRedisExpectationsMet(t, server)
}

// TestRedisWriteReportForClusterErrorHandling3 checks how the method
// WriteReportForCluster handles errors
func TestRedisWriteReportForClusterErrorHandling3(t *testing.T) {
	errorMessage := "expiration set error!"

	client, server := getMockRedis(t)

	// Redis client needs to be initialized
	err := client.Init()
	if err != nil {
		t.Fatal(err)
	}

	// it is expected that key will be set with given expiration period
	expectedKey := constructExpectedKey(testdata.OrgID, testdata.ClusterName, testdata.RequestID1)
	server.ExpectSet(expectedKey, "", client.Expiration).SetVal("OK")

	timestamp := time.Now()

	expectedData := ctypes.SimplifiedReport{
		OrgID:              int(testdata.OrgID),
		RequestID:          string(testdata.RequestID1),
		ClusterID:          string(testdata.ClusterName),
		ReceivedTimestamp:  testdata.LastCheckedAt,
		ProcessedTimestamp: timestamp,
		RuleHitsCSV:        "",
	}

	expectedReportKey := expectedKey + ":reports"
	server.ExpectHSet(expectedReportKey, expectedData).SetVal(1)
	server.ExpectExpire(expectedReportKey, client.Expiration).SetErr(errors.New(errorMessage))

	err = client.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules, []types.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, timestamp,
		testdata.KafkaOffset, testdata.RequestID1)

	assert.Error(t, err)
	assert.EqualError(t, err, errorMessage)
	assertRedisExpectationsMet(t, server)
}
