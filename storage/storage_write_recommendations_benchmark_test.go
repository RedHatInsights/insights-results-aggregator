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
	"fmt"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

var Report20Rules = types.ClusterReport(`
{
	"system": {
		"metadata": {},
		"hostname": null
	},
	"reports": [
		{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "` + testdata.ErrorKey1 + `",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "` + testdata.ErrorKey2 + `",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek3",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek4",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek5",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek6",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek7",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek8",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek9",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek10",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek11",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek12",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek13",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek14",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek15",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek16",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek17",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek18",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek19",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek20",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek200",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek21",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek22",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek23",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek24",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek25",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek26",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek27",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek28",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		},
{
			"component": "` + string(testdata.Rule1ID) + `",
			"key": "ek29",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule1ExtraData) + `
		},
		{
			"component": "` + string(testdata.Rule2ID) + `",
			"key": "ek30",
			"user_vote": 0,
			"disabled": false,
			"details": ` + helpers.ToJSONString(testdata.Rule2ExtraData) + `
		}
	],
	"fingerprints": [],
	"skips": [],
	"info": []
}
`)

var exists = struct{}{}

type set struct {
	content map[string]struct{}
}

// newSet function allows us to make sure there is no duplicated cluster name
// in the entries we use for testing.
func newSet() *set {
	s := &set{}
	s.content = make(map[string]struct{})
	return s
}

func (s *set) add(value string) {
	s.content[value] = exists
}

func (s *set) remove(value string) {
	delete(s.content, value)
}

func (s *set) contains(value string) bool {
	_, c := s.content[value]
	return c
}

func resetTimerForBenchmark(b *testing.B) {
	b.ResetTimer()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	b.StartTimer()
}

func stopBenchmarkTimer(b *testing.B) {
	b.StopTimer()
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// Only create recommendation table in the test DB
func mustPrepareRecommendationsBenchmark(b *testing.B) (storage.Storage, *sql.DB, func()) {
	// Postgres queries are very verbose at DEBUG log level, so it's better
	// to silence them this way to make benchmark results easier to find.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(b, false)
	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	_, err := conn.Exec("DROP TABLE IF EXISTS recommendations;")
	helpers.FailOnError(b, err)

	_, err = conn.Exec("CREATE TABLE recommendations (cluster_id VARCHAR NOT NULL, rule_fqdn VARCHAR NOT NULL, error_key VARCHAR NOT NULL, PRIMARY KEY(cluster_id, rule_fqdn, error_key));")
	helpers.FailOnError(b, err)

	return mockStorage, conn, closer
}

// Only create recommendation table in the test DB, and insert numRows entries
// in the table before the benchmarking timers are reset
func mustPrepareRecommendationsBenchmarkWithEntries(b *testing.B, numRows int) (storage.Storage, *sql.DB, func()) {
	mockStorage, conn, closer := mustPrepareReportAndRecommendationsBenchmark(b)

	for i := 0; i < numRows; i++ {
		cluster := uuid.New().String()
		_, err := conn.Exec(`
			INSERT INTO recommendations (
				cluster_id, rule_fqdn, error_key
			) VALUES ($1, $2, $3)
		`, cluster, "a rule module", "an error key")
		helpers.FailOnError(b, err)
		storage.SetClustersLastChecked(mockStorage.(*storage.DBStorage), types.ClusterName(cluster), time.Now())
	}

	return mockStorage, conn, closer

}

func mustPrepareReportAndRecommendationsBenchmark(b *testing.B) (storage.Storage, *sql.DB, func()) {
	// Postgres queries are very verbose at DEBUG log level, so it's better
	// to silence them this way to make benchmark results easier to find.
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(b, false)

	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))

	_, err := conn.Exec("DROP TABLE IF EXISTS recommendations;")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("DROP TABLE IF EXISTS report;")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("DROP TABLE IF EXISTS rule_hit;")
	helpers.FailOnError(b, err)

	_, err = conn.Exec("CREATE TABLE recommendations (cluster_id VARCHAR NOT NULL, rule_fqdn VARCHAR NOT NULL, error_key VARCHAR NOT NULL, PRIMARY KEY(cluster_id, rule_fqdn, error_key));")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("CREATE TABLE report (org_id INTEGER NOT NULL, cluster VARCHAR NOT NULL UNIQUE, report VARCHAR NOT NULL, reported_at TIMESTAMP, last_checked_at TIMESTAMP, kafka_offset BIGINT NOT NULL DEFAULT 0, PRIMARY KEY(org_id, cluster));")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("CREATE TABLE rule_hit (org_id INTEGER NOT NULL, cluster_id VARCHAR NOT NULL, rule_fqdn VARCHAR NOT NULL, error_key VARCHAR NOT NULL, template_data VARCHAR NOT NULL, PRIMARY KEY(cluster_id, org_id, rule_fqdn, error_key));")
	helpers.FailOnError(b, err)

	return mockStorage, conn, closer
}

func mustCleanupAfterRecommendationsBenchmark(b *testing.B, conn *sql.DB, closer func()) {
	_, err := conn.Exec("DROP TABLE recommendations;")
	helpers.FailOnError(b, err)

	closer()
}

func mustCleanupAfterReportAndRecommendationsBenchmark(b *testing.B, conn *sql.DB, closer func()) {
	_, err := conn.Exec("DROP TABLE recommendations;")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("DROP TABLE report;")
	helpers.FailOnError(b, err)
	_, err = conn.Exec("DROP TABLE rule_hit;")
	helpers.FailOnError(b, err)

	closer()
}

// BenchmarkNewRecommendationsWithoutConflict inserts many non-conflicting
// rows into the benchmark table using storage.WriteRecommendationsForCluster.
func BenchmarkNewRecommendationsWithoutConflict(b *testing.B) {
	storage, conn, closer := mustPrepareRecommendationsBenchmark(b)

	clusterIDSet := newSet()
	for i := 0; i < 1; i++{
		id := uuid.New().String()
		if !clusterIDSet.contains(id) {
			clusterIDSet.add(id)
		}
	}

	resetTimerForBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for id := range clusterIDSet.content {
			err := storage.WriteRecommendationsForCluster(types.ClusterName(id), testdata.Report3Rules)
			helpers.FailOnError(b, err)
		}
	}

	stopBenchmarkTimer(b)
	mustCleanupAfterRecommendationsBenchmark(b, conn, closer)
}

// BenchmarkNewRecommendationsExistingClusterConflict inserts 2 conflicting
// rows into the benchmark table, forcing deletion of existing rows first
func BenchmarkNewRecommendationsExistingClusterConflict(b *testing.B) {
	mockStorage, conn, closer := mustPrepareRecommendationsBenchmark(b)

	clusterIDSet := newSet()
	for i := 0; i < 1; i++{
		id := uuid.New().String()
		if !clusterIDSet.contains(id) {
			clusterIDSet.add(id)
			storage.SetClustersLastChecked(mockStorage.(*storage.DBStorage), types.ClusterName(id), time.Now())
		}
	}
	clusterIds := make([]string, 2*len(clusterIDSet.content))
	idx := 0
	for id := range clusterIDSet.content {
		clusterIds[idx] = id
		idx++
	}
	//Now, we duplicate them
	for id := range clusterIDSet.content {
		clusterIds[idx] = id
		idx++
	}

	resetTimerForBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for _, id := range clusterIds {
			err := mockStorage.WriteRecommendationsForCluster(types.ClusterName(id), testdata.Report2Rules)
			helpers.FailOnError(b, err)
		}
	}

	stopBenchmarkTimer(b)
	mustCleanupAfterRecommendationsBenchmark(b, conn, closer)
}

// BenchmarkNewRecommendations2000initialEntries benchmarks the insertion of
// 20 entries in the recommendation table which has been previously filled
// with 2000 entries. The inserted entries do not conflict with the existing
// ones. This allows to benchmark the WriteRecommendationsForCluster method
// when the table already has a lot of rows.
func BenchmarkNewRecommendations2000initialEntries(b *testing.B) {
	mockStorage, conn, closer := mustPrepareRecommendationsBenchmarkWithEntries(b, 2000)

	clusterIDSet := newSet()
	for i := 0; i < 2; i++{
		id := uuid.New().String()
		if !clusterIDSet.contains(id) {
			clusterIDSet.add(id)
			storage.SetClustersLastChecked(mockStorage.(*storage.DBStorage), types.ClusterName(id), time.Now())
		}
	}

	resetTimerForBenchmark(b)

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for id := range clusterIDSet.content {
			err := mockStorage.WriteRecommendationsForCluster(types.ClusterName(id), Report20Rules)
			helpers.FailOnError(b, err)
		}
	}

	stopBenchmarkTimer(b)
	mustCleanupAfterRecommendationsBenchmark(b, conn, closer)
}

// BenchmarkWriteReportAndRecommendationsNoConflict inserts reports and the
// corresponding recommendations so we can compare insertion times
func BenchmarkWriteReportAndRecommendationsNoConflict(b *testing.B) {
	storage, conn, closer := mustPrepareReportAndRecommendationsBenchmark(b)

	clusterIDSet := newSet()
	for i := 0; i < 1; i++{
		id := uuid.New().String()
		if !clusterIDSet.contains(id) {
			clusterIDSet.add(id)
		}
	}

	resetTimerForBenchmark(b)

	/*
	for benchIter := 0; benchIter < b.N; benchIter++ {
		for id := range clusterIDSet.content {
			err := storage.WriteReportForCluster(types.OrgID(1), types.ClusterName(id), testdata.Report0Rules, testdata.ReportEmptyRulesParsed, time.Now(), types.KafkaOffset(0))
			helpers.FailOnError(b, err)
		}
	}

	b.ResetTimer()
	b.StartTimer()

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for id := range clusterIDSet.content {
			err := storage.WriteRecommendationsForCluster(types.ClusterName(id), testdata.Report0Rules)
			helpers.FailOnError(b, err)
		}
	}

	b.ResetTimer()
	b.StartTimer()*/

	tStart := time.Now()

	for benchIter := 0; benchIter < b.N; benchIter++ {
		for id := range clusterIDSet.content {
			err := storage.WriteReportForCluster(types.OrgID(1), types.ClusterName(id), testdata.Report2Rules, testdata.Report2RulesParsed, time.Now(), types.KafkaOffset(0))
			helpers.FailOnError(b, err)
			err = storage.WriteRecommendationsForCluster(types.ClusterName(id), Report20Rules)
			helpers.FailOnError(b, err)
		}
	}

	tEnd := time.Now()
	duration := tEnd.Sub(tStart)
	fmt.Printf("Test Duration: %v microsecs\n", duration.Microseconds())
	stopBenchmarkTimer(b)
	mustCleanupAfterReportAndRecommendationsBenchmark(b, conn, closer)
}
