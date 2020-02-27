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
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/stretchr/testify/assert"
)

func TestInitAndGetSQLDriverWithLogsOK(t *testing.T) {
	logger := log.New(os.Stdout, "[sql]", log.LstdFlags)
	driverName, err := storage.InitAndGetSQLDriverWithLogs(storage.DBDriverSQLite3, logger)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "sqlite3WithHooks", driverName)

	driverName, err = storage.InitAndGetSQLDriverWithLogs(storage.DBDriverPostgres, logger)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "postgresWithHooks", driverName)
}

func TestInitAndGetSQLDriverWithLogsDriverNotFound(t *testing.T) {
	logger := log.New(os.Stdout, "[sql]", log.LstdFlags)
	const nonExistingDriver = -1
	_, err := storage.InitAndGetSQLDriverWithLogs(nonExistingDriver, logger)
	if err == nil || err.Error() != fmt.Sprintf("driver %v is not supported", nonExistingDriver) {
		t.Fatal(fmt.Errorf("expected driver not supported error, got %+v", err))
	}
}

func TestSQLHooksLoggingArgsJSON(t *testing.T) {
	const query = "SELECT 1"
	var params = []interface{}{}

	buf := new(bytes.Buffer)
	logger := log.New(buf, "[sql]", log.LstdFlags)
	hooks := storage.SQLHooks{logger}

	_, err := hooks.Before(context.Background(), query, params...)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params)+" took",
	)
}

func TestSQLHooksLoggingArgsNotJSON(t *testing.T) {
	// just for better test coverage :)
	const query = "SELECT 1"
	var params = []interface{}{math.Inf(1)}

	buf := new(bytes.Buffer)
	logger := log.New(buf, "[sql]", log.LstdFlags)
	hooks := storage.SQLHooks{logger}

	_, err := hooks.Before(context.Background(), query, params...)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(
		t,
		buf.String(),
		fmt.Sprintf(storage.LogFormatterString, query, params)+" took",
	)
}
