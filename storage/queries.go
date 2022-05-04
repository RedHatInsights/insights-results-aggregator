// Copyright 2022 Red Hat, Inc
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

package storage

import "github.com/RedHatInsights/insights-results-aggregator/types"

func (storage DBStorage) getReportUpsertQuery() string {
	if storage.dbDriverType == types.DBDriverSQLite3 {
		return `
			INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset, gathered_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
	}

	return `
		INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset, gathered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (cluster)
		DO UPDATE SET org_id = $1, report = $3, reported_at = $4, last_checked_at = $5, kafka_offset = $6, gathered_at = $7
	`
}

func (storage DBStorage) getReportInfoUpsertQuery() string {
	if storage.dbDriverType == types.DBDriverSQLite3 {
		return `
			INSERT OR REPLACE INTO report_info(org_id, cluster_id, version_info)
			VALUES ($1, $2, $3)
		`
	}

	return `
		INSERT INTO report_info(org_id, cluster_id, version_info)
		VALUES ($1, $2, $3)
		ON CONFLICT (cluster_id)
		DO UPDATE SET org_id = $1, version_info = $3
	`
}
