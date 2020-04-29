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
	rowCount    = 10000
	insertQuery = "INSERT INTO benchmark_tab (name, value) VALUES ($1, $2);"
)

func mustPrepareBenchmark(b *testing.B) (storage.Storage, *sql.DB) {
	mockStorage := helpers.MustGetMockStorage(b, false)
	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	if _, err := conn.Exec("CREATE TABLE benchmark_tab (id SERIAL, name VARCHAR(256), value VARCHAR(4096));"); err != nil {
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
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

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
