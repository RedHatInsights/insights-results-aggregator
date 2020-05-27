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
	"database/sql"
	"testing"

	"github.com/rs/zerolog"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	rowCount    = 1000
	insertQuery = "INSERT INTO benchmark_tab (name, value) VALUES ($1, $2);"
	upsertQuery = "INSERT INTO benchmark_tab (id, name, value) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name=$2, value=$3;"
	// SQLite alternative to the upsert query above:
	// upsertQuery = "REPLACE INTO benchmark_tab (id, name, value) VALUES ($1, $2, $3);"
)

func mustPrepareBenchmark(b *testing.B) (storage.Storage, *sql.DB, func()) {
	// Postgres queries are very verbose at DEBUG log level, so it's better
	// to silence them this way to make benchmark results easier to find.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	mockStorage, closer := helpers.MustGetPostgresStorage(b, false)
	// Alternative using the file-based SQLite DB storage:
	// mockStorage, _ := helpers.MustGetSQLiteFileStorage(b)
	// Old version using the in-memory SQLite DB storage:
	// mockStorage := helpers.MustGetMockStorage(b, false)

	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	_, err := conn.Exec("DROP TABLE IF EXISTS benchmark_tab;")
	helpers.FailOnError(b, err)

	_, err = conn.Exec("CREATE TABLE benchmark_tab (id SERIAL PRIMARY KEY, name VARCHAR(256), value VARCHAR(4096));")
	helpers.FailOnError(b, err)

	b.ResetTimer()
	b.StartTimer()

	return mockStorage, conn, closer
}

func mustCleanupAfterBenchmark(b *testing.B, conn *sql.DB, closer func()) {
	b.StopTimer()

	_, err := conn.Exec("DROP TABLE benchmark_tab;")
	helpers.FailOnError(b, err)

	closer()
}

// BenchmarkStorageGenericInsertExecDirectlySingle executes a single INSERT statement directly.
func BenchmarkStorageGenericInsertExecDirectlySingle(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		_, err := conn.Exec(insertQuery, "John Doe", "Hello World!")
		helpers.FailOnError(b, err)
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}

// BenchmarkStorageGenericInsertPrepareExecSingle prepares an INSERT statement and then executes it once.
func BenchmarkStorageGenericInsertPrepareExecSingle(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		helpers.FailOnError(b, err)

		_, err = stmt.Exec("John Doe", "Hello World!")
		helpers.FailOnError(b, err)
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}

// BenchmarkStorageGenericInsertExecDirectlyMany executes the INSERT query row by row,
// each in a separate sql.DB.Exec() call.
func BenchmarkStorageGenericInsertExecDirectlyMany(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := conn.Exec(insertQuery, "John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}

// BenchmarkStorageGenericInsertPrepareExecMany executes the same exact INSERT statements,
// but it prepares them beforehand and only supplies the parameters with each call.
func BenchmarkStorageGenericInsertPrepareExecMany(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		helpers.FailOnError(b, err)

		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := stmt.Exec("John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}

// BenchmarkStorageUpsertWithoutConflict inserts many non-conflicting
// rows into the benchmark table using the upsert query.
func BenchmarkStorageUpsertWithoutConflict(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := conn.Exec(upsertQuery, (benchIter*rowCount)+rowId+1, "John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}

// BenchmarkStorageUpsertConflict insert many mutually conflicting
// rows into the benchmark table using the uspert query.
func BenchmarkStorageUpsertConflict(b *testing.B) {
	_, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := conn.Exec(upsertQuery, 1, "John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, conn, closer)
}
