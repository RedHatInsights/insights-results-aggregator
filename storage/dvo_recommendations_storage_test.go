/*
Copyright © 2023 Red Hat, Inc.

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
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

// TestNewDVOStorageError checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageError(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "sql",
	})
	assert.EqualError(t, err, "driver non existing driver is not supported")
}

// TestNewDVOStorageNoType checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageNoType(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
	})
	assert.EqualError(t, err, "Unknown storage type ''")
}

// TestNewDVOStorageWrongType checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageWrongType(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "foobar",
	})
	assert.EqualError(t, err, "Unknown storage type 'foobar'")
}

// TestNewDVOStorageReturnedImplementation check what implementation of storage is returnd
func TestNewDVOStorageReturnedImplementation(t *testing.T) {
	s, _ := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "sql",
	})
	assert.IsType(t, &storage.DVORecommendationsDBStorage{}, s)

	s, _ = storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "noop",
	})
	assert.IsType(t, &storage.NoopDVOStorage{}, s)

	s, _ = storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "redis",
	})
	assert.Nil(t, s, "redis type is not supported for DVO storage")
}

func TestDBStorage_getWorkloadsInsertStatement(t *testing.T) {
	fakeStorage := storage.NewDVORecommendationsFromConnection(nil, -1)
	r := fakeStorage.GetWorkloadsInsertStatement(3)

	// 5*3 placeholders expected
	const expected = "INSERT INTO dvo.dvo_report(org_id, cluster_id, namespace_id, namespace_name, report, recommendations, objects, reported_at, last_checked_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9),($10,$11,$12,$13,$14,$15,$16,$17,$18),($19,$20,$21,$22,$23,$24,$25,$26,$27)"
	assert.Equal(t, expected, r)
}
