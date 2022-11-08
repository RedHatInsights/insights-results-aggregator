// Copyright 2020, 2021, 2022 Red Hat, Inc
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

package migration_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func GetTxForMigration(t *testing.T) (*sql.Tx, *sql.DB, sqlmock.Sqlmock) {
	db, expects := ira_helpers.MustGetMockDBWithExpects(t)

	expects.ExpectBegin()

	tx, err := db.Begin()
	helpers.FailOnError(t, err)

	return tx, db, expects
}

func TestAllMigrations(t *testing.T) {
	db, dbDriver, closer := prepareDB(t)
	defer closer()

	err := migration.InitInfoTable(db)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	helpers.FailOnError(t, err)
}

func TestMigrationsOneByOne(t *testing.T) {
	db, dbDriver, closer := prepareDB(t)
	defer closer()

	allMigrations := make([]migration.Migration, len(*migration.Migrations))
	copy(allMigrations, *migration.Migrations)
	*migration.Migrations = []migration.Migration{}

	for i := 0; i < len(allMigrations); i++ {
		// add one migration to the list
		*migration.Migrations = append(*migration.Migrations, allMigrations[i])

		err := migration.InitInfoTable(db)
		helpers.FailOnError(t, err)

		err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
		helpers.FailOnError(t, err)
	}
}

func TestMigration1_TableReportAlreadyExists(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	_, err := db.Exec(`CREATE TABLE report(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "table report already exists")
}

func TestMigration1_TableReportDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	// set to the version with the report table
	err := migration.SetDBVersion(db, dbDriver, 1)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE report;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: report")
}

func TestMigration2_TableRuleAlreadyExists(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	_, err := db.Exec(`CREATE TABLE rule(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "table rule already exists")
}

func TestMigration2_TableRuleDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	// set to the version where table rule exists
	err := migration.SetDBVersion(db, dbDriver, 2)
	helpers.FailOnError(t, err)

	if dbDriver == types.DBDriverSQLite3 {
		_, err = db.Exec(`DROP TABLE rule;`)
		helpers.FailOnError(t, err)
	} else {
		_, err = db.Exec(`DROP TABLE rule CASCADE;`)
		helpers.FailOnError(t, err)
	}

	// try to set to the first version
	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: rule")
}

func TestMigration2_TableRuleErrorKeyAlreadyExists(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	_, err := db.Exec(`CREATE TABLE rule_error_key(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "table rule_error_key already exists")
}

func TestMigration2_TableRuleErrorKeyDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	// set to the latest version
	err := migration.SetDBVersion(db, dbDriver, 2)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE rule_error_key;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: rule_error_key")
}

func TestMigration3_TableClusterRuleUserFeedbackAlreadyExists(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	_, err := db.Exec(`CREATE TABLE cluster_rule_user_feedback(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "table cluster_rule_user_feedback already exists")
}

func TestMigration3_TableClusterRuleUserFeedbackDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	// set to the latest version
	err := migration.SetDBVersion(db, dbDriver, 3)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: cluster_rule_user_feedback")
}

func TestMigration4_StepUp_TableClusterRuleUserFeedbackDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	err := migration.SetDBVersion(db, dbDriver, 3)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "no such table: cluster_rule_user_feedback")
}

func TestMigration4_StepDown_TableClusterRuleUserFeedbackDoesNotExist(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	err := migration.SetDBVersion(db, dbDriver, 4)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: cluster_rule_user_feedback")
}

func TestMigration4_CreateTableError(t *testing.T) {
	expectedErr := fmt.Errorf("create table error")
	mig4 := migration.Mig0004ModifyClusterRuleUserFeedback

	for _, method := range []func(*sql.Tx, types.DBDriver) error{mig4.StepUp, mig4.StepDown} {
		func(method func(*sql.Tx, types.DBDriver) error) {
			tx, db, expects := GetTxForMigration(t)
			defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

			expects.ExpectExec("ALTER TABLE").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("CREATE TABLE").
				WillReturnError(expectedErr)

			err := method(tx, types.DBDriverGeneral)
			assert.EqualError(t, err, expectedErr.Error())
		}(method)
	}
}

