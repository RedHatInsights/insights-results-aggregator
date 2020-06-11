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

package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestInitSQLDriverWithLogs(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	driverName := storage.InitSQLDriverWithLogs(
		&sqlite3.SQLiteDriver{},
		"sqlite3",
	)
	assert.Equal(t, "sqlite3WithHooks", driverName)

	driverName = storage.InitSQLDriverWithLogs(
		&pq.Driver{},
		"postgres",
	)
	assert.Equal(t, "postgresWithHooks", driverName)
}

// TestInitSQLDriverWithLogsMultipleCalls tests if InitSQLDriverWithLogs
// does not panic on multiple calls and is idempotent
func TestInitSQLDriverWithLogsMultipleCalls(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	for i := 0; i < 10; i++ {
		driverName := storage.InitSQLDriverWithLogs(
			&sqlite3.SQLiteDriver{},
			"sqlite3",
		)
		assert.Equal(t, "sqlite3WithHooks", driverName)
	}
}

func TestSQLHooksLoggingArgsJSON(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	const query = "SELECT 1"
	params := make([]interface{}, 0)

	buf := new(bytes.Buffer)
	logger := zerolog.New(buf).With().Str("type", "SQL").Logger()
	log.Logger = logger

	hooks := storage.SQLHooks{}

	_, err := hooks.Before(context.Background(), query, params...)
	helpers.FailOnError(t, err)

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params),
	)

	_, err = hooks.After(
		context.WithValue(context.Background(), storage.SQLHooksKeyQueryBeginTime, time.Now()),
		query,
		params...,
	)
	helpers.FailOnError(t, err)

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params)+" took",
	)
}

func TestSQLHooksLoggingArgsNotJSON(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// just for better test coverage :)
	const query = "SELECT 1"
	var params = []interface{}{math.Inf(1)}

	buf := new(bytes.Buffer)
	logger := zerolog.New(buf).With().Str("type", "SQL").Logger()
	log.Logger = logger
	hooks := storage.SQLHooks{}

	_, err := hooks.Before(context.Background(), query, params...)
	helpers.FailOnError(t, err)

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params),
	)

	_, err = hooks.After(
		context.WithValue(context.Background(), storage.SQLHooksKeyQueryBeginTime, time.Now()),
		query,
		params...,
	)
	helpers.FailOnError(t, err)

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params)+" took",
	)
}
