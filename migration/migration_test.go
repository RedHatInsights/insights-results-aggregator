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

package migration_test

import (
	"database/sql"
	sql_driver "database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	dbClosedErrorMsg    = "sql: database is closed"
	noSuchTableErrorMsg = "no such table: public.migration_info"
	stepErrorMsg        = "migration Step Error"
)

var (
	stepNoopFn = func(tx *sql.Tx, _ types.DBDriver) error {
		return nil
	}
	stepErrorFn = func(tx *sql.Tx, _ types.DBDriver) error {
		return fmt.Errorf(stepErrorMsg)
	}
	stepRollbackFn = func(tx *sql.Tx, _ types.DBDriver) error {
		return tx.Rollback()
	}
	testMigration = migration.Migration{
		StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
			_, err := tx.Exec("CREATE TABLE migration_test_table (col INTEGER);")
			return err
		},
		StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
			_, err := tx.Exec("DROP TABLE migration_test_table")
			return err
		},
	}
	testMigrations = []migration.Migration{testMigration}
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// TestMigrationInit checks that database migration table initialization succeeds.
func TestMigrationInit(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()
	dbSchema := db.GetDBSchema()
	err := migration.InitInfoTable(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)
}

func TestMigrationInitDBSchema(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()
	dbSchema := db.GetDBSchema()
	err := migration.InitDBSchema(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	err = migration.InitInfoTable(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)
}

func TestMigrationInitDBSchemaMultipleTimes(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()
	dbSchema := db.GetDBSchema()
	err := migration.InitDBSchema(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	err = migration.InitInfoTable(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	// running again must be idempotent
	err = migration.InitDBSchema(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)
}

// TestMigrationInitDBSchemaEmptySchema must work with empty schema (uses default "public")
func TestMigrationInitDBSchemaEmptySchema(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()
	err := migration.InitDBSchema(dbConn, "")
	helpers.FailOnError(t, err)

	err = migration.InitInfoTable(dbConn, "")
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, "")
	helpers.FailOnError(t, err)
}

func TestMigrationInitDBSchemaWrongSchema(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()
	err := migration.InitDBSchema(dbConn, "-1")
	assert.Error(t, err)
}

// TestMigrationReInit checks that an attempt to re-initialize an already initialized
// migration info table will simply result in a no-op without any error.
func TestMigrationReInit(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	err := migration.InitInfoTable(dbConn, dbSchema)
	helpers.FailOnError(t, err)
}

func TestMigrationInitNotOneRow(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	_, err := dbConn.Exec("INSERT INTO public.migration_info(version) VALUES(10);")
	helpers.FailOnError(t, err)

	const expectedErrStr = "unexpected number of rows in migration info table (expected: 1, reality: 2)"
	err = migration.InitInfoTable(dbConn, dbSchema)
	assert.EqualError(t, err, expectedErrStr)
}

// TestMigrationGetVersion checks that the initial database migration version is 0.
func TestMigrationGetVersion(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	version, err := migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	assert.Equal(t, migration.Version(0), version, "unexpected database version")
}

func TestMigrationGetVersionMissingInfoTable(t *testing.T) {
	// Prepare DB without preparing the migration info table.
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	_, err := migration.GetDBVersion(db.GetConnection(), db.GetDBSchema())
	assert.EqualError(t, err, noSuchTableErrorMsg)
}

func TestMigrationGetVersionMultipleRows(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	_, err := dbConn.Exec("INSERT INTO public.migration_info(version) VALUES(10);")
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	assert.EqualError(t, err, "migration info table contain 2 rows")
}

func TestMigrationGetVersionEmptyTable(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	_, err := dbConn.Exec("DELETE FROM migration_info;")
	helpers.FailOnError(t, err)

	_, err = migration.GetDBVersion(dbConn, dbSchema)
	assert.EqualError(t, err, "migration info table contain 0 rows")
}

func TestMigrationGetVersionInvalidType(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	dbConn := db.GetConnection()

	_, err := dbConn.Exec("CREATE TABLE public.migration_info ( version TEXT );")
	helpers.FailOnError(t, err)

	_, err = dbConn.Exec("INSERT INTO public.migration_info(version) VALUES('hello world');")
	helpers.FailOnError(t, err)

	const expectedErrStr = `sql: Scan error on column index 0, name "version": ` +
		`converting driver.Value type string ("hello world") to a uint: invalid syntax`
	_, err = migration.GetDBVersion(dbConn, db.GetDBSchema())
	assert.EqualError(t, err, expectedErrStr)
}

// TestMigrationSetVersion checks that it is possible to change
// the database version in both direction (upgrade and downgrade).
func TestMigrationSetVersion(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	// Step-up from 0 to 1.
	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, testMigrations)
	helpers.FailOnError(t, err)

	version, err := migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	assert.Equal(t, migration.Version(1), version, "unexpected database version")

	// Step-down from 1 to 0.
	err = migration.SetDBVersion(dbConn, dbDriver, dbSchema, 0, testMigrations)
	helpers.FailOnError(t, err)

	version, err = migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	assert.Equal(t, migration.Version(0), version, "unexpected database version")
}

func TestMigrationNoInfoTable(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	defer closer()

	// Intentionally missing info table initialization here.

	_, err := migration.GetDBVersion(db.GetConnection(), db.GetDBSchema())
	assert.EqualError(
		t, err, noSuchTableErrorMsg, "migration info table should be missing when not initialized",
	)
}

func TestMigrationSetVersionSame(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	// Step-up from 0 to 1.
	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, testMigrations)
	helpers.FailOnError(t, err)

	// Set version to.
	err = migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, testMigrations)
	helpers.FailOnError(t, err)

	version, err := migration.GetDBVersion(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	assert.Equal(t, migration.Version(1), version, "unexpected database version")
}

