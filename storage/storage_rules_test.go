// Copyright 2020, 2021 Red Hat, Inc
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

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func mustWriteReport3Rules(t *testing.T, mockStorage storage.Storage) {
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.Report3RulesParsed, testdata.LastCheckedAt, time.Now(), testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorage_ToggleRuleForCluster(t *testing.T) {
	for _, state := range []storage.RuleToggle{
		storage.RuleToggleDisable, storage.RuleToggleEnable,
	} {
		func(state storage.RuleToggle) {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()

			mustWriteReport3Rules(t, mockStorage)

			helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
				testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, state,
			))

			_, err := mockStorage.GetFromClusterRuleToggle(testdata.ClusterName, testdata.Rule1ID)
			helpers.FailOnError(t, err)
		}(state)
	}
}

func TestDBStorage_ToggleRuleForCluster_UnexpectedRuleToggleValue(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, -999,
	)
	assert.EqualError(t, err, "Unexpected rule toggle value")
}

func TestDBStorage_ToggleRuleForCluster_DBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	err := mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, storage.RuleToggleDisable,
	)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageGetTogglesForRules_NoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, nil,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorageGetTogglesForRules_AllRulesEnabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, testdata.RuleOnReportResponses,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorageGetTogglesForRules_OneRuleDisabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, storage.RuleToggleDisable,
	))

	result, err := mockStorage.GetTogglesForRules(
		testdata.ClusterName, testdata.RuleOnReportResponses,
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
				testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, state,
			))

			toggledRule, err := mockStorage.GetFromClusterRuleToggle(testdata.ClusterName, testdata.Rule1ID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testdata.ClusterName, toggledRule.ClusterID)
			assert.Equal(t, testdata.Rule1ID, toggledRule.RuleID)
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

// TestDBStorageListRulesReasonsOnDBError checks that no rules reasons are
// returned for DB error.
func TestDBStorageListRulesReasonsOnDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediatelly
	closer()

	// try to read list of reasons
	_, err := mockStorage.ListOfReasons(testdata.UserID)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageListRulesReasonsEmptyDB checks that no rules reasons are
// returned for empty DB.
func TestDBStorageListRulesReasonsEmptyDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// try to read list of reasons
	reasons, err := mockStorage.ListOfReasons(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect no rules reasons to be returned
	assert.Len(t, reasons, 0)
}

// TestDBStorageListOfRulesReasonsOneRule checks that one rule is returned
// for non empty DB.
// TODO: enable when user_id is properly handled (stored) into database!
func TestDBStorageListOfRulesReasonsOneRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// write some rules into database
	mustWriteReport3Rules(t, mockStorage)

	const feedbackMessage = "feedback message"

	// store one reason
	helpers.FailOnError(t, mockStorage.AddFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1,
		testdata.UserID, feedbackMessage,
	))

	// try to read list of reasons
	reasons, err := mockStorage.ListOfReasons(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect 1 rule reason to be returned
	assert.Len(t, reasons, 1)

	// check the content of returned data
	reason := reasons[0]
	assert.Equal(t, testdata.ClusterName, reason.ClusterID)
	assert.Equal(t, testdata.Rule1ID, reason.RuleID)
	assert.Equal(t, testdata.ErrorKey1, string(reason.ErrorKey))
	assert.Equal(t, feedbackMessage, reason.Message)
}

// TestDBStorageListOfDisabledRulesDBError checks that no rules are returned
// for DB error.
func TestDBStorageListOfDisabledRulesDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediatelly
	closer()

	// try to read list of disabled rules
	_, err := mockStorage.ListOfDisabledRules(testdata.UserID)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageListOfDisabledRulesEmptyDB checks that no rules are returned
// for empty DB.
func TestDBStorageListOfDisabledRulesEmptyDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// try to read list of disabled rules
	disabledRules, err := mockStorage.ListOfDisabledRules(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect no rules to be returned
	assert.Len(t, disabledRules, 0)
}

// TestDBStorageListOfDisabledRulesOneRule checks that one rule is returned
// for non empty DB.
func TestDBStorageListOfDisabledRulesOneRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// write some rules into database
	mustWriteReport3Rules(t, mockStorage)

	// disable one rule
	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1,
		testdata.UserID, storage.RuleToggleDisable,
	))

	// try to read list of disabled rules
	disabledRules, err := mockStorage.ListOfDisabledRules(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect 1 rule to be returned
	assert.Len(t, disabledRules, 1)

	// check the content of returned data
	disabledRule := disabledRules[0]
	assert.Equal(t, testdata.ClusterName, disabledRule.ClusterID)
	assert.Equal(t, testdata.Rule1ID, disabledRule.RuleID)
	assert.Equal(t, testdata.ErrorKey1, string(disabledRule.ErrorKey))
	assert.Equal(t, int(storage.RuleToggleDisable), int(disabledRule.Disabled))
}

