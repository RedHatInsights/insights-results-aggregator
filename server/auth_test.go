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

package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	ctypes "github.com/RedHatInsights/insights-results-types"
	types "github.com/RedHatInsights/insights-results-types"
)

var configAuth = server.Configuration{
	Address:                      ":8080",
	APIPrefix:                    "/api/test/",
	Debug:                        false,
	Auth:                         true,
	AuthType:                     "xrh",
	MaximumFeedbackMessageLength: 255,
}

// TestMissingAuthToken checks how the missing auth. token header (expected in HTTP request) is handled
func TestMissingAuthToken(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1},
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status": "Missing auth token"}`,
	})
}

// TestMalformedAuthToken checks whether string that is not BASE64-encoded can't be decoded
func TestMalformedAuthToken(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1},
		XRHIdentity:  "!",
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status": "Malformed authentication token"}`,
	})
}

// TestInvalidAuthToken checks whether token header that is not properly encoded is handled correctly
func TestInvalidAuthToken(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1},
		XRHIdentity:  "123456qwerty",
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status": "Malformed authentication token"}`,
	})
}

// TestInvalidAuthToken checks whether token header that does not contain correct JSON
// (encoded by BASE64) is handled correctly
func TestInvalidJsonAuthToken(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1},
		XRHIdentity:  "aW52YWxpZCBqc29uCg==",
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status": "Malformed authentication token"}`,
	})
}

// TestBadOrganizationID checks if organization ID is checked properly
func TestBadOrganizationID(t *testing.T) {
	providedOrgID := 12345
	orgIDInXRH := 2
	body := fmt.Sprintf(`{"status":"you have no permissions to get or change info about the organization with ID %v; you can access info about organization with ID %v"}`, providedOrgID, orgIDInXRH)
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{providedOrgID},
		XRHIdentity: helpers.MakeXRHTokenString(t, &types.Token{
			Identity: ctypes.Identity{
				AccountNumber: testdata.UserID,
				OrgID:         ctypes.OrgID(orgIDInXRH),
				User: ctypes.User{
					UserID: testdata.UserID,
				},
			},
		}),
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
		Body:       body,
	})
}
