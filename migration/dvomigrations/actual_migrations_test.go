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

package dvomigrations_test

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/migration/dvomigrations"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

type ruleHitsCount map[string]int

func (in ruleHitsCount) Value() (driver.Value, error) {
	return json.Marshal(in)
}

func (in *ruleHitsCount) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("not byte array")
	}

	return json.Unmarshal(b, &in)
}

func TestAllMigrations(t *testing.T) {
	db, closer := helpers.PrepareDBDVO(t)
	defer closer()

	dbConn := db.GetConnection()
	dbSchema := db.GetDBSchema()

	err := migration.InitInfoTable(dbConn, dbSchema)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(dbConn, db.GetDBDriverType(), dbSchema, db.GetMaxVersion(), dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)

	// downgrade back to 0
	err = migration.SetDBVersion(dbConn, db.GetDBDriverType(), dbSchema, migration.Version(0), dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)
}

func Test0004RuleHitsCount(t *testing.T) {
	db, closer := helpers.PrepareDBDVO(t)
	defer closer()

	dbConn := db.GetConnection()
	dbSchema := db.GetDBSchema()

	err := migration.SetDBVersion(dbConn, db.GetDBDriverType(), dbSchema, migration.Version(3), dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)

	// insert before mig 0004 to test that default values are parsable
	_, err = dbConn.Exec(`
	INSERT INTO dvo.dvo_report (org_id, cluster_id, namespace_id, recommendations, objects) VALUES ($1, $2, $3, $4, $5);
	`,
		testdata.OrgID,
		testdata.ClusterName,
		"namespace",
		1,
		2,
	)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(dbConn, db.GetDBDriverType(), dbSchema, migration.Version(4), dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)

	var ruleHits ruleHitsCount
	err = dbConn.QueryRow(`
		SELECT rule_hits_count FROM dvo.dvo_report WHERE cluster_id = $1;`,
		testdata.ClusterName,
	).Scan(
		&ruleHits,
	)

	helpers.FailOnError(t, err)
	// must be valid parsable json for existing rows
	assert.Equal(t, "{}", helpers.ToJSONString(ruleHits))

	cID := testdata.GetRandomClusterID()

	ruleHitsInput := ruleHitsCount{
		string(testdata.Rule1CompositeID): 1,
	}
	// insert a struct directly, implemented Value() method will take care of marshalling, validation, ..
	_, err = dbConn.Exec(`
	INSERT INTO dvo.dvo_report (org_id, cluster_id, namespace_id, recommendations, objects, rule_hits_count) VALUES ($1, $2, $3, $4, $5, $6);
	`,
		testdata.OrgID,
		cID,
		"namespace",
		1,
		2,
		ruleHitsInput,
	)
	helpers.FailOnError(t, err)

	err = dbConn.QueryRow(`
	SELECT rule_hits_count FROM dvo.dvo_report WHERE cluster_id = $1;`,
		cID,
	).Scan(
		&ruleHits,
	)

	helpers.FailOnError(t, err)
	assert.Equal(t, ruleHitsInput, ruleHits)
}

func TestMigration5_TableRuntimesHeartbeatsAlreadyExists(t *testing.T) {
	db, closer := helpers.PrepareDBDVO(t)
	defer closer()

	dbConn := db.GetConnection()

	err := migration.SetDBVersion(dbConn, db.GetDBDriverType(), db.GetDBSchema(), 4, dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)

	_, err = dbConn.Exec(`CREATE TABLE dvo.runtimes_heartbeats(c INTEGER);`)
	helpers.FailOnError(t, err)

	err = migration.SetDBVersion(dbConn, db.GetDBDriverType(), db.GetDBSchema(), db.GetMaxVersion(), dvomigrations.UsableDVOMigrations)
	assert.EqualError(t, err, "table runtimes_heartbeats already exists")
}

func TestMigration5_TableRuntimesHeartbeatsDoesNotExist(t *testing.T) {
	db, closer := helpers.PrepareDBDVO(t)
	defer closer()

	dbConn := db.GetConnection()

	err := migration.SetDBVersion(dbConn, db.GetDBDriverType(), db.GetDBSchema(), 5, dvomigrations.UsableDVOMigrations)
	helpers.FailOnError(t, err)

	_, err = dbConn.Exec(`DROP TABLE dvo.runtimes_heartbeats;`)
	helpers.FailOnError(t, err)

	// try to set to the first version
	err = migration.SetDBVersion(dbConn, db.GetDBDriverType(), db.GetDBSchema(), 4, dvomigrations.UsableDVOMigrations)
	assert.EqualError(t, err, "no such table: runtimes_heartbeats")
}