// TestDBStorageListOfDisabledRulesTwoRules checks that two rules are returned
// for non empty DB.
func TestDBStorageListOfDisabledRulesTwoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// write some rules into database
	mustWriteReport3Rules(t, mockStorage)

	// disable two rules
	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1,
		testdata.UserID, storage.RuleToggleDisable,
	))
	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule2ID, testdata.ErrorKey1,
		testdata.UserID, storage.RuleToggleDisable,
	))

	// try to read list of disabled rules
	disabledRules, err := mockStorage.ListOfDisabledRules(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect 2 rules to be returned
	assert.Len(t, disabledRules, 2)

	// check the content of returned data
	disabledRule := disabledRules[0]
	assert.Equal(t, testdata.ClusterName, disabledRule.ClusterID)
	assert.Equal(t, int(storage.RuleToggleDisable), int(disabledRule.Disabled))

	disabledRule = disabledRules[1]
	assert.Equal(t, testdata.ClusterName, disabledRule.ClusterID)
	assert.Equal(t, int(storage.RuleToggleDisable), int(disabledRule.Disabled))
}

// TestDBStorageListOfDisabledRulesNoRule checks that no rule is returned
// for non empty DB when all rules are enabled.
func TestDBStorageListOfDisabledRulesNoRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// write some rules into database
	mustWriteReport3Rules(t, mockStorage)

	// enable one rule
	helpers.FailOnError(t, mockStorage.ToggleRuleForCluster(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1,
		testdata.UserID, storage.RuleToggleEnable,
	))

	// try to read list of disabled rules
	disabledRules, err := mockStorage.ListOfDisabledRules(testdata.UserID)
	helpers.FailOnError(t, err)

	// we expect no rules to be returned
	assert.Len(t, disabledRules, 0)
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
				testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, vote, "",
			))

			feedback, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID)
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
				testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, vote, "",
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
//				testdata.OrgID, testdata.ClusterName, report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
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
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteLike, "",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteDislike, "",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, types.ErrorKey(testdata.ErrorKey1), feedback.ErrorKey)
	assert.Equal(t, "", feedback.Message)
	assert.Equal(t, types.UserVoteDislike, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageTextFeedback(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "test feedback",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID,
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
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "message1",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "message2",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID,
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

	_, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID)
	if _, ok := err.(*types.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

func TestDBStorageFeedbackErrorDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	_, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageVoteOnRuleDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	err := mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteNone, "")
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

	err = mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteNone, "")
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
			error_key VARCHAR NOT NULL,

			PRIMARY KEY(cluster_id, rule_id, user_id, error_key)
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
				error_key VARCHAR NOT NULL,

				PRIMARY KEY(cluster_id, rule_id, user_id, error_key)
			)
		`
	}

	// create a table with a bad type
	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	err = mockStorage.VoteOnRule("non int", testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteNone, "")
	assert.Error(t, err)
	const sqliteErrMessage = "CHECK constraint failed: cluster_rule_user_feedback"
	const postgresErrMessage = "pq: invalid input syntax for integer: \"non int\""
	if err.Error() != sqliteErrMessage && !strings.HasPrefix(err.Error(), postgresErrMessage) {
		t.Fatalf("expected one of: \n%v\n%v\ngot:\n%v", sqliteErrMessage, postgresErrMessage, err.Error())
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

	err := mockStorage.VoteOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteNone, "")
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

func TestDBStorageGetDisableFeedback(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	feedbacks, err := mockStorage.GetUserDisableFeedbackOnRules(
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
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteLike, "",
	))
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testdata.ClusterName, testdata.Rule2ID, testdata.ErrorKey1, testdata.UserID, types.UserVoteDislike, "",
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

func TestDBStorageTextDisableFeedback(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "test feedback",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, "test feedback", feedback.Message)
	assert.Equal(t, types.UserVoteNone, feedback.UserVote)
}

func TestDBStorageDisableFeedbackChangeMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "message1",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.AddFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID, "message2",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRuleDisable(
		testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
	assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
	assert.Equal(t, testdata.UserID, feedback.UserID)
	assert.Equal(t, "message2", feedback.Message)
	assert.Equal(t, types.UserVoteNone, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageDisableFeedbackErrorItemNotFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, err := mockStorage.GetUserFeedbackOnRuleDisable(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID)
	if _, ok := err.(*types.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

func TestDBStorageDisableFeedbackErrorDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	_, err := mockStorage.GetUserFeedbackOnRuleDisable(testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageListClustersForHittingRules checks the list of HittingClustersData
// objects retrieved when ListOfClustersForOrgSpecificRule is called
func TestDBStorageListClustersForHittingRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	clusterIds := []ctypes.ClusterName{
		testdata.GetRandomClusterID(),
		testdata.GetRandomClusterID(),
		testdata.GetRandomClusterID(),
	}
	helpers.FailOnError(t, mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, clusterIds[0], testdata.Report3Rules,
	))
	//ClusterIds[1] is not associated to any rule hit and is not expected in any response
	helpers.FailOnError(t, mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, clusterIds[1], testdata.Report0Rules,
	))
	helpers.FailOnError(t, mockStorage.WriteRecommendationsForCluster(
		testdata.Org2ID, clusterIds[2], testdata.Report2Rules,
	))

	//TODO: Add these to test data to ensure consistency

	//Rule1|ERR_KEY1 is present in testdata.Report3Rules and testdata.Report2Rules,
	//but only clusters for testdata.OrgID are returned
	expectedClustersOrg1Rule1Err1 := []ctypes.HittingClustersData{
		{Cluster: clusterIds[0]},
	}
	//Rule2|ERR_KEY2 is present in testdata.Report3Rules and testdata.Report2Rules,
	//but only clusters for testdata.OrgID are returned
	expectedClustersOrg1Rule2Err2 := []ctypes.HittingClustersData{
		{Cluster: clusterIds[0]},
	}
	//Rule3|ERR_KEY3 is present in testdata.Report3Rules
	expectedClustersOrg1Rule3Err3 := []ctypes.HittingClustersData{
		{Cluster: clusterIds[0]},
	}
	//Rule1|ERR_KEY1 is present in testdata.Report3Rules and testdata.Report2Rules,
	//but only clusters for testdata.Org2ID are returned
	expectedClustersOrg2Rule1Err1 := []ctypes.HittingClustersData{
		{Cluster: clusterIds[2]},
	}
	//Rule2|ERR_KEY2 is present in testdata.Report3Rules and testdata.Report2Rules,
	//but only clusters for testdata.Org2ID are returned
	expectedClustersOrg2Rule2Err2 := []ctypes.HittingClustersData{
		{Cluster: clusterIds[2]},
	}

	list, err := mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule1CompositeID), nil)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedClustersOrg1Rule1Err1, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule2CompositeID), nil)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedClustersOrg1Rule2Err2, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule3CompositeID), nil)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedClustersOrg1Rule3Err3, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.Org2ID, types.RuleSelector(testdata.Rule1CompositeID), nil)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedClustersOrg2Rule1Err1, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.Org2ID, types.RuleSelector(testdata.Rule1CompositeID), nil)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedClustersOrg2Rule2Err2, list)

	//Now let's add some active clusters filtering
	//Rule1|ERR_KEY1 is present in testdata.Report3Rules and testdata.Report2Rules,
	//but only clusters for testdata.OrgID are returned, and since clusterIds[0] is
	//not active, an empty list of hitting clusters should be returned as well as an
	//ItemNotFoundError
	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule1CompositeID), []string{string(clusterIds[1]), string(clusterIds[2])})
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.Org2ID, types.RuleSelector(testdata.Rule1CompositeID), []string{string(clusterIds[0]), string(clusterIds[1])})
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)
}

// TestDBStorageListClustersForHittingRulesOrgNotFound checks that an empty
// list of HittingClustersData objects is returned when the given org ID
// has no associated entries in the recommendation table
func TestDBStorageListClustersForHittingRulesOrgNotFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	helpers.FailOnError(t, mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.GetRandomClusterID(), testdata.Report3Rules,
	))

	list, err := mockStorage.ListOfClustersForOrgSpecificRule(testdata.Org2ID, types.RuleSelector(testdata.Rule1CompositeID), nil)
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)
}

// TestDBStorageListClustersForHittingRulesOrgNotFound checks that an empty
// list of HittingClustersData objects is returned when the given rule selector
// has no associated entries in the recommendation table, as well as an
// ItemNotFoundError, independently of if a list of active clusters is passed
// as the SQL query filter.
func TestDBStorageListClustersForHittingRulesRuleNotFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	helpers.FailOnError(t, mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.GetRandomClusterID(), testdata.Report2Rules,
	))

	list, err := mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule3CompositeID), nil)
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)

	list, err = mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule3CompositeID), []string{string(testdata.GetRandomClusterID())})
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)
}

// TestDBStorageListClustersForHittingRulesNoRowsFound checks that an empty
// list of HittingClustersData objects is returned, as well as an
// ItemNotFoundError (converted in a Error 404), when the SQL query for
// hitting recommendations returns no rows (Any other DB error will
// be indicated to client as a 503).
func TestDBStorageListClustersForHittingRulesNoRowsFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	list, err := mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule3CompositeID), nil)
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)
}

// TestDBStorageListClustersForHittingRulesNoRowsFound checks that an empty
// list of HittingClustersData objects is returned, as well as an
// ItemNotFoundError (converted in a Error 404), when the SQL query for
// hitting recommendations returns no rows (Any other DB error will
// be indicated to client as a 503). In this case, a list of active clusters
// is given, which changes the query made to the DB, but not the expected
// behavior.
func TestDBStorageListFilteredClustersForHittingRulesNoRowsFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	list, err := mockStorage.ListOfClustersForOrgSpecificRule(testdata.OrgID, types.RuleSelector(testdata.Rule3CompositeID), []string{string(testdata.ClusterName)})
	assert.Error(t, err)
	assert.IsType(t, &utypes.ItemNotFoundError{}, err)
	assert.Equal(t, []ctypes.HittingClustersData{}, list)
}
