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

package storage_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/stretchr/testify/assert"
)

var (
	ruleContentActiveOK = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": {
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "active",
					},
				},
			},
		},
	}
	ruleContentInactiveOK = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": {
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "inactive",
					},
				},
			},
		},
	}
	ruleContentBadStatus = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("reason"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": {
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "bad",
					},
				},
			},
		},
	}
	ruleContentNull = content.RuleContentDirectory{
		"rc": content.RuleContent{},
	}
	ruleContentExample1 = content.RuleContentDirectory{
		"rc": content.RuleContent{
			Summary:    []byte("summary"),
			Reason:     []byte("summary"),
			Resolution: []byte("resolution"),
			MoreInfo:   []byte("more info"),
			Plugin: content.RulePluginInfo{
				Name:         "test rule",
				NodeID:       string(testClusterName),
				ProductCode:  "product code",
				PythonModule: string(testRuleID),
			},
			ErrorKeys: map[string]content.RuleErrorKeyContent{
				"ek": {
					Generic: []byte("generic"),
					Metadata: content.ErrorKeyMetadata{
						Condition:   "condition",
						Description: "description",
						Impact:      1,
						Likelihood:  1,
						PublishDate: "1970-01-01 00:00:00",
						Status:      "active",
					},
				},
			},
		},
	}
)

func TestDBStorageLoadRuleContentActiveOK(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentActiveOK)
	helpers.FailOnError(t, err)
}

func TestDBStorageLoadRuleContentDBError(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentActiveOK)
	if err == nil {
		t.Fatalf("error expected, got %+v", err)
	}
}

func TestDBStorageLoadRuleContentDeleteDBError(t *testing.T) {
	const errorStr = "delete error"
	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()
	expects.ExpectExec("DELETE FROM rule_error_key").
		WillReturnError(fmt.Errorf(errorStr))

	err := mockStorage.LoadRuleContent(ruleContentActiveOK)
	assert.EqualError(t, err, errorStr)
}

func TestDBStorageLoadRuleContentCommitDBError(t *testing.T) {
	const errorStr = "commit error"
	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()
	expects.ExpectExec("DELETE FROM rule_error_key").WillReturnResult(driver.ResultNoRows)
	expects.ExpectCommit().WillReturnError(fmt.Errorf(errorStr))

	err := mockStorage.LoadRuleContent(content.RuleContentDirectory{})
	assert.EqualError(t, err, errorStr)
}

func TestDBStorageLoadRuleContentInactiveOK(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentInactiveOK)
	helpers.FailOnError(t, err)
}

func TestDBStorageLoadRuleContentNull(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentNull)
	if err == nil || err.Error() != "NOT NULL constraint failed: rule.summary" {
		t.Fatal(err)
	}
}

func TestDBStorageLoadRuleContentBadStatus(t *testing.T) {
	s := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, s)
	dbStorage := s.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentBadStatus)
	if err == nil || err.Error() != "invalid rule error key status: 'bad'" {
		t.Fatal(err)
	}
}

func TestDBStorageGetContentForRulesEmpty(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules:     nil,
		SkippedRules: nil,
		PassedRules:  nil,
		TotalCount:   0,
	})
	helpers.FailOnError(t, err)

	assert.Empty(t, res)
}

func TestDBStorageGetContentForRulesDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules:     nil,
		SkippedRules: nil,
		PassedRules:  nil,
		TotalCount:   0,
	})
	if err == nil {
		t.Fatalf("error expected, got %+v", err)
	}
}

func TestDBStorageGetContentForRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(ruleContentExample1)
	helpers.FailOnError(t, err)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules: []types.RuleOnReport{
			{
				Module:   string(testRuleID),
				ErrorKey: "ek",
			},
		},
		TotalCount: 1,
	})
	helpers.FailOnError(t, err)

	assert.Equal(t, []types.RuleContentResponse{
		{
			ErrorKey:     "ek",
			RuleModule:   string(testRuleID),
			Description:  "description",
			Generic:      "generic",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    1,
			RiskOfChange: 0,
		},
	}, res)
}

func TestDBStorageGetContentForMultipleRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := dbStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	res, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules: []types.RuleOnReport{
			{
				Module:   "test.rule1.report",
				ErrorKey: "ek1",
			},
			{
				Module:   "test.rule2.report",
				ErrorKey: "ek2",
			},
			{
				Module:   "test.rule3.report",
				ErrorKey: "ek3",
			},
		},
		TotalCount: 3,
	})
	helpers.FailOnError(t, err)

	assert.Len(t, res, 3)

	// db doesn't and shouldn't guarantee order
	sort.Slice(res, func(firstIndex, secondIndex int) bool {
		return res[firstIndex].ErrorKey < res[secondIndex].ErrorKey
	})

	// TODO: check risk of change when it will be returned correctly
	// total risk is `(impact + likelihood) / 2`
	assert.Equal(t, []types.RuleContentResponse{
		{
			ErrorKey:     "ek1",
			RuleModule:   "test.rule1",
			Description:  "rule 1 description",
			Generic:      "rule 1 details",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    3,
			RiskOfChange: 0,
		},
		{
			ErrorKey:     "ek2",
			RuleModule:   "test.rule2",
			Description:  "rule 2 description",
			Generic:      "rule 2 details",
			CreatedAt:    "1970-01-02T00:00:00Z",
			TotalRisk:    4,
			RiskOfChange: 0,
		},
		{
			ErrorKey:     "ek3",
			RuleModule:   "test.rule3",
			Description:  "rule 3 description",
			Generic:      "rule 3 details",
			CreatedAt:    "1970-01-03T00:00:00Z",
			TotalRisk:    2,
			RiskOfChange: 0,
		},
	}, res)
}

