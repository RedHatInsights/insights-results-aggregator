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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	apiURL = "http://localhost:8080/api/v1"
)

// TestConsumingMessage writes message to kafka and expects results on rest api
func TestConsumingMessage(t *testing.T) {
	brokerCfg := helpers.GetTestBrokerCfg(t, testTopic)

	_, _, err := producer.ProduceMessage(brokerCfg, testMessage)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(fmt.Sprintf("%v/report/%v/%v", apiURL, testOrgID, testClusterName))
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, `{"report":"{}","status":"ok"}`, strings.TrimSpace(string(respBytes)))
}
