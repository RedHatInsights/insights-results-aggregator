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

func mustPrepareBenchmark(b *testing.B) (storage.Storage, *sql.DB, func()) {
	mockStorage, closer := helpers.MustGetMockStorage(b, false)

	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	_, err := conn.Exec("CREATE TABLE benchmark_tab (id SERIAL, name VARCHAR(256), value VARCHAR(4096));")
	helpers.FailOnError(b, err)

	b.ResetTimer()
	b.StartTimer()

	return mockStorage, conn, closer
}

func mustCleanupAfterBenchmark(b *testing.B, stor storage.Storage, conn *sql.DB, closer func()) {
	b.StopTimer()

	_, err := conn.Exec("DROP TABLE benchmark_tab;")
	helpers.FailOnError(b, err)

	err = stor.Close()
	helpers.FailOnError(b, err)

	closer()
}

// BenchmarkStorageExecDirectlySingle executes a single INSERT statement directly.
func BenchmarkStorageExecDirectlySingle(b *testing.B) {
	stor, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		_, err := conn.Exec(insertQuery, "John Doe", "Hello World!")
		helpers.FailOnError(b, err)
	}

	mustCleanupAfterBenchmark(b, stor, conn, closer)
}

// BenchmarkStoragePrepareExecSingle prepares an INSERT statement and then executes it once.
func BenchmarkStoragePrepareExecSingle(b *testing.B) {
	stor, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		helpers.FailOnError(b, err)

		_, err = stmt.Exec("John Doe", "Hello World!")
		helpers.FailOnError(b, err)
	}

	mustCleanupAfterBenchmark(b, stor, conn, closer)
}

// BenchmarkStorageExecDirectlyMany executes the INSERT query row by row,
// each in a separate sql.DB.Exec() call.
func BenchmarkStorageExecDirectlyMany(b *testing.B) {
	stor, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := conn.Exec(insertQuery, "John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn, closer)
}

// BenchmarkStoragePrepareExecMany executes the same exact INSERT statements,
// but it prepares them beforehand and only supplies the parameters with each call.
func BenchmarkStoragePrepareExecMany(b *testing.B) {
	stor, conn, closer := mustPrepareBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		stmt, err := conn.Prepare(insertQuery)
		helpers.FailOnError(b, err)

		for rowId := 0; rowId < rowCount; rowId++ {
			_, err := stmt.Exec("John Doe", "Hello World!")
			helpers.FailOnError(b, err)
		}
	}

	mustCleanupAfterBenchmark(b, stor, conn, closer)
}
