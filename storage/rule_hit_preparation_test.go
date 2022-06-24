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

package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestDBStorage_getRuleHitInsertStatement(t *testing.T) {
	fakeStorage := storage.NewFromConnection(nil, -1)
	r := fakeStorage.GetRuleHitInsertStatement(testdata.Report3RulesParsed)

	// 5*3 placeholders expected
	const expected = "INSERT INTO rule_hit(org_id, cluster_id, rule_fqdn, error_key, template_data, created_at) VALUES ($1,$2,$3,$4,$5,$6),($7,$8,$9,$10,$11,$12),($13,$14,$15,$16,$17,$18)"
	assert.Equal(t, r, expected)
}

func TestDBStorage_valuesForRuleHitsInsert(t *testing.T) {
	v := storage.ValuesForRuleHitsInsert(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3RulesParsed,
		make(map[string]types.Timestamp),
	)

	// 6 values for 3 rule hits expected
	assert.Len(t, v, 6*3)

	// just elementary tests
	for i := 0; i < 6*3; i += 6 {
		assert.Equal(t, v[i], testdata.OrgID)
		assert.Equal(t, v[i+1], testdata.ClusterName)
	}
}
