// Copyright 2020 Red Hat, Inc
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

package helpers

import (
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// APIRequest is a type for APIRequest
type APIRequest = helpers.APIRequest

// APIResponse is a type for APIResponse
type APIResponse = helpers.APIResponse

var (
	// ExecuteRequest executes request
	ExecuteRequest = helpers.ExecuteRequest
	// CheckResponseBodyJSON checks response body
	CheckResponseBodyJSON = helpers.CheckResponseBodyJSON
	// AssertReportResponsesEqual fails if report responses aren't equal
	AssertReportResponsesEqual = helpers.AssertReportResponsesEqual
)

// DefaultServerConfig is a default config used by AssertAPIRequest
var DefaultServerConfig = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	Auth:        false,
}

// AssertAPIRequest creates new server with provided mockStorage
// (which you can keep nil so it will be created automatically)
// and provided serverConfig(you can leave it empty to use the default one)
// sends api request and checks api response (see docs for APIRequest and APIResponse)
func AssertAPIRequest(
	t testing.TB,
	mockStorage storage.Storage,
	serverConfig *server.Configuration,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	if mockStorage == nil {
		var closer func()
		mockStorage, closer = MustGetMockStorage(t, true)
		defer closer()
	}
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}

	testServer := server.New(*serverConfig, mockStorage)

	helpers.AssertAPIRequest(t, testServer, serverConfig.APIPrefix, request, expectedResponse)
}