func TestMigration4_InsertError(t *testing.T) {
	expectedErr := fmt.Errorf("insert error")
	mig4 := migration.Mig0004ModifyClusterRuleUserFeedback

	for _, method := range []func(*sql.Tx, types.DBDriver) error{mig4.StepUp, mig4.StepDown} {
		func(method func(*sql.Tx, types.DBDriver) error) {
			tx, db, expects := GetTxForMigration(t)
			defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

			expects.ExpectExec("ALTER TABLE").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("CREATE TABLE").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("INSERT INTO").
				WillReturnError(expectedErr)

			err := method(tx, types.DBDriverGeneral)
			assert.EqualError(t, err, expectedErr.Error())
		}(method)
	}
}

func TestMigration4_DropTableError(t *testing.T) {
	expectedErr := fmt.Errorf("drop table error")
	mig4 := migration.Mig0004ModifyClusterRuleUserFeedback

	for _, method := range []func(*sql.Tx, types.DBDriver) error{mig4.StepUp, mig4.StepDown} {
		func(method func(*sql.Tx, types.DBDriver) error) {
			tx, db, expects := GetTxForMigration(t)
			defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

			expects.ExpectExec("ALTER TABLE").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("CREATE TABLE").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("INSERT INTO").
				WillReturnResult(driver.ResultNoRows)
			expects.ExpectExec("DROP TABLE").
				WillReturnError(expectedErr)

			err := method(tx, types.DBDriverGeneral)
			assert.EqualError(t, err, expectedErr.Error())
		}(method)
	}
}

func TestMigration5_TableAlreadyExists(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	_, err := db.Exec("CREATE TABLE consumer_error(c INTEGER);")
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, migration.GetMaxVersion())
	assert.EqualError(t, err, "table consumer_error already exists")
}

func TestMigration5_NoSuchTable(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	err := migration.SetDBVersion(db, dbDriver, 5)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE consumer_error`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 0)
	assert.EqualError(t, err, "no such table: consumer_error")
}

func TestMigration13(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// migration is not implemented for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 12)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5)
	`,
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReport3Rules,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 13)
	helpers.FailOnError(t, err)

	assertRule := func(ruleFQDN types.RuleID, errorKey types.ErrorKey, expectedTemplateData string) {
		var (
			templateData string
		)

		err := db.QueryRow(`
			SELECT
				template_data
			FROM
				rule_hit
			WHERE
				org_id = $1 AND cluster_id = $2 AND
				rule_fqdn = $3 AND error_key = $4
		`,
			testdata.OrgID,
			testdata.ClusterName,
			ruleFQDN,
			errorKey,
		).Scan(
			&templateData,
		)
		helpers.FailOnError(t, err)

		if helpers.IsStringJSON(expectedTemplateData) {
			assert.JSONEq(t, expectedTemplateData, templateData)
		} else {
			assert.Equal(t, expectedTemplateData, templateData)
		}
	}

	assertRule(testdata.Rule1ID, testdata.ErrorKey1, helpers.ToJSONString(testdata.Rule1ExtraData))
	assertRule(testdata.Rule2ID, testdata.ErrorKey2, helpers.ToJSONString(testdata.Rule2ExtraData))
	assertRule(testdata.Rule3ID, testdata.ErrorKey3, helpers.ToJSONString(testdata.Rule3ExtraData))
}

func TestMigration16(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	err := migration.SetDBVersion(db, dbDriver, 15)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key)
		VALUES ($1, $2, $3, $4)
	`,
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Rule1Name,
		testdata.ErrorKey1,
	)
	assert.Error(t, err, `Expected error since recommendation table does not exist yet`)

	if dbDriver == types.DBDriverSQLite3 {
		assert.Contains(t, err.Error(), "no such table: recommendation")
	} else if dbDriver == types.DBDriverPostgres {
		assert.Contains(t, err.Error(), `relation "recommendation" does not exist`)
	}

	err = migration.SetDBVersion(db, dbDriver, 16)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key)
		VALUES ($1, $2, $3, $4)
	`,
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Rule1Name,
		testdata.ErrorKey1,
	)
	helpers.FailOnError(t, err)
}

