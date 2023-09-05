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
	"strings"
	"time"

	"errors"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-content-service/content"
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

func constructExpectedKey(_ types.OrgID, _ types.ClusterName, _ types.RequestID) string {
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

// TestCloseRedis1 checks if it is possible to close initialized Redis client
func TestCloseRedis1(t *testing.T) {
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration)

	// check results
	assert.NotNil(t, client)
	assert.NoError(t, err)

	// try to close the client
	err = client.Close()
	assert.NoError(t, err)
}

// TestCloseRedis2 checks if it is possible to close unitialized Redis client
func TestCloseRedis2(t *testing.T) {
	client := storage.RedisStorage{}

	// check results
	assert.NotNil(t, client)

	// try to close the client
	err := client.Close()
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
		testdata.RequestID1)

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
		testdata.RequestID1)

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
		testdata.RequestID1)

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
		testdata.RequestID1)

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
		testdata.RequestID1)

	assert.Error(t, err)
	assert.EqualError(t, err, errorMessage)
	assertRedisExpectationsMet(t, server)
}

// Don't decrease code coverage by non-functional and not covered code.

// TestRedisStorageEmptyMethods1 calls empty methods that just needs to be
// defined in order for RedisStorage to satisfy Storage interface.
func TestRedisStorageEmptyMethods1(_ *testing.T) {
	RedisStorage := storage.RedisStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")

	_, _ = RedisStorage.ListOfOrgs()
	_, _ = RedisStorage.ListOfClustersForOrg(orgID, time.Now())
	_, _ = RedisStorage.ReadOrgIDsForClusters([]types.ClusterName{clusterName})
	_, _ = RedisStorage.ReportsCount()
	_, _ = RedisStorage.ReadReportsForClusters([]types.ClusterName{clusterName})
	_, _ = RedisStorage.DoesClusterExist(clusterName)

	_, _, _, _, _ = RedisStorage.ReadReportForCluster(orgID, clusterName)
	_, _ = RedisStorage.ReadReportInfoForCluster(orgID, clusterName)
	_, _ = RedisStorage.ReadClusterVersionsForClusterList(orgID, []string{string(clusterName)})
	_, _, _ = RedisStorage.ReadReportForClusterByClusterName(clusterName)
	_ = RedisStorage.DeleteReportsForOrg(orgID)
	_ = RedisStorage.DeleteReportsForCluster(clusterName)

	_ = RedisStorage.MigrateToLatest()
	_ = RedisStorage.GetConnection()
	RedisStorage.PrintRuleDisableDebugInfo()
	_ = RedisStorage.GetDBDriverType()
	_ = RedisStorage.WriteConsumerError(nil, nil)
}

// TestRedisStorageEmptyMethods2 calls empty methods that just needs to be
// defined in order for RedisStorage to satisfy Storage interface.
func TestRedisStorageEmptyMethods2(_ *testing.T) {
	RedisStorage := storage.RedisStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")
	ruleID := types.RuleID("")
	errorKey := types.ErrorKey("")
	userID := types.UserID("")
	ruleSelector := types.RuleSelector("")

	_, _, _ = RedisStorage.ReadDisabledRule(orgID, ruleID, errorKey)
	_ = RedisStorage.VoteOnRule(clusterName, ruleID, errorKey, orgID, userID, 0, "some message")
	_ = RedisStorage.AddOrUpdateFeedbackOnRule(clusterName, ruleID, errorKey, orgID, userID, "")
	_ = RedisStorage.AddFeedbackOnRuleDisable(clusterName, ruleID, errorKey, orgID, userID, "")
	_, _ = RedisStorage.GetUserFeedbackOnRuleDisable(clusterName, ruleID, errorKey, userID)
	_, _ = RedisStorage.GetUserFeedbackOnRule(clusterName, ruleID, errorKey, userID)
	_ = RedisStorage.LoadRuleContent(content.RuleContentDirectory{})
	_, _ = RedisStorage.GetRuleByID(ruleID)
	_, _ = RedisStorage.GetOrgIDByClusterID(clusterName)
	_, _ = RedisStorage.ListOfSystemWideDisabledRules(orgID)
	_, _ = RedisStorage.ListOfClustersForOrgSpecificRule(orgID, ruleSelector, nil)
	_, _ = RedisStorage.ListOfClustersForOrgSpecificRule(orgID, ruleSelector, []string{"a"})
	_, _ = RedisStorage.ReadRecommendationsForClusters([]string{}, types.OrgID(1))
	_, _ = RedisStorage.ReadClusterListRecommendations([]string{}, types.OrgID(1))
	_, _ = RedisStorage.ListOfDisabledClusters(orgID, ruleID, errorKey)
	_, _ = RedisStorage.ReadSingleRuleTemplateData(orgID, clusterName, ruleID, errorKey)
}

