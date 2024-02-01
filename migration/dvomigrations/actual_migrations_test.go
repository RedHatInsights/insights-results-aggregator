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

/*
func TestAllMigrations(t *testing.T) {
	db, closer := ira_helpers.PrepareDB(t)
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
*/
