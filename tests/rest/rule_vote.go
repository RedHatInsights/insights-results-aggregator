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

package tests

import (
	"fmt"
	"strings"

	"github.com/verdverm/frisby"
)

const (
	improperClusterID = "000000000000000000000000000000000000"
	knownCluster      = "00000000-0000-0000-0000-000000000000"

	anyRule   = "0" // we don't care
	knownRule = "foo"
)

var knownClustersForOrganization1 []string = []string{
	"00000000-0000-0000-0000-000000000000",
	"00000000-0000-0000-ffff-000000000000",
	"00000000-0000-0000-0000-ffffffffffff",
}

var unknownClusters []string = []string{
	"abcdef00-0000-0000-0000-000000000000",
	"abcdef00-0000-0000-1111-000000000000",
	"abcdef00-0000-0000-ffff-000000000000",
}

var improperClusterIDs []string = []string{
	"xyz",                                   // definitely wrong
	"00000000-0000-0000-0000-00000000000",   // shorter by one character
	"0000000-0000-0000-0000-000000000000",   // shorter by one character
	"00000000-000-0000-0000-000000000000",   // shorter by one character
	"00000000-0000-000-0000-000000000000",   // shorter by one character
	"00000000-0000-0000-000-000000000000",   // shorter by one character
	"00000000-0000-0000-0000-fffffffffffff", // longer by one character
	"00000000f-0000-0000-0000-ffffffffffff", // longer by one character
	"00000000-0000f-0000-0000-ffffffffffff", // longer by one character
	"00000000-0000-0000f-0000-ffffffffffff", // longer by one character
	"00000000-0000-0000-0000f-ffffffffffff", // longer by one character
}

var knownRules []string = []string{
	"foo",
	"bar",
	"xyzzy",
}

var unknownRules []string = []string{
	"rule1",
	"rule2",
	"rule3",
}

func constructURLVoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/like", apiURL, clusterID, ruleID)
}

func constructURLUnvoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/dislike", apiURL, clusterID, ruleID)
}

func constructURLResetVoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/reset_vote", apiURL, clusterID, ruleID)
}

func checkOkStatus(url string, message string) {
	f := frisby.Create(message).Put(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status != "ok" {
		f.AddError(fmt.Sprintf("Expected ok status, but got '%s' instead", statusResponse.Status))
	}
}

func checkInvalidUUIDFormat(url string, message string) {
	f := frisby.Create(message).Put(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status == "ok" {
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead", statusResponse.Status))
	}
	if !strings.Contains(statusResponse.Status, "Error: 'invalid UUID ") {
		f.AddError(fmt.Sprintf("Unexpected error reported: %v", statusResponse.Status))
	}

	f.PrintReport()
}

func checkItemNotFound(url string, message string) {
	f := frisby.Create(message).Put(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status == "ok" {
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead", statusResponse.Status))
	}
	if !strings.Contains(statusResponse.Status, "was not found in the storage") {
		f.AddError(fmt.Sprintf("Unexpected error reported: %v", statusResponse.Status))
	}

	f.PrintReport()
}

// reproducerForIssue385 checks whether the issue https://github.com/RedHatInsights/insights-results-aggregator/issues/385 has been fixed
func reproducerForIssue385() {
	url := constructURLVoteForRule("000000000000000000000000000000000000", 0)
	f := frisby.Create("Reproducer for issue #385 (https://github.com/RedHatInsights/insights-results-aggregator/issues/385)").Put(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status == "ok" {
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead", statusResponse.Status))
	}
	if !strings.Contains(statusResponse.Status, "Error: 'invalid UUID format'") {
		f.AddError("Unexpected error reported")
	}

	f.PrintReport()
}