func TestMigration19(t *testing.T) {

	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// nothing worth testing for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 18)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT created_at FROM recommendation`).Err()
	assert.Error(t, err, "created_at column should not exist")
	err = db.QueryRow(`SELECT rule_id FROM recommendation`).Err()
	assert.Error(t, err, "rule_id column should not exist")

	correctRuleID := testdata.Rule1ID + "|" + testdata.ErrorKey1
	incorrectRuleFQDN := testdata.Rule1ID + "." + testdata.ErrorKey1

	expectedRuleAfterMigration := string(testdata.Rule1ID)

	_, err = db.Exec(`
		INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key)
		VALUES ($1, $2, $3, $4)
		`,
		testdata.OrgID,
		testdata.ClusterName,
		incorrectRuleFQDN,
		testdata.ErrorKey1,
	)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key)
		VALUES ($1, $2, $3, $4)
		`,
		testdata.Org2ID,
		testdata.ClusterName,
		correctRuleID,
		testdata.ErrorKey1,
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 19)
	helpers.FailOnError(t, err)

	var (
		ruleFQDN string
		ruleID   string
	)

	err = db.QueryRow(`
			SELECT
				rule_fqdn, rule_id
			FROM
				recommendation
			WHERE
				org_id = $1`,
		testdata.OrgID,
	).Scan(
		&ruleFQDN, &ruleID,
	)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedRuleAfterMigration, ruleFQDN)
	assert.Equal(t, string(correctRuleID), ruleID)

	err = db.QueryRow(`
			SELECT
				rule_fqdn, rule_id
			FROM
				recommendation
			WHERE
				org_id = $1`,
		testdata.Org2ID,
	).Scan(
		&ruleFQDN, &ruleID,
	)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedRuleAfterMigration, ruleFQDN)
	assert.Equal(t, string(correctRuleID), ruleID)
	var timestamp time.Time

	err = db.QueryRow(`
			SELECT
				created_at
			FROM
				recommendation
			WHERE
				org_id = $1`,
		testdata.OrgID,
	).Scan(
		&timestamp,
	)
	helpers.FailOnError(t, err)
	assert.False(t, timestamp.IsZero(), "The timestamp column was not created with a default value")
	assert.True(t, timestamp.UTC().Equal(timestamp), "The stored timestamp is not in UTC format")

	// Step down should remove created_at and rule_id columns
	err = migration.SetDBVersion(db, dbDriver, 18)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT created_at FROM recommendation`).Err()
	assert.Error(t, err, "created_at column should not exist")
	err = db.QueryRow(`SELECT rule_id FROM recommendation`).Err()
	assert.Error(t, err, "rule_id column should not exist")
}

func TestMigration18(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	err := migration.SetDBVersion(db, dbDriver, 17)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO advisor_ratings
		(user_id, org_id, rule_id, error_key, rated_at, last_updated_at, rating)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
	`,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1Name,
		testdata.ErrorKey1,
		time.Now(),
		time.Now(),
		types.UserVoteLike,
	)
	assert.Error(t, err, `Expected error since advisor_ratings table does not exist yet`)

	if dbDriver == types.DBDriverSQLite3 {
		assert.Contains(t, err.Error(), "no such table: advisor_ratings")
	} else if dbDriver == types.DBDriverPostgres {
		assert.Contains(t, err.Error(), `relation "advisor_ratings" does not exist`)
	}

	err = migration.SetDBVersion(db, dbDriver, 18)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO advisor_ratings
		(user_id, org_id, rule_id, error_key, rated_at, last_updated_at, rating)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
	`,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1Name,
		testdata.ErrorKey1,
		time.Now(),
		time.Now(),
		types.UserVoteLike,
	)
	helpers.FailOnError(t, err)
}

