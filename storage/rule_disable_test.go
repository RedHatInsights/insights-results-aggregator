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
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

// Check the method DisableRuleSystemWide.
func TestDBStorageDisableRuleSystemWide(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	// try to call the method
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, "x",
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// close storage
	closer()
}

// Check the method DisableRuleSystemWide in case of DB error.
func TestDBStorageDisableRuleSystemWideOnDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediately
	closer()

	// try to call the method
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, "x",
	)

	// we expect the error to happen
	assert.EqualError(t, err, "sql: database is closed")
}

// Check the method EnableRuleSystemWide.
func TestDBStorageEnableRuleSystemWide(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	// try to call the method
	err := mockStorage.EnableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// close storage
	closer()
}

// Check the method DisableRuleSystemWide to check ON CONFLICT
// This shouldn't happen in real enviornment because
// Re-enabling/Updating justification/Getting from the rule_disable table is used
func TestDBStorageEnableRuleSystemWideDifferentUser(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	user1Justification := "first user reason"
	user2Justification := "second user reason"

	// try to call the method
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, user1Justification,
	)
	// we expect no error
	helpers.FailOnError(t, err)

	// read the rule
	r, found, err := mockStorage.ReadDisabledRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)
	helpers.FailOnError(t, err)

	// check returned values
	assert.True(t, found, "Rule should be found")
	assert.Equal(t, user1Justification, r.Justification, "Justification must be correct")

	// try to call the method with same org, but different user
	err = mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, user2Justification,
	)
	// we expect no error
	helpers.FailOnError(t, err)

	// read the rule
	r, found, err = mockStorage.ReadDisabledRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)
	helpers.FailOnError(t, err)

	// check returned values
	assert.True(t, found, "Rule should be found")
	assert.Equal(t, user2Justification, r.Justification, "Justification must be correct")

	// close storage
	closer()
}

// Check the method EnableRuleSystemWide in case of DB error.
func TestDBStorageEnableRuleSystemWideOnDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediately
	closer()

	// try to call the method
	err := mockStorage.EnableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)

	// we expect the error to happen
	assert.EqualError(t, err, "sql: database is closed")
}

// Check the method UpdateDisabledRuleJustification.
func TestDBStorageUpdateDisabledRuleJustifiction(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	// try to call the method
	err := mockStorage.UpdateDisabledRuleJustification(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, "z",
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// close storage
	closer()
}

// Check the method UpdateDisabledRuleJustification in case of DB error.
func TestDBStorageUpdateDisabledRuleJustification(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediately
	closer()

	// try to call the method
	err := mockStorage.UpdateDisabledRuleJustification(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, "y",
	)

	// we expect the error to happen
	assert.EqualError(t, err, "sql: database is closed")
}

// Check the method ReadDisabledRule.
func TestDBStorageReadDisabledRuleNoRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	// try to call the method
	_, found, err := mockStorage.ReadDisabledRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// check returned values
	assert.False(t, found, "Rule should not be found")

	// close storage
	closer()
}

// Check the method ReadDisabledRule.
func TestDBStorageReadDisabledRuleOneRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	const justification = "JUSTIFICATION"

	// fill-in database
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, justification,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// try to call the method
	r, found, err := mockStorage.ReadDisabledRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// check returned values
	assert.True(t, found, "Rule should be found")
	assert.Equal(t, justification, r.Justification, "Rule should be found")

	// close storage
	closer()
}

// Check the method ReadDisabledRule in case of DB error.
func TestDBStorageReadDisabledRuleOnRBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediately
	closer()

	// try to call the method
	_, _, err := mockStorage.ReadDisabledRule(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1,
	)

	// we expect the error to happen
	assert.EqualError(t, err, "sql: database is closed")
}

// Check the method ListOfSystemWideDisabledRules.
func TestDBStorageListOfSystemWideDisabledRulesNoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	// try to call the method
	list, err := mockStorage.ListOfSystemWideDisabledRules(testdata.OrgID)

	// we expect no error
	helpers.FailOnError(t, err)

	// check the list
	assert.Equal(t, 0, len(list), "List of disabled rules should be empty")

	// close storage
	closer()
}

// Check the method ListOfSystemWideDisabledRules.
func TestDBStorageListOfSystemWideDisabledRulesOneRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	const justification = "JUSTIFICATION"

	// fill-in database
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, justification,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// try to call the method
	list, err := mockStorage.ListOfSystemWideDisabledRules(testdata.OrgID)

	// we expect no error
	helpers.FailOnError(t, err)

	// check the list
	assert.Equal(t, 1, len(list), "List of disabled rules should contain just one item")

	// check the item in a list
	assert.Equal(t, justification, list[0].Justification, "Rule should be found")

	// close storage
	closer()
}

// Check the method ListOfSystemWideDisabledRules.
func TestDBStorageListOfSystemWideDisabledRulesTwoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	const justification = "JUSTIFICATION"

	// fill-in database
	err := mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule1ID, testdata.ErrorKey1, justification,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// disable second rule
	err = mockStorage.DisableRuleSystemWide(
		testdata.OrgID, testdata.Rule2ID, testdata.ErrorKey2, justification,
	)

	// we expect no error
	helpers.FailOnError(t, err)

	// try to call the method
	list, err := mockStorage.ListOfSystemWideDisabledRules(testdata.OrgID)

	// we expect no error
	helpers.FailOnError(t, err)

	// check the list
	assert.Equal(t, 2, len(list), "List of disabled rules should contain two items")

	// check items in a list
	assert.Equal(t, justification, list[0].Justification, "Rule should be found")
	assert.Equal(t, justification, list[1].Justification, "Rule should be found")

	// close storage
	closer()
}

// Check the method ListOfSystemWideDisabledRules in case of DB error.
func TestDBStorageListOfSystemWideDisabledRulesDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// close storage immediately
	closer()

	// try to call the method
	_, err := mockStorage.ListOfSystemWideDisabledRules(testdata.OrgID)

	// we expect the error to happen
	assert.EqualError(t, err, "sql: database is closed")
}
