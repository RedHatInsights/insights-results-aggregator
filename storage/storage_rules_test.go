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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func mustWriteReport3Rules(t *testing.T, mockStorage storage.Storage) {
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorageToggleRule(t *testing.T) {
	for _, state := range []storage.RuleToggle{
		storage.RuleToggleDisable, storage.RuleToggleEnable,
	} {
		func(state storage.RuleToggle) {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()

			mustWriteReport3Rules(t, mockStorage)

			helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
				testdata.ClusterName, testdata.Rule1ID, testdata.UserID, state,
			))

			_, err := mockStorage.GetFromClusterRuleToggle(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
			helpers.FailOnError(t, err)
		}(state)
	}
}

func TestDBStorageGetTogglesForRules_NoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, nil, testdata.UserID,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorageGetTogglesForRules_AllRulesEnabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, testdata.RuleOnReportResponses, testdata.UserID,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorageGetTogglesForRules_OneRuleDisabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, storage.RuleToggleDisable,
	))

	result, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, testdata.RuleOnReportResponses, testdata.UserID,
	)

	helpers.FailOnError(t, err)

	assert.Equal(
		t,
		map[types.RuleID]bool{
			testdata.Rule1ID: true,
		},
		result,
	)
}

func TestDBStorageToggleRuleAndGet(t *testing.T) {
	for _, state := range []storage.RuleToggle{
		storage.RuleToggleDisable, storage.RuleToggleEnable,
	} {
		func(state storage.RuleToggle) {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()

			mustWriteReport3Rules(t, mockStorage)

			helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
				testdata.ClusterName, testdata.Rule1ID, testdata.UserID, state,
			))

			toggledRule, err := mockStorage.GetFromClusterRuleToggle(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testdata.ClusterName, toggledRule.ClusterID)
			assert.Equal(t, testdata.Rule1ID, toggledRule.RuleID)
			assert.Equal(t, testdata.UserID, toggledRule.UserID)
			assert.Equal(t, state, toggledRule.Disabled)
			if toggledRule.Disabled == storage.RuleToggleDisable {
				assert.Equal(t, sql.NullTime{}, toggledRule.EnabledAt)
			} else {
				assert.Equal(t, sql.NullTime{}, toggledRule.DisabledAt)
			}

			helpers.FailOnError(t, mockStorage.Close())
		}(state)
	}
}

// TODO: make it work with the new arch
//func TestDBStorageToggleRulesAndList(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mustWriteReport3Rules(t, mockStorage)
//
//	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
//		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, storage.RuleToggleDisable,
//	))
//
//	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
//		testdata.ClusterName, testdata.Rule2ID, testdata.UserID, storage.RuleToggleDisable,
//	))
//
//	toggledRules, err := mockStorage.ListDisabledRulesForCluster(testdata.ClusterName, testdata.UserID)
//	helpers.FailOnError(t, err)
//
//	assert.Len(t, toggledRules, 2)
//}

// TODO: make it work with the new arch
//func TestDBStorageDeleteDisabledRule(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mustWriteReport3Rules(t, mockStorage)
//
//	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
//		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, storage.RuleToggleDisable,
//	))
//
//	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
//		testdata.ClusterName, testdata.Rule2ID, testdata.UserID, storage.RuleToggleDisable,
//	))
//
//	toggledRules, err := mockStorage.ListDisabledRulesForCluster(testdata.ClusterName, testdata.UserID)
//	helpers.FailOnError(t, err)
//
//	assert.Len(t, toggledRules, 2)
//
//	helpers.FailOnError(t, mockStorage.DeleteFromRuleClusterToggle(
//		testdata.ClusterName, testdata.Rule2ID, testdata.UserID,
//	))
//
//	toggledRules, err = mockStorage.ListDisabledRulesForCluster(testdata.ClusterName, testdata.UserID)
//	helpers.FailOnError(t, err)
//
//	assert.Len(t, toggledRules, 1)
//}

func TestDBStorageVoteOnRule(t *testing.T) {
	for _, vote := range []types.UserVote{
		types.UserVoteDislike, types.UserVoteLike, types.UserVoteNone,
	} {
		func(vote types.UserVote) {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()

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
		}(vote)
	}
}

func TestDBStorageVoteOnRule_NoCluster(t *testing.T) {
	for _, vote := range []types.UserVote{
		types.UserVoteDislike, types.UserVoteLike, types.UserVoteNone,
	} {
		func(vote types.UserVote) {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()

			err := mockStorage.VoteOnRule(
				testdata.ClusterName, testdata.Rule1ID, testdata.UserID, vote,
			)
			assert.Error(t, err)
			assert.Regexp(t, "operation violates foreign key", err.Error())
		}(vote)
	}
}