func TestMigration20(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// nothing worth testing for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 19)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT rule_fqdn FROM advisor_ratings`).Err()
	assert.Error(t, err, "rule_fqdn column should not exist")

	_, err = db.Exec(`
		INSERT INTO advisor_ratings
		(user_id, org_id, rule_id, error_key, rated_at, last_updated_at, rating)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
	`,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		time.Now(),
		time.Now(),
		types.UserVoteLike,
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 20)
	helpers.FailOnError(t, err)

	var (
		ruleFQDN string
		ruleID   string
	)

	err = db.QueryRow(`
			SELECT
				rule_fqdn, rule_id
			FROM
				advisor_ratings
			WHERE
				user_id = $1 AND org_id = $2`,
		testdata.UserID, testdata.OrgID,
	).Scan(
		&ruleFQDN, &ruleID,
	)
	helpers.FailOnError(t, err)
	assert.Equal(t, testdata.Rule1ID, types.RuleID(ruleFQDN))
	assert.Equal(t, testdata.Rule1CompositeID, types.RuleID(ruleID))

	// Step down should rename rule_fqdn column to rule_id and it contains only plugin name
	err = migration.SetDBVersion(db, dbDriver, 19)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT rule_fqdn FROM advisor_ratings`).Err()
	assert.Error(t, err, "rule_fqdn column should not exist")
	err = db.QueryRow(`
			SELECT
				rule_id
			FROM
				advisor_ratings
			WHERE
				user_id = $1 AND org_id = $2`,
		testdata.UserID, testdata.OrgID,
	).Scan(
		&ruleID,
	)
	helpers.FailOnError(t, err)
	assert.Equal(t, testdata.Rule1ID, types.RuleID(ruleID))
}

func TestMigration22(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 21)
	helpers.FailOnError(t, err)

	var expectedCorrectCount int

	// insert toggles and feedbacks
	for i := 0; i < 10; i++ {
		ruleID := testdata.Rule1ID
		clusterID := testdata.GetRandomClusterID()

		// we expect 4 with the suffix (correct) and 6 without the suffix (to be deleted)
		if i%2 == 0 {
			ruleID += ".report"
			expectedCorrectCount++
		}

		// insert toggles
		_, err = db.Exec(`
				INSERT INTO cluster_rule_toggle
				(cluster_id, rule_id, error_key, user_id, disabled, updated_at)
				VALUES
				($1, $2, $3, $4, $5, $6)
			`,
			clusterID,
			ruleID,
			testdata.ErrorKey1, // we don't care about error key or any other column
			testdata.UserID,
			storage.RuleToggleDisable,
			time.Now(),
		)
		helpers.FailOnError(t, err)

		// insert disable feedbacks
		_, err = db.Exec(`
				INSERT INTO cluster_user_rule_disable_feedback
				(cluster_id, rule_id, error_key, user_id, message, added_at, updated_at)
				VALUES
				($1, $2, $3, $4, $5, $6, $6)
			`,
			clusterID,
			ruleID,
			testdata.ErrorKey1, // we don't care about error key or any other column
			testdata.UserID,
			"",
			time.Now(),
		)
		helpers.FailOnError(t, err)
	}

	// migrate to 22
	err = migration.SetDBVersion(db, dbDriver, 22)
	helpers.FailOnError(t, err)

	// retrieve numbers of rows
	var togglesAfterMigrationCount, feedbacksAfterMigrationCount int

	err = db.QueryRow(`
	SELECT
		count(*)
	FROM
		cluster_rule_toggle
	`).Scan(
		&togglesAfterMigrationCount,
	)
	helpers.FailOnError(t, err)
	// must match with expected count
	assert.Equal(t, expectedCorrectCount, togglesAfterMigrationCount)

	err = db.QueryRow(`
	SELECT
		count(*)
	FROM
		cluster_user_rule_disable_feedback
	`).Scan(
		&feedbacksAfterMigrationCount,
	)
	helpers.FailOnError(t, err)
	// must match with expected count
	assert.Equal(t, expectedCorrectCount, feedbacksAfterMigrationCount)

	// there is no need to test StepDown because it's a NOOP
}

func TestMigration24(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 23)
	helpers.FailOnError(t, err)

	_, err = db.Exec(
		`SELECT created_at FROM rule_hit`,
	)
	assert.NotNil(t, err)

	// migrate to 24
	err = migration.SetDBVersion(db, dbDriver, 24)
	helpers.FailOnError(t, err)

	_, err = db.Exec(
		`SELECT created_at FROM rule_hit`,
	)
	helpers.FailOnError(t, err)
}

func TestMigration25(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 24)
	helpers.FailOnError(t, err)

	_, err = db.Exec(
		`SELECT impacted_since FROM recommendation`,
	)
	assert.Error(t, err)

	// migrate to 25
	err = migration.SetDBVersion(db, dbDriver, 25)
	helpers.FailOnError(t, err)

	_, err = db.Exec(
		`SELECT impacted_since FROM recommendation`,
	)
	helpers.FailOnError(t, err)
}

func TestMigration26(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 25)
	helpers.FailOnError(t, err)

	_, err = db.Exec(
		`SELECT org_id FROM cluster_rule_toggle`,
	)
	assert.Error(t, err)

	_, err = db.Exec(
		`SELECT org_id FROM cluster_rule_user_feedback`,
	)
	assert.Error(t, err)

	_, err = db.Exec(
		`SELECT org_id FROM cluster_user_rule_disable_feedback`,
	)
	assert.Error(t, err)

	// insert into report table
	_, err = db.Exec(`
		INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5)
		`,
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReport3Rules,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// insert into cluster_rule_toggle
	_, err = db.Exec(
		`INSERT INTO cluster_rule_toggle (cluster_id, rule_id, error_key, user_id, disabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		testdata.ClusterName,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.UserID,
		1,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	unknownClusterID := testdata.GetRandomClusterID()
	// insert into cluster_rule_toggle
	_, err = db.Exec(
		`INSERT INTO cluster_rule_toggle (cluster_id, rule_id, error_key, user_id, disabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		unknownClusterID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.UserID,
		1,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// migrate to 26, popoulating org_id column based on report table
	err = migration.SetDBVersion(db, dbDriver, 26)
	helpers.FailOnError(t, err)

	var orgID types.OrgID
	err = db.QueryRow(`
	SELECT
		org_id
	FROM
		cluster_rule_toggle
	WHERE
		cluster_id = $1
	`, testdata.ClusterName,
	).Scan(
		&orgID,
	)
	helpers.FailOnError(t, err)

	// org_id must match that in report table
	assert.Equal(t, orgID, testdata.OrgID)

	err = db.QueryRow(`
	SELECT
		org_id
	FROM
		cluster_rule_toggle
	WHERE
		cluster_id = $1
	`, unknownClusterID,
	).Scan(
		&orgID,
	)
	helpers.FailOnError(t, err)

	// cluster wasn't found in report table, org_id will be default value 0
	assert.Equal(t, orgID, types.OrgID(0))
}

func TestMigration27(t *testing.T) {
	const countQuery = "SELECT count(*) FROM %v"
	// contains 2 valid and 1 invalid orgID
	var orgIDList = []string{"0", "1", "2"}
	var tableList = []string{
		"cluster_rule_toggle",
		"cluster_rule_user_feedback",
		"cluster_user_rule_disable_feedback",
	}

	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 26)
	helpers.FailOnError(t, err)

	// insert into report table because of DB constraints
	_, err = db.Exec(`
		INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5)
		`,
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReport3Rules,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// insert 3 rows into table, one of them invalid
	for i, orgID := range orgIDList {
		// insert into cluster_rule_toggle
		userID := fmt.Sprintf("%v", i)
		_, err = db.Exec(
			`INSERT INTO cluster_rule_toggle (cluster_id, rule_id, error_key, user_id, disabled, updated_at, org_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			testdata.ClusterName,
			testdata.Rule1ID,
			userID, // random stuff because of DB constraints
			testdata.UserID,
			1,
			testdata.LastCheckedAt,
			orgID,
		)
		helpers.FailOnError(t, err)

		// insert into cluster_user_rule_disable_feedback
		_, err = db.Exec(
			`INSERT INTO cluster_user_rule_disable_feedback
				(cluster_id, rule_id, error_key, user_id, message, added_at, updated_at, org_id)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8)`,
			testdata.ClusterName,
			testdata.Rule1ID,
			testdata.ErrorKey1,
			userID,
			"reason",
			testdata.LastCheckedAt,
			testdata.LastCheckedAt,
			orgID,
		)
		helpers.FailOnError(t, err)

		// insert into cluster_rule_user_feedback
		_, err = db.Exec(
			`INSERT INTO cluster_rule_user_feedback
				(cluster_id, rule_id, error_key, user_id, message, added_at, updated_at, user_vote, org_id)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			testdata.ClusterName,
			testdata.Rule1ID,
			testdata.ErrorKey1,
			userID,
			"reason",
			testdata.LastCheckedAt,
			testdata.LastCheckedAt,
			1,
			orgID,
		)
		helpers.FailOnError(t, err)
	}

	// check correct number of rows inserted (3)
	for _, table := range tableList {
		var cnt int
		query := fmt.Sprintf(countQuery, table)
		err = db.QueryRow(query).Scan(&cnt)
		helpers.FailOnError(t, err)
		// expect 3 rows
		assert.Equal(t, cnt, 3)
	}

	// migrate to 27, deleting rows where org_id = 0
	err = migration.SetDBVersion(db, dbDriver, 27)
	helpers.FailOnError(t, err)

	// check correct number of rows remaining (2)
	for _, table := range tableList {
		var cnt int
		query := fmt.Sprintf(countQuery, table)
		err = db.QueryRow(query).Scan(&cnt)
		helpers.FailOnError(t, err)
		// expect 2 rows, one was supposed to be deleted in all tables
		assert.Equal(t, cnt, 2)
	}
}

func TestMigration28(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// sqlite is no longer supported
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 27)
	helpers.FailOnError(t, err)

	// insert into rule_disable
	_, err = db.Exec(
		`INSERT INTO rule_disable
				(org_id, user_id, rule_id, error_key, created_at)
			VALUES
				($1, $2, $3, $4, $5)`,
		testdata.OrgID,
		testdata.UserID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// insert into rule_disable different user_id, same org_id
	_, err = db.Exec(
		`INSERT INTO rule_disable
				(org_id, user_id, rule_id, error_key, created_at)
			VALUES
				($1, $2, $3, $4, $5)`,
		testdata.OrgID,
		testdata.User2ID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.LastCheckedAt,
	)
	// possible to insert more than 1 row per org
	helpers.FailOnError(t, err)

	// delete from table
	_, err = db.Exec(
		`DELETE FROM rule_disable`,
	)
	helpers.FailOnError(t, err)

	// migrate to different constraint
	err = migration.SetDBVersion(db, dbDriver, 28)
	helpers.FailOnError(t, err)

	// insert into rule_disable
	_, err = db.Exec(
		`INSERT INTO rule_disable
				(org_id, user_id, rule_id, error_key, created_at)
			VALUES
				($1, $2, $3, $4, $5)`,
		testdata.OrgID,
		testdata.UserID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// insert into rule_disable different user_id, same org_id
	_, err = db.Exec(
		`INSERT INTO rule_disable
				(org_id, user_id, rule_id, error_key, created_at)
			VALUES
				($1, $2, $3, $4, $5)`,
		testdata.OrgID,
		testdata.User2ID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		testdata.LastCheckedAt,
	)
	// not possible to insert more than 1 row per org
	assert.Error(t, err)
}

func TestMigration29(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// nothing worth testing for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 28)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO cluster_rule_toggle
		(cluster_id, user_id, org_id, rule_id, error_key, disabled, disabled_at, updated_at)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		testdata.ClusterName,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		1,
		time.Now(),
		time.Now(),
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 29)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT user_id FROM cluster_rule_toggle`).Err()
	assert.Error(t, err, "user_id column should not exist")

	err = migration.SetDBVersion(db, dbDriver, 28)
	helpers.FailOnError(t, err)

	var userID types.UserID
	err = db.QueryRow(`
	SELECT
		user_id
	FROM
		cluster_rule_toggle
	`,
	).Scan(
		&userID,
	)
	helpers.FailOnError(t, err)

	// default value on stepdown
	assert.Equal(t, userID, types.UserID("-1"))
}

