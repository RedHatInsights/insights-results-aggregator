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

func constructURLVoteForRule(clusterID string, ruleID int) string {
	return fmt.Sprintf("%sclusters/%s/rules/%d/like", apiURL, clusterID, ruleID)
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
