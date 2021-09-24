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

package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestHTTPServer_RateOnRule(t *testing.T) {
	ratingBody := `{"rule": "rule_module|error_key", "rating": -1}`
	expectedResponseBody := fmt.Sprintf(`{"ratings":%s, "status":"ok"}`, ratingBody)

	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.Rating,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.UserID},
		Body:         ratingBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       expectedResponseBody,
	})
}

func TestHTTPServer_RateOnRuleNoContent(t *testing.T) {
	expectedError := `{"status": "client didn't provide request body"}`

	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.Rating,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       expectedError,
	})
}

func TestHTTPServer_RateOnRuleBadContent(t *testing.T) {
	ratingBody := `{"rule": "rule_module", "rating": -1}`

	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.Rating,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.UserID},
		Body:         ratingBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}
