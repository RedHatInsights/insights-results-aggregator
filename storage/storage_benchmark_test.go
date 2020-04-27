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

// To run the benchmarks defined in this file, you can use the following command:
// go test -benchmem -run=^$ github.com/RedHatInsights/insights-results-aggregator/storage -bench '^BenchmarkStorage' -benchtime=5s

package storage_test

import (
	"database/sql"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	rowCount    = 10000
	insertQuery = "INSERT INTO benchmark_tab (name, value) VALUES ($1, $2);"
)

func mustPrepareBenchmark(b *testing.B) (storage.Storage, *sql.DB) {
	mockStorage, err := helpers.GetMockStorage(false)
	if err != nil {
		b.Fatal(err)
	}

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

// BenchmarkStorageExecDirectlySingle executes a single INSERT statement directly.
func BenchmarkStorageExecDirectlySingle(b *testing.B) {
	stor, conn := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		if _, err := conn.Exec(insertQuery, "John Doe", "Hello World!"); err != nil {
			b.Fatal(err)
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn)
}

// BenchmarkStoragePrepareExecSingle prepares an INSERT statement and then executes it once.
func BenchmarkStoragePrepareExecSingle(b *testing.B) {
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

// BenchmarkStorageExecDirectlyMany executes the INSERT query row by row,
// each in a separate sql.DB.Exec() call.
func BenchmarkStorageExecDirectlyMany(b *testing.B) {
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

// BenchmarkStoragePrepareExecMany executes the same exact INSERT statements,
// but it prepares them beforehand and only supplies the parameters with each call.
func BenchmarkStoragePrepareExecMany(b *testing.B) {
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
