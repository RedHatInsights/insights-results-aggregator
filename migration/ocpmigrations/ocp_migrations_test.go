package ocpmigrations_test

import (
	"database/sql"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func prepareDB(t *testing.T) (*sql.DB, types.DBDriver, func()) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)

	return dbStorage.GetConnection(), dbStorage.GetDBDriverType(), closer
}

func prepareDBAndInfo(t *testing.T) (*sql.DB, types.DBDriver, func()) {
	db, dbDriver, closer := prepareDB(t)

	if err := migration.InitInfoTable(db); err != nil {
		closer()
		t.Fatal(err)
	}

	return db, dbDriver, closer
}
