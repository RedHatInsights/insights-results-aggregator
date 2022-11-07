// Copyright 2021 Red Hat, Inc
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

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestDBStorage_RateOnRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	err := mockStorage.RateOnRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, types.UserVoteLike,
	)
	helpers.FailOnError(t, err)
}

func TestDBStorage_GetRuleRating(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	err := mockStorage.RateOnRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, types.UserVoteLike,
	)
	helpers.FailOnError(t, err)

	rating, err := mockStorage.GetRuleRating(testdata.OrgID, types.RuleSelector(testdata.Rule1CompositeID))
	helpers.FailOnError(t, err)

	assert.Equal(t, types.UserVoteLike, rating.Rating)
	assert.Equal(t, string(testdata.Rule1CompositeID), rating.Rule)
}

func TestDBStorage_GetRuleRating_NotFound(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	err := mockStorage.RateOnRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, types.UserVoteLike,
	)
	helpers.FailOnError(t, err)

	_, err = mockStorage.GetRuleRating(testdata.OrgID, types.RuleSelector(testdata.Rule2CompositeID))
	assert.NotNil(t, err)
}
