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
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestReadReportMetainfoForClusterNonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{"non-int", testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'org_id' with value 'non-int'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportMetainfoForClusterNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"Error during parsing param 'org_id' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportMetainfoForClusterBadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestReadNonExistingReportMetainfo(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Body: fmt.Sprintf(
			`{"status":"Item with ID %v/%v was not found in the storage"}`, testdata.OrgID, testdata.ClusterName,
		),
	})
}

func TestReadExistingEmptyReportMetainfo(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, testdata.ReportEmptyRulesParsed, testdata.LastCheckedAt, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"metainfo": {
				"count": -1,
				"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
				"stored_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
			}
		}`,
	})
}

func TestReadReportMetainfoDBError(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Internal Server Error"}`,
	})
}

func TestReadReportMetainfo(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.Report3RulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportMetainfoEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"metainfo": {
				"count": 3,
				"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
				"stored_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
			}
		}`,
	})
}
