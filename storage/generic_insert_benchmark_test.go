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

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/rs/zerolog"
)

const (
	rowCount    = 1000
	insertQuery = "INSERT INTO benchmark_tab (name, value) VALUES ($1, $2);"
	upsertQuery = "INSERT INTO benchmark_tab (id, name, value) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name=$2, value=$3;"
	// SQLite alternative to the upsert query above:
	// upsertQuery = "REPLACE INTO benchmark_tab (id, name, value) VALUES ($1, $2, $3);"
)

func mustPrepareBenchmark(b *testing.B) (storage.Storage, *sql.DB) {
	// Postgres queries are very verbose at DEBUG log level, so it's better
	// to silence them this way to make benchmark results easier to find.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	mockStorage, _ := helpers.MustGetPostgresStorage(b)
	// Alternative using the file-based SQLite DB storage:
	// mockStorage, _ := helpers.MustGetSQLiteFileStorage(b)
	// Old version using the in-memory SQLite DB storage:
	// mockStorage := helpers.MustGetMockStorage(b, false)

	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	if _, err := conn.Exec("DROP TABLE IF EXISTS benchmark_tab;"); err != nil {
		b.Fatal(err)
	}

	if _, err := conn.Exec("CREATE TABLE benchmark_tab (id SERIAL PRIMARY KEY, name VARCHAR(256), value VARCHAR(4096));"); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.StartTimer()

	return mockStorage, conn
}

func mustCleanupAfterBenchmark(b *testing.B, stor storage.Storage, conn *sql.DB) {
	b.StopTimer()

	if _, err := conn.Exec("DROP TABLE benchmark_tab;"); err != nil {
		b.Fatal(err)
	}

	if err := stor.Close(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkStorageGenericInsertExecDirectlySingle executes a single INSERT statement directly.
func BenchmarkStorageGenericInsertExecDirectlySingle(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		if _, err := conn.Exec(insertQuery, "John Doe", "Hello World!"); err != nil {
			b.Fatal(err)
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStorageGenericInsertPrepareExecSingle prepares an INSERT statement and then executes it once.
func BenchmarkStorageGenericInsertPrepareExecSingle(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		if err != nil {
			b.Fatal(err)
		}

		if _, err := stmt.Exec("John Doe", "Hello World!"); err != nil {
			b.Fatal(err)
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStorageGenericInsertExecDirectlyMany executes the INSERT query row by row,
// each in a separate sql.DB.Exec() call.
func BenchmarkStorageGenericInsertExecDirectlyMany(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			if _, err := conn.Exec(insertQuery, "John Doe", "Hello World!"); err != nil {
				b.Fatal(err)
			}
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStorageGenericInsertPrepareExecMany executes the same exact INSERT statements,
// but it prepares them beforehand and only supplies the parameters with each call.
func BenchmarkStorageGenericInsertPrepareExecMany(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		if err != nil {
			b.Fatal(err)
		}

		for rowId := 0; rowId < rowCount; rowId++ {
			if _, err := stmt.Exec("John Doe", "Hello World!"); err != nil {
				b.Fatal(err)
			}
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStorageUpsertWithoutConflict inserts many non-conflicting
// rows into the benchmark table using the upsert query.
func BenchmarkStorageUpsertWithoutConflict(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			if _, err := conn.Exec(upsertQuery, rowId+1, "John Doe", "Hello World!"); err != nil {
				b.Fatal(err)
			}
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStorageUpsertConflict insert many mutually conflicting
// rows into the benchmark table using the uspert query.
func BenchmarkStorageUpsertConflict(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			if _, err := conn.Exec(upsertQuery, 1, "John Doe", "Hello World!"); err != nil {
				b.Fatal(err)
			}
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}
