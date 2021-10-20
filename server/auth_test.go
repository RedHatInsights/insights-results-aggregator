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

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

var configAuth = server.Configuration{
	Address:                      ":8080",
	APIPrefix:                    "/api/test/",
	Debug:                        false,
	Auth:                         true,
	AuthType:                     "xrh",
	MaximumFeedbackMessageLength: 255,
}

var configAuth2 = server.Configuration{
	Address:                      ":8080",
	APIPrefix:                    "/api/test/",
	Debug:                        true,
	Auth:                         true,
	AuthType:                     "jwt",
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

// TestJWTToken checks authorization through Authorization header
func TestJWTToken(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth2, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.ClustersForOrganizationEndpoint,
		EndpointArgs:       []interface{}{1234},
		AuthorizationToken: "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50X251bWJlciI6IjUyMTM0NzYiLCJvcmdfaWQiOiIxMjM0In0.Y9nNaZXbMEO6nz2EHNaCvHxPM0IaeT7GGR-T8u8h_nr_2b5dYsCQiZGzzkBupRJruHy9K6acgJ08JN2Q28eOAEVk_ZD2EqO43rSOS6oe8uZmVo-nCecdqovHa9PqW8RcZMMxVfGXednw82kKI8j1aT_nbJ1j9JZt3hnHM4wtqydelMij7zKyZLHTWFeZbDDCuEIkeWA6AdIBCMdywdFTSTsccVcxT2rgv4mKpxY1Fn6Vu_Xo27noZW88QhPTHbzM38l9lknGrvJVggrzMTABqWEXNVHbph0lXjPWsP7pe6v5DalYEBN2r3a16A6s3jPfI86cRC6_oeXotlW6je0iKQ",
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"clusters":[],"status":"ok"}`,
	})
}

// TestJWTToken checks authorization through Authorization header
func TestJWTTokenMalformed(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth2, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1234},
		// do not pass token itself
		AuthorizationToken: "Bearer",
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status":"Invalid/Malformed auth token"}`,
	})
}

// TestJWTTokenMalformedJSON checks authorization through Authorization header with bad JSON in JWT token
func TestJWTTokenMalformedJSON(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &configAuth2, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1234},
		// pass bad json
		AuthorizationToken: "Bearer bm90LWpzb24K.bm90LWpzb24K.bm90LWpzb24K",
	}, &helpers.APIResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       `{"status":"Malformed authentication token"}`,
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
	orgIDInXRH := 1234
	body := fmt.Sprintf(`{"status":"you have no permissions to get or change info about the organization with ID %v; you can access info about organization with ID %v"}`, providedOrgID, orgIDInXRH)
	helpers.AssertAPIRequest(t, nil, &configAuth, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{providedOrgID},
		XRHIdentity:  "eyJpZGVudGl0eSI6IHsiaW50ZXJuYWwiOiB7Im9yZ19pZCI6ICIxMjM0In19fQo=",
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
		Body:       body,
	})
}