// TestRedisStorageEmptyMethods3 calls empty methods that just needs to be
// defined in order for RedisStorage to satisfy Storage interface.
func TestRedisStorageEmptyMethods3(_ *testing.T) {
	RedisStorage := storage.RedisStorage{}
	orgID := types.OrgID(1)
	clusterName := types.ClusterName("")
	rule := types.Rule{}
	ruleID := types.RuleID("")
	ruleErrorKey := types.RuleErrorKey{}
	errorKey := types.ErrorKey("")
	userID := types.UserID("")

	_ = RedisStorage.DisableRuleSystemWide(orgID, ruleID, errorKey, "justification#1")
	_ = RedisStorage.EnableRuleSystemWide(orgID, ruleID, errorKey)
	_ = RedisStorage.UpdateDisabledRuleJustification(orgID, ruleID, errorKey, "justification#2")
	_ = RedisStorage.WriteReportInfoForCluster(orgID, clusterName, nil, time.Time{})
	_ = RedisStorage.WriteRecommendationsForCluster(orgID, clusterName, "", types.Timestamp(""))
	_ = RedisStorage.CreateRule(rule)
	_ = RedisStorage.DeleteRule(ruleID)
	_ = RedisStorage.CreateRuleErrorKey(ruleErrorKey)
	_ = RedisStorage.DeleteRuleErrorKey(ruleID, errorKey)
	_ = RedisStorage.ToggleRuleForCluster(clusterName, ruleID, errorKey, orgID, storage.RuleToggle(0))
	_ = RedisStorage.DeleteFromRuleClusterToggle(clusterName, ruleID)
	_ = RedisStorage.RateOnRule(orgID, ruleID, errorKey, types.UserVote(1))
	_, _ = RedisStorage.GetFromClusterRuleToggle(clusterName, ruleID)
	_, _ = RedisStorage.GetTogglesForRules(clusterName, nil, orgID)
	_, _ = RedisStorage.GetUserFeedbackOnRules(clusterName, nil, userID)
	_, _ = RedisStorage.GetUserDisableFeedbackOnRules(clusterName, nil, userID)
	_, _ = RedisStorage.GetRuleWithContent(ruleID, errorKey)
	_, _ = RedisStorage.ListOfDisabledRules(orgID)
	_, _ = RedisStorage.GetRuleRating(orgID, types.RuleSelector(""))
	_, _ = RedisStorage.ListOfReasons(userID)
	_, _ = RedisStorage.ListOfDisabledRulesForClusters([]string{""}, orgID)
}

func Test_GetRuleHitsCSV_CCXDEV_11329_Reproducer(t *testing.T) {
	var reportItems []types.ReportItem
	_ = copy(reportItems, testdata.Report3RulesParsed)

	// add .report suffix to rule modules
	for i := range reportItems {
		ruleModule := reportItems[i].Module
		ruleModule = ruleModule + storage.ReportSuffix
		reportItems[i].Module = ruleModule
	}

	ruleHitsParsed := storage.GetRuleHitsCSV(reportItems)
	// split the CSV and iterate over the rule hits
	ruleHitsSplit := strings.Split(ruleHitsParsed, ",")
	for _, ruleHit := range ruleHitsSplit {
		ruleModule := strings.Split(ruleHit, "|")[0]
		ruleModuleSplit := strings.Split(ruleModule, ".")
		ruleModuleSuffix := ruleModuleSplit[len(ruleModuleSplit)-1]
		// rule module must not include the .report suffix after parsing
		assert.NotEqual(t, ruleModuleSuffix, "report")
	}
}