// TODO: fix according to the new architecture
//func TestDBStorageVoteOnRule_NoRule(t *testing.T) {
//	for _, vote := range []types.UserVote{
//		types.UserVoteDislike, types.UserVoteLike, types.UserVoteNone,
//	} {
//		func(vote types.UserVote) {
//			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//			defer closer()
//
//			err := mockStorage.WriteReportForCluster(
//				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
//			)
//			helpers.FailOnError(t, err)
//
//			err = mockStorage.VoteOnRule(
//				testdata.ClusterName, testdata.Rule1ID, testdata.UserID, vote,
//			)
//			assert.Error(t, err)
//			assert.Regexp(t, "operation violates foreign key", err.Error())
//		}(vote)
//	}
//}

func TestDBStorageChangeVote(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteLike,
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteDislike,
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, "", feedback.Message)
	assert.Equal(t, types.UserVoteDislike, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageTextFeedback(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

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
	assert.Equal(t, types.UserVoteNone, feedback.UserVote)
}

func TestDBStorageFeedbackChangeMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

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
	assert.Equal(t, types.UserVoteNone, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageFeedbackErrorItemNotFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
	if _, ok := err.(*types.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

func TestDBStorageFeedbackErrorDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	_, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageVoteOnRuleDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	err := mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteNone)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageVoteOnRuleUnsupportedDriverError(t *testing.T) {
	connection, err := sql.Open("sqlite3", ":memory:")
	helpers.FailOnError(t, err)

	mockStorage := storage.NewFromConnection(connection, -1)
	defer ira_helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.MigrateToLatest())

	err = mockStorage.Init()
	helpers.FailOnError(t, err)

	err = mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteNone)
	assert.EqualError(t, err, "DB driver -1 is not supported")
}

func TestDBStorageVoteOnRuleDBExecError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()
	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

	query := `
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
	`

	if os.Getenv("INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB") == "postgres" {
		query = `
			CREATE TABLE cluster_rule_user_feedback (
				cluster_id INTEGER NOT NULL,
				rule_id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				message INTEGER NOT NULL,
				user_vote INTEGER NOT NULL,
				added_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,

				PRIMARY KEY(cluster_id, rule_id, user_id)
			)
		`
	}

	// create a table with a bad type
	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	err = mockStorage.VoteOnRule("non int", testdata.Rule1ID, testdata.UserID, types.UserVoteNone)
	assert.Error(t, err)
	const sqliteErrMessage = "CHECK constraint failed: cluster_rule_user_feedback"
	const postgresErrMessage = "pq: invalid input syntax for integer"
	if err.Error() != sqliteErrMessage && !strings.HasPrefix(err.Error(), postgresErrMessage) {
		t.Fatalf("expected on of: \n%v\n%v\ngot:\n%v", sqliteErrMessage, postgresErrMessage, err.Error())
	}
}

func TestDBStorageVoteOnRuleDBCloseError(t *testing.T) {
	// TODO: seems to be not coverable because of the bug in golang
	// related issues:
	// https://github.com/DATA-DOG/go-sqlmock/issues/185
	// https://github.com/golang/go/issues/37973

	const errStr = "close error"

	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpects(t)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectPrepare("INSERT").
		WillBeClosed().
		WillReturnCloseError(fmt.Errorf(errStr)).
		ExpectExec().
		WillReturnResult(driver.ResultNoRows)

	err := mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteNone)
	helpers.FailOnError(t, err)

	// TODO: uncomment when issues upthere resolved
	//assert.Contains(t, buf.String(), errStr)
}

func TestDBStorageGetVotesForNoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	feedbacks, err := mockStorage.GetUserFeedbackOnRules(
		testdata.ClusterName, testdata.RuleOnReportResponses, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Len(t, feedbacks, 0)
}

func TestDBStorageGetVotes(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.UserID, types.UserVoteLike,
	))
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule2ID, testdata.UserID, types.UserVoteDislike,
	))

	feedbacks, err := mockStorage.GetUserFeedbackOnRules(
		testdata.ClusterName, testdata.RuleOnReportResponses, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Len(t, feedbacks, 2)

	assert.Equal(t, types.UserVoteLike, feedbacks[testdata.Rule1ID])
	assert.Equal(t, types.UserVoteDislike, feedbacks[testdata.Rule2ID])
	assert.Equal(t, types.UserVoteNone, feedbacks[testdata.Rule3ID])
}
