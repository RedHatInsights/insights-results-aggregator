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
	"net/http"
	"testing"
	"time"

	"github.com/verdverm/frisby"

	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	testTopic       = "ccx.ocp.results"
	testOrgID       = "55"
	testClusterName = "d88d9108-2fde-445c-9fde-28de92f09443"
	testMessage     = `{
		"OrgID": ` + testOrgID + `,
		"ClusterName": "` + testClusterName + `",
		"Report": {},
		"LastChecked": "2020-01-23T16:15:59.478901889Z"
	}`
	apiURL                           = "http://localhost:8080/api/v1"
	testTimeout                      = 10 * time.Second
	testProcessingMessageCheckPeriod = 1 * time.Second
)

// TestConsumingMessage writes message to kafka and expects results on rest api
func TestConsumingMessage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		brokerCfg := helpers.GetTestBrokerCfg(t, testTopic)

		_, _, err := producer.ProduceMessage(brokerCfg, testMessage)
		if err != nil {
			t.Fatal(err)
		}

		for {
			// wait till message is actually processed
			resp, err := http.Get(apiURL + fmt.Sprintf("/report/%v/%v", testOrgID, testClusterName))
			if err != nil {
				t.Fatal(err)
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				break
			} else if resp.StatusCode != http.StatusNotFound {
				t.Fatal(fmt.Errorf("unexpected response status %v", resp.Status))
			}

			time.Sleep(testProcessingMessageCheckPeriod)
		}

		frisby.Create("test consuming message /report/org_id/cluster_name").
			Get(apiURL+fmt.Sprintf("/report/%v/%v", testOrgID, testClusterName)).
			Send().
			ExpectStatus(http.StatusOK).
			ExpectJson("report", "{}").
			ExpectJson("status", "ok")

		frisby.Create("test consuming message /organization").
			Get(apiURL + "/organization").
			Send().
			ExpectStatus(http.StatusOK).
			Expect(helpers.FrisbyExpectItemInArray("organizations", 55))

		frisby.Create("test consuming message /cluster/org_id").
			Get(apiURL + fmt.Sprintf("/cluster/%v", testOrgID)).
			Send().
			ExpectStatus(http.StatusOK).
			Expect(helpers.FrisbyExpectItemInArray("clusters", "d88d9108-2fde-445c-9fde-28de92f09443"))

		for key, err := range frisby.Global.Errors() {
			t.Fatal(key, err)
		}
	}, testTimeout)
}
