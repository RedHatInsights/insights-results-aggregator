package migration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestAllMigrations(t *testing.T) {
	db := prepareDB(t)
	defer closeDB(t, db)

	err := migration.InitInfoTable(db)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, migration.GetMaxVersion())
	helpers.FailOnError(t, err)
}

func TestMigrationsOneByOne(t *testing.T) {
	db := prepareDB(t)
	defer closeDB(t, db)

	allMigrations := make([]migration.Migration, len(*migration.Migrations))
	copy(allMigrations, *migration.Migrations)
	*migration.Migrations = []migration.Migration{}

	for i := 0; i < len(allMigrations); i++ {
		// add one migration to the list
		*migration.Migrations = append(*migration.Migrations, allMigrations[i])

		err := migration.InitInfoTable(db)
		helpers.FailOnError(t, err)

		err = migration.SetDBVersion(db, migration.GetMaxVersion())
		helpers.FailOnError(t, err)
	}
}

func TestAllMigrations_Migration1TableReportAlreadyExists(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	_, err := db.Exec(`CREATE TABLE report(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, migration.GetMaxVersion())
	assert.EqualError(t, err, "table report already exists")
}

func TestAllMigrations_Migration1TableReportDoesNotExist(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	// set to the latest version
	err := migration.SetDBVersion(db, migration.GetMaxVersion())
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE report;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, 0)
	assert.EqualError(t, err, "no such table: report")
}

func TestAllMigrations_Migration2TableRuleAlreadyExists(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	_, err := db.Exec(`CREATE TABLE rule(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, migration.GetMaxVersion())
	assert.EqualError(t, err, "table rule already exists")
}

func TestAllMigrations_Migration2TableRuleDoesNotExist(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	// set to the latest version
	err := migration.SetDBVersion(db, migration.GetMaxVersion())
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE rule;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, 0)
	assert.EqualError(t, err, "no such table: rule")
}

func TestAllMigrations_Migration2TableRuleErrorKeyAlreadyExists(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	_, err := db.Exec(`CREATE TABLE rule_error_key(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, migration.GetMaxVersion())
	assert.EqualError(t, err, "table rule_error_key already exists")
}

func TestAllMigrations_Migration2TableRuleErrorKeyDoesNotExist(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	// set to the latest version
	err := migration.SetDBVersion(db, migration.GetMaxVersion())
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE rule_error_key;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, 0)
	assert.EqualError(t, err, "no such table: rule_error_key")
}

func TestAllMigrations_Migration3TableClusterRuleUserFeedbackAlreadyExists(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	_, err := db.Exec(`CREATE TABLE cluster_rule_user_feedback(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(db, migration.GetMaxVersion())
	assert.EqualError(t, err, "table cluster_rule_user_feedback already exists")
}

func TestAllMigrations_Migration3TableClusterRuleUserFeedbackDoesNotExist(t *testing.T) {
	db := prepareDBAndInfo(t)
	defer closeDB(t, db)

	// set to the latest version
	err := migration.SetDBVersion(db, migration.GetMaxVersion())
	helpers.FailOnError(t, err)

	_, err = db.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(db, 0)
	assert.EqualError(t, err, "no such table: cluster_rule_user_feedback")
}
