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
	ruleConfigOne       = content.GlobalRuleConfig{Impact: map[string]int{"One": 1}}
	ruleContentActiveOK = content.RuleContentDirectory{
		Config: ruleConfigOne,
		Rules: map[string]content.RuleContent{
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
							Impact:      "One",
							Likelihood:  1,
							PublishDate: "1970-01-01 00:00:00",
							Status:      "active",
						},
					},
				},
			},
		},
	}
	ruleContentInactiveOK = content.RuleContentDirectory{
		Config: ruleConfigOne,
		Rules: map[string]content.RuleContent{
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
							Impact:      "One",
							Likelihood:  1,
							PublishDate: "1970-01-01 00:00:00",
							Status:      "inactive",
						},
					},
				},
			},
		},
	}
	ruleContentBadStatus = content.RuleContentDirectory{
		Config: ruleConfigOne,
		Rules: map[string]content.RuleContent{
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
							Impact:      "One",
							Likelihood:  1,
							PublishDate: "1970-01-01 00:00:00",
							Status:      "bad",
						},
					},
				},
			},
		},
	}
	ruleContentNull = content.RuleContentDirectory{
		Config: ruleConfigOne,
		Rules: map[string]content.RuleContent{
			"rc": content.RuleContent{},
		},
	}
	ruleContentExample1 = content.RuleContentDirectory{
		Config: ruleConfigOne,
		Rules: map[string]content.RuleContent{
			"rc": content.RuleContent{
				Summary:    []byte("summary"),
				Reason:     []byte("reason"),
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
							Impact:      "One",
							Likelihood:  1,
							PublishDate: "1970-01-01 00:00:00",
							Status:      "active",
						},
					},
				},
			},
		},
	}
)

func mustWriteReport3Rules(t *testing.T, mockStorage storage.Storage) {
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)
}

func TestDBStorageLoadRuleContentActiveOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentActiveOK)
	helpers.FailOnError(t, err)
}

func TestDBStorageLoadRuleContentDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentActiveOK)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageLoadRuleContentInsertIntoRuleErrorKeyError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

	// create a table with a bad type
	_, err := connection.Exec(`
		DROP TABLE rule_error_key;
		CREATE TABLE rule_error_key (
			"error_key"     INTEGER NOT NULL CHECK(typeof("error_key") = 'integer'),
			"rule_module"   VARCHAR NOT NULL REFERENCES rule(module),
			"condition"     VARCHAR NOT NULL,
			"description"   VARCHAR NOT NULL,
			"impact"        INTEGER NOT NULL,
			"likelihood"    INTEGER NOT NULL,
			"publish_date"  TIMESTAMP NOT NULL,
			"active"        BOOLEAN NOT NULL,
			"generic"       VARCHAR NOT NULL,

			PRIMARY KEY("error_key", "rule_module")
		)
	`)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	assert.EqualError(t, err, "CHECK constraint failed: rule_error_key")
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
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentInactiveOK)
	helpers.FailOnError(t, err)
}

func TestDBStorageLoadRuleContentNull(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentNull)
	assert.EqualError(t, err, "NOT NULL constraint failed: rule.summary")
}

func TestDBStorageLoadRuleContentBadStatus(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentBadStatus)
	assert.EqualError(t, err, "invalid rule error key status: 'bad'")
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
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageGetContentForRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(ruleContentExample1)
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
			Reason:       "reason",
			Resolution:   "resolution",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    1,
			RiskOfChange: 0,
			TemplateData: nil,
		},
	}, res)
}

func TestDBStorageGetContentForMultipleRulesOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
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

	// TODO: check risk of change when it will be returned correctly
	// total risk is `(impact + likelihood) / 2`
	// db doesn't and shouldn't guarantee order so we're using ElementsMatch
	assert.ElementsMatch(t, []types.RuleContentResponse{
		{
			ErrorKey:     "ek1",
			RuleModule:   "test.rule1",
			Description:  "rule 1 description",
			Generic:      "rule 1 details",
			Reason:       "rule 1 reason",
			Resolution:   "rule 1 resolution",
			CreatedAt:    "1970-01-01T00:00:00Z",
			TotalRisk:    3,
			RiskOfChange: 0,
			TemplateData: nil,
		},
		{
			ErrorKey:     "ek2",
			RuleModule:   "test.rule2",
			Description:  "rule 2 description",
			Generic:      "rule 2 details",
			Reason:       "rule 2 reason",
			Resolution:   "rule 2 resolution",
			CreatedAt:    "1970-01-02T00:00:00Z",
			TotalRisk:    4,
			RiskOfChange: 0,
			TemplateData: nil,
		},
		{
			ErrorKey:     "ek3",
			RuleModule:   "test.rule3",
			Description:  "rule 3 description",
			Generic:      "rule 3 details",
			Reason:       "rule 3 reason",
			Resolution:   "rule 3 resolution",
			CreatedAt:    "1970-01-03T00:00:00Z",
			TotalRisk:    2,
			RiskOfChange: 0,
			TemplateData: nil,
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
		"reason",
		"resolution",
		"publish_date",
		"impact",
		"likelihood",
	}

	values := make([]driver.Value, 0)
	for _, val := range columns {
		values = append(values, val)
	}

	// return bad values
	expects.ExpectQuery("SELECT (.*) FROM rule (.*) rule_error_key").WillReturnRows(
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

func TestDBStorageGetContentForRulesRowsError(t *testing.T) {
	const rowErr = "row error"

	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	columns := []string{
		"error_key",
		"rule_module",
		"description",
		"generic",
		"reason",
		"resolution",
		"publish_date",
		"impact",
		"likelihood",
	}

	values := []driver.Value{
		"ek", "rule_module", "desc", "generic", "reason", "resolution", 0, 0, 0,
	}

	// return bad values
	expects.ExpectQuery("SELECT (.*) FROM rule (.*) rule_error_key").WillReturnRows(
		sqlmock.NewRows(columns).AddRow(values...).RowError(0, fmt.Errorf(rowErr)),
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), rowErr)
	assert.Contains(t, buf.String(), "SQL rows error while retrieving content for rules")
}

func TestDBStorageVoteOnRule(t *testing.T) {
	for _, vote := range []storage.UserVote{
		storage.UserVoteDislike, storage.UserVoteLike, storage.UserVoteNone,
	} {
		mockStorage := helpers.MustGetMockStorage(t, true)

		mustWriteReport3Rules(t, mockStorage)

		helpers.FailOnError(t, mockStorage.VoteOnRule(
			testdata.ClusterName, testdata.Rule1ID, testdata.UserID, vote,
		))

		feedback, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
		helpers.FailOnError(t, err)

		assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
		assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
		assert.Equal(t, testdata.UserID, feedback.UserID)
		assert.Equal(t, "", feedback.Message)
		assert.Equal(t, vote, feedback.UserVote)

		helpers.FailOnError(t, mockStorage.Close())
	}
}

func TestDBStorageVoteOnRule_NoCluster(t *testing.T) {
	for _, vote := range []storage.UserVote{
		storage.UserVoteDislike, storage.UserVoteLike, storage.UserVoteNone,
	} {
		mockStorage := helpers.MustGetMockStorage(t, true)

		err := mockStorage.VoteOnRule(
			testdata.ClusterName, testdata.Rule1ID, testdata.UserID, vote,
		)
		assert.EqualError(t, err, "FOREIGN KEY constraint failed")
	}
}

func TestDBStorageVoteOnRule_NoRule(t *testing.T) {
	for _, vote := range []storage.UserVote{
		storage.UserVoteDislike, storage.UserVoteLike, storage.UserVoteNone,
	} {
		mockStorage := helpers.MustGetMockStorage(t, true)

		err := mockStorage.WriteReportForCluster(
			testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
		)
		helpers.FailOnError(t, err)

		err = mockStorage.VoteOnRule(
			testdata.ClusterName, testdata.Rule1ID, testdata.UserID, vote,
		)
		assert.EqualError(t, err, "FOREIGN KEY constraint failed")
	}
}

func TestDBStorageChangeVote(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, storage.UserVoteLike,
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, storage.UserVoteDislike,
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, "", feedback.Message)
	assert.Equal(t, storage.UserVoteDislike, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageTextFeedback(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, "test feedback",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, "test feedback", feedback.Message)
	assert.Equal(t, storage.UserVoteNone, feedback.UserVote)
}

func TestDBStorageFeedbackChangeMessage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, "message1",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, "message2",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
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
	assert.EqualError(t, err, "sql: database is closed")
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

	err = mockStorage.Init()
	helpers.FailOnError(t, err)

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

	// TODO: uncomment when issues upthere resolved
	//assert.Contains(t, buf.String(), errStr)
}

func TestDBStorage_CreateRule(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	helpers.FailOnError(t, err)
}

func TestDBStorage_CreateRule_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorage_CreateRuleErrorKey(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	helpers.FailOnError(t, err)

	err = mockStorage.CreateRuleErrorKey(types.RuleErrorKey{
		ErrorKey:    "error_key",
		RuleModule:  "module",
		Condition:   "condition",
		Description: "description",
		Impact:      1,
		Likelihood:  2,
		PublishDate: testdata.LastCheckedAt,
		Active:      true,
		Generic:     "generic",
	})
	helpers.FailOnError(t, err)
}

func TestDBStorage_CreateRuleErrorKey_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	helpers.FailOnError(t, err)

	helpers.MustCloseStorage(t, mockStorage)

	err = mockStorage.CreateRuleErrorKey(types.RuleErrorKey{
		ErrorKey:    "error_key",
		RuleModule:  "rule_module",
		Condition:   "condition",
		Description: "description",
		Impact:      1,
		Likelihood:  2,
		PublishDate: testdata.LastCheckedAt,
		Active:      true,
		Generic:     "generic",
	})
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorage_DeleteRule(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.CreateRule(types.Rule{
		Module: "module",
	})
	helpers.FailOnError(t, err)

	err = mockStorage.DeleteRule("module")
	helpers.FailOnError(t, err)
}

func TestDBStorage_DeleteRule_NotFound(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.DeleteRule("module")
	assert.EqualError(t, err, "Item with ID module was not found in the storage")
}

func TestDBStorage_DeleteRule_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.DeleteRule("module")
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorage_DeleteRuleErrorKey(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	helpers.FailOnError(t, err)

	err = mockStorage.CreateRuleErrorKey(types.RuleErrorKey{
		ErrorKey:    "error_key",
		RuleModule:  "module",
		Condition:   "condition",
		Description: "description",
		Impact:      1,
		Likelihood:  2,
		PublishDate: testdata.LastCheckedAt,
		Active:      true,
		Generic:     "generic",
	})
	helpers.FailOnError(t, err)

	err = mockStorage.DeleteRuleErrorKey("module", "error_key")
	helpers.FailOnError(t, err)
}

func TestDBStorage_DeleteRuleErrorKey_NotFound(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.DeleteRuleErrorKey("module", "error_key")
	assert.EqualError(t, err, "Item with ID module/error_key was not found in the storage")
}

func TestDBStorage_DeleteRuleErrorKey_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)

	err := mockStorage.CreateRule(types.Rule{
		Module:     "module",
		Name:       "name",
		Summary:    "summary",
		Reason:     "reason",
		Resolution: "resolution",
		MoreInfo:   "more_info",
	})
	helpers.FailOnError(t, err)

	err = mockStorage.CreateRuleErrorKey(types.RuleErrorKey{
		ErrorKey:    "error_key",
		RuleModule:  "module",
		Condition:   "condition",
		Description: "description",
		Impact:      1,
		Likelihood:  2,
		PublishDate: testdata.LastCheckedAt,
		Active:      true,
		Generic:     "generic",
	})
	helpers.FailOnError(t, err)

	helpers.MustCloseStorage(t, mockStorage)

	err = mockStorage.DeleteRuleErrorKey("module", "error_key")
	assert.EqualError(t, err, "sql: database is closed")
}