func TestMigrationSetVersionTargetTooHigh(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	// Step-up from 0 to 2 (impossible -- only 1 migration is available).
	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 2, testMigrations)
	assert.EqualError(t, err, "invalid target version (available version range is 0-1)")
}

// TestMigrationSetVersionUpError checks that an error during a step-up is correctly handled.
func TestMigrationSetVersionUpError(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	tMigrations := []migration.Migration{
		{
			StepUp:   stepErrorFn,
			StepDown: stepNoopFn,
		},
	}

	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, tMigrations)
	assert.EqualError(t, err, stepErrorMsg)
}

// TestMigrationSetVersionDownError checks that an error during a step-down is correctly handled.
func TestMigrationSetVersionDownError(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	tMigrations := []migration.Migration{
		{
			StepUp:   stepNoopFn,
			StepDown: stepErrorFn,
		},
	}

	// First we need to step-up before we can step-down.
	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, tMigrations)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(dbConn, dbDriver, dbSchema, 0, tMigrations)
	assert.EqualError(t, err, stepErrorMsg)
}

// TestMigrationSetVersionCurrentTooHighError makes sure that if the current DB version
// is outside of the available migration range, it is reported as an error.
func TestMigrationSetVersionCurrentTooHighError(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	_, err := dbConn.Exec("UPDATE public.migration_info SET version=10;")
	helpers.FailOnError(t, err)

	const expectedErrStr = "current version (10) is outside of available migration boundaries"
	err = migration.SetDBVersion(dbConn, dbDriver, dbSchema, 0, testMigrations)
	assert.EqualError(t, err, expectedErrStr)
}

func TestMigrationInitClosedDB(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	// Intentionally no `defer` here.
	closer()

	err := migration.InitInfoTable(db.GetConnection(), db.GetDBSchema())
	assert.EqualError(t, err, dbClosedErrorMsg)
}

func TestMigrationGetVersionClosedDB(t *testing.T) {
	dbConn, _, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	// Intentionally no `defer` here.
	closer()

	_, err := migration.GetDBVersion(dbConn, dbSchema)
	assert.EqualError(t, err, dbClosedErrorMsg)
}

func TestMigrationSetVersionClosedDB(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	// Intentionally no `defer` here.
	closer()

	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 0, testMigrations)
	assert.EqualError(t, err, dbClosedErrorMsg)
}

func TestMigrationInitRollbackStep(t *testing.T) {
	dbConn, dbDriver, dbSchema, closer := ira_helpers.PrepareDBAndInfo(t)
	defer closer()

	tMigrations := []migration.Migration{
		{
			StepUp:   stepRollbackFn,
			StepDown: stepNoopFn,
		},
	}

	const expectedErrStr = "sql: transaction has already been committed or rolled back"
	err := migration.SetDBVersion(dbConn, dbDriver, dbSchema, 1, tMigrations)
	assert.EqualError(t, err, expectedErrStr)
}

