// Copyright 2023 Red Hat, Inc
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

import (
	"database/sql"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// NoopDVOStorage represents a storage which does nothing (for benchmarking without a storage)
type NoopDVOStorage struct{}

// Init noop
func (*NoopDVOStorage) Init() error {
	return nil
}

// Close noop
func (*NoopDVOStorage) Close() error {
	return nil
}

// GetMigrations noop
func (*NoopDVOStorage) GetMigrations() []migration.Migration {
	return nil
}

// GetDBDriverType noop
func (*NoopDVOStorage) GetDBDriverType() types.DBDriver {
	return types.DBDriver(-1)
}

// GetConnection noop
func (*NoopDVOStorage) GetConnection() *sql.DB {
	return nil
}

// GetMaxVersion noop
func (*NoopDVOStorage) GetMaxVersion() migration.Version {
	return migration.Version(0)
}

// GetDBSchema noop
func (*NoopDVOStorage) GetDBSchema() migration.Schema {
	return migration.Schema("")
}

// MigrateToLatest noop
func (*NoopDVOStorage) MigrateToLatest() error {
	return nil
}

// ReportsCount noop
func (*NoopDVOStorage) ReportsCount() (int, error) {
	return 0, nil
}

// WriteReportForCluster noop
func (*NoopDVOStorage) WriteReportForCluster(
	types.OrgID, types.ClusterName, types.ClusterReport, []types.WorkloadRecommendation, time.Time, time.Time, time.Time,
	types.RequestID,
) error {
	return nil
}

// ReadWorkloadsForOrganization noop
func (*NoopDVOStorage) ReadWorkloadsForOrganization(types.OrgID, map[types.ClusterName]struct{}, bool) ([]types.WorkloadsForNamespace, error) {
	return nil, nil
}

// ReadWorkloadsForClusterAndNamespace noop
func (*NoopDVOStorage) ReadWorkloadsForClusterAndNamespace(
	types.OrgID,
	types.ClusterName,
	string,
) (types.DVOReport, error) {
	return types.DVOReport{}, nil
}

// DeleteReportsForOrg noop
func (*NoopDVOStorage) DeleteReportsForOrg(types.OrgID) error {
	return nil
}