func TestMigration30(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// nothing worth testing for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 29)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO rule_disable
		(user_id, org_id, rule_id, error_key, justification, created_at, updated_at)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
	`,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		"",
		time.Now(),
		time.Now(),
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 30)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT user_id FROM rule_disable`).Err()
	assert.Error(t, err, "user_id column should not exist")

	err = migration.SetDBVersion(db, dbDriver, 29)
	helpers.FailOnError(t, err)

	var userID types.UserID
	err = db.QueryRow(`
	SELECT
		user_id
	FROM
		rule_disable
	`,
	).Scan(
		&userID,
	)
	helpers.FailOnError(t, err)

	// default value on stepdown
	assert.Equal(t, userID, types.UserID("-1"))
}

func TestMigration31(t *testing.T) {
	db, dbDriver, closer := prepareDBAndInfo(t)
	defer closer()

	if dbDriver == types.DBDriverSQLite3 {
		// nothing worth testing for sqlite
		return
	}

	err := migration.SetDBVersion(db, dbDriver, 30)
	helpers.FailOnError(t, err)

	_, err = db.Exec(`
		INSERT INTO advisor_ratings
		(user_id, org_id, rule_fqdn, error_key, rated_at, last_updated_at, rating, rule_id)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		testdata.UserID,
		testdata.OrgID,
		testdata.Rule1ID,
		testdata.ErrorKey1,
		time.Now(),
		time.Now(),
		1,
		testdata.Rule1CompositeID,
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, dbDriver, 31)
	helpers.FailOnError(t, err)

	err = db.QueryRow(`SELECT user_id FROM advisor_ratings`).Err()
	assert.Error(t, err, "user_id column should not exist")

	err = migration.SetDBVersion(db, dbDriver, 29)
	helpers.FailOnError(t, err)

	var userID types.UserID
	err = db.QueryRow(`
	SELECT
		user_id
	FROM
		advisor_ratings
	`,
	).Scan(
		&userID,
	)
	helpers.FailOnError(t, err)

	// default value on stepdown
	assert.Equal(t, userID, types.UserID("-1"))
}