func TestInitInfoTable_BeginTransactionDBError(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
	closer()
	err := migration.InitInfoTable(db.GetConnection(), db.GetDBSchema())
	assert.EqualError(t, err, "sql: database is closed")
}

func TestInitInfoTable_InitTableDBError(t *testing.T) {
	const errStr = "create table error"

	db, expects := ira_helpers.MustGetMockDBWithExpects(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectBegin()
	expects.ExpectExec("CREATE TABLE IF NOT EXISTS public.migration_info").WillReturnError(fmt.Errorf(errStr))
	expects.ExpectRollback()

	err := migration.InitInfoTable(db, "")
	assert.EqualError(t, err, errStr)
}

func TestInitInfoTable_InitVersionDBError(t *testing.T) {
	const errStr = "insert error"

	db, expects := ira_helpers.MustGetMockDBWithExpects(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectBegin()
	expects.ExpectExec("CREATE TABLE IF NOT EXISTS public.migration_info").WillReturnResult(sql_driver.ResultNoRows)
	expects.ExpectExec("INSERT INTO public.migration_info").WillReturnError(fmt.Errorf(errStr))
	expects.ExpectRollback()

	err := migration.InitInfoTable(db, "")
	assert.EqualError(t, err, errStr)
}

func TestInitInfoTable_CountDBError(t *testing.T) {
	const errStr = "count error"

	db, expects := ira_helpers.MustGetMockDBWithExpects(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectBegin()
	expects.ExpectExec("CREATE TABLE IF NOT EXISTS public.migration_info").WillReturnResult(sql_driver.ResultNoRows)
	expects.ExpectExec("INSERT INTO public.migration_info").WillReturnResult(sql_driver.ResultNoRows)
	expects.ExpectQuery("SELECT COUNT.+FROM public.migration_info").WillReturnError(fmt.Errorf(errStr))
	expects.ExpectRollback()

	err := migration.InitInfoTable(db, "")
	assert.EqualError(t, err, errStr)
}

func updateVersionInDBCommon(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, expects := ira_helpers.MustGetMockDBWithExpects(t)

	expects.ExpectBegin()
	expects.ExpectExec("CREATE TABLE IF NOT EXISTS public.migration_info").WillReturnResult(sql_driver.ResultNoRows)
	expects.ExpectExec("INSERT INTO public.migration_info").WillReturnResult(sql_driver.ResultNoRows)
	expects.ExpectQuery("SELECT COUNT.+FROM public.migration_info").WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow(1),
	)
	expects.ExpectCommit()

	err := migration.InitInfoTable(db, "")
	helpers.FailOnError(t, err)

	expects.ExpectQuery("SELECT COUNT.+FROM public.migration_info").WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow(1),
	)
	expects.ExpectQuery("SELECT version FROM public.migration_info").WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow(0),
	)
	expects.ExpectBegin()
	expects.ExpectExec("CREATE TABLE migration_test_table").WillReturnResult(sql_driver.ResultNoRows)

	return db, expects
}

func TestUpdateVersionInDB_RowsAffectedError(t *testing.T) {
	const errStr = "rows affected error"

	db, expects := updateVersionInDBCommon(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectExec("UPDATE public.migration_info SET version").
		WithArgs(1).
		WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf(errStr)))

	err := migration.SetDBVersion(db, types.DBDriverGeneral, "", 1, testMigrations)
	assert.EqualError(t, err, errStr)
}

func TestUpdateVersionInDB_MoreThan1RowAffected(t *testing.T) {
	db, expects := updateVersionInDBCommon(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectExec("UPDATE public.migration_info SET version").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(1, 2))

	err := migration.SetDBVersion(db, types.DBDriverGeneral, "", 1, testMigrations)
	assert.EqualError(
		t, err, "unexpected number of affected rows in migration info table (expected: 1, reality: 2)",
	)
}

func TestWithTransaction_Panic(t *testing.T) {
	const errStr = "panic"

	db, expects := ira_helpers.MustGetMockDBWithExpects(t)
	defer ira_helpers.MustCloseMockDBWithExpects(t, db, expects)

	expects.ExpectBegin()
	expects.ExpectRollback()

	defer func() {
		p := recover()
		assert.Equal(t, p, errStr, "panic is expected")
	}()

	_ = migration.WithTransaction(db, func(tx *sql.Tx) error {
		panic(errStr)
	})
	t.Fatal("not expected to go here")
}
