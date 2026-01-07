// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_data "github.com/RedHatInsights/insights-results-aggregator/tests/data"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Don't decrease code coverage by non-functional and not covered code.

// TestDVONoopStorageEmptyMethods1 calls empty methods that just needs to be
// defined in order for NoopStorage to satisfy Storage interface.
func TestDVONoopStorageEmptyMethods(_ *testing.T) {
	noopStorage := storage.NoopDVOStorage{}
	orgID := types.OrgID(1)

	_ = noopStorage.Init()
	_ = noopStorage.Close()
	_, _ = noopStorage.ReportsCount()
	_ = noopStorage.DeleteReportsForOrg(orgID)
	_ = noopStorage.MigrateToLatest()
	_ = noopStorage.GetConnection()
	_ = noopStorage.GetDBDriverType()

	_ = noopStorage.WriteReportForCluster(0, "", "", ira_data.ValidDVORecommendation, time.Now(), time.Now(), time.Now(), "")
}