func TestDBStorageGetContentForRulesScanError(t *testing.T) {
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	columns := []string{
		"error_key",
		"rule_module",
		"description",
		"generic",
		"publish_date",
		"impact",
		"likelihood",
	}

	values := make([]driver.Value, 0)
	for _, val := range columns {
		values = append(values, val)
	}

	// return bad values
	expects.ExpectQuery("SELECT .* FROM rule_error_key").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(values...),
	)

	_, err := mockStorage.GetContentForRules(types.ReportRules{
		HitRules: []types.RuleOnReport{
			{
				Module:   "rule_module",
				ErrorKey: "error_key",
			},
		},
		TotalCount: 1,
	})
	helpers.FailOnError(t, err)

	assert.Regexp(t, "converting driver.Value type .+ to .*", buf.String())
}

func TestDBStorageVoteOnRule(t *testing.T) {
	for _, vote := range []storage.UserVote{
		storage.UserVoteDislike, storage.UserVoteLike, storage.UserVoteNone,
	} {
		mockStorage := helpers.MustGetMockStorage(t, true)

		helpers.FailOnError(t, mockStorage.VoteOnRule(
			testClusterName, testRuleID, testUserID, vote,
		))

		feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
		helpers.FailOnError(t, err)

		assert.Equal(t, testClusterName, feedback.ClusterID)
		assert.Equal(t, testRuleID, feedback.RuleID)
		assert.Equal(t, testUserID, feedback.UserID)
		assert.Equal(t, "", feedback.Message)
		assert.Equal(t, vote, feedback.UserVote)

		helpers.FailOnError(t, mockStorage.Close())
	}
}

func TestDBStorageChangeVote(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testClusterName, testRuleID, testUserID, storage.UserVoteLike,
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testClusterName, testRuleID, testUserID, storage.UserVoteDislike,
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "", feedback.Message)
	assert.Equal(t, storage.UserVoteDislike, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageTextFeedback(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "test feedback",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "test feedback", feedback.Message)
	assert.Equal(t, storage.UserVoteNone, feedback.UserVote)
}

func TestDBStorageFeedbackChangeMessage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "message1",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "message2",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "message2", feedback.Message)
	assert.Equal(t, storage.UserVoteNone, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageFeedbackErrorItemNotFound(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	if _, ok := err.(*storage.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

func TestDBStorageFeedbackErrorDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	if err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("expected sql database is closed error, got %T, %+v", err, err)
	}
}

func TestDBStorageVoteOnRuleDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.VoteOnRule(testClusterName, testRuleID, testUserID, storage.UserVoteNone)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageVoteOnRuleUnsupportedDriverError(t *testing.T) {
	connection, err := sql.Open("sqlite3", ":memory:")
	helpers.FailOnError(t, err)

	mockStorage := storage.NewFromConnection(connection, -1)
	defer helpers.MustCloseStorage(t, mockStorage)

	err = mockStorage.VoteOnRule(testClusterName, testRuleID, testUserID, storage.UserVoteNone)
	assert.EqualError(t, err, "DB driver -1 is not supported")
}

func TestDBStorageVoteOnRuleDBExecError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)
	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

	// create a table with a bad type
	_, err := connection.Exec(`
		CREATE TABLE cluster_rule_user_feedback (
			cluster_id INTEGER NOT NULL CHECK(typeof(cluster_id) = 'integer'),
			rule_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			message INTEGER NOT NULL,
			user_vote INTEGER NOT NULL,
			added_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,

			PRIMARY KEY(cluster_id, rule_id, user_id)
		)
	`)
	helpers.FailOnError(t, err)

	err = mockStorage.VoteOnRule("non int", testRuleID, testUserID, storage.UserVoteNone)
	assert.EqualError(t, err, "CHECK constraint failed: cluster_rule_user_feedback")
}

func TestDBStorageVoteOnRuleDBCloseError(t *testing.T) {
	// TODO: seems to be not coverable because of the bug in golang
	// related issues:
	// https://github.com/DATA-DOG/go-sqlmock/issues/185
	// https://github.com/golang/go/issues/37973

	const errStr = "close error"

	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectPrepare("INSERT").
		WillBeClosed().
		WillReturnCloseError(fmt.Errorf(errStr)).
		ExpectExec().
		WillReturnResult(driver.ResultNoRows)

	err := mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testUserID, storage.UserVoteNone)
	helpers.FailOnError(t, err)

	assert.Contains(t, errStr, buf.String())
}
